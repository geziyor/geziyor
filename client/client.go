package client

import (
	"context"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrNoCookieJar is the error type for missing cookie jar
	ErrNoCookieJar = errors.New("cookie jar is not available")
	ErrWrongStatus = errors.New("wrong response status code")
)

// Client is a small wrapper around *http.Client to provide new methods.
type Client struct {
	*http.Client
	maxBodySize           int64
	charsetDetectDisabled bool
	retryTimes            int
	retryHTTPCodes        []int
}

const (
	DefaultUserAgent        = "Geziyor 1.0"
	DefaultMaxBody    int64 = 1024 * 1024 * 1024 // 1GB
	DefaultRetryTimes       = 2
)

var (
	DefaultRetryHTTPCodes = []int{500, 502, 503, 504, 522, 524, 408}
)

// NewClient creates http.Client with modified values for typical web scraper
func NewClient(maxBodySize int64, charsetDetectDisabled bool, retryTimes int, retryHTTPCodes []int) *Client {
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          0,    // Default: 100
			MaxIdleConnsPerHost:   1000, // Default: 2
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: time.Second * 180, // Google's timeout
	}

	client := Client{
		Client:                httpClient,
		maxBodySize:           maxBodySize,
		charsetDetectDisabled: charsetDetectDisabled,
		retryTimes:            retryTimes,
		retryHTTPCodes:        retryHTTPCodes,
	}

	return &client
}

// DoRequest selects appropriate request handler, client or Chrome
func (c *Client) DoRequest(req *Request) (resp *Response, err error) {
	if req.Rendered {
		resp, err = c.DoRequestChrome(req)
	}
	resp, err = c.DoRequestClient(req)

	// Retry on Error
	if err != nil {
		if req.retryCounter < c.retryTimes {
			req.retryCounter++
			log.Println("Retrying:", req.URL.String())
			return c.DoRequest(req)
		}
		return resp, errors.Wrap(err, "Response error")
	}

	// Retry on http status codes
	for _, statusCode := range c.retryHTTPCodes {
		if req.retryCounter < c.retryTimes {
			if resp.StatusCode == statusCode {
				req.retryCounter++
				log.Println("Retrying:", req.URL.String(), resp.StatusCode)
				return c.DoRequest(req)
			}
		} else {
			return nil, ErrWrongStatus
		}
	}

	return
}

// DoRequestClient is a simple wrapper to read response according to options.
func (c *Client) DoRequestClient(req *Request) (*Response, error) {
	// Do request
	resp, err := c.Do(req.Request)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	// Limit response body reading
	bodyReader := io.LimitReader(resp.Body, c.maxBodySize)

	// Decode response
	if resp.Request.Method != "HEAD" && resp.ContentLength > 0 {
		if req.Encoding != "" {
			if enc, _ := charset.Lookup(req.Encoding); enc != nil {
				bodyReader = transform.NewReader(bodyReader, enc.NewDecoder())
			}
		} else {
			if !c.charsetDetectDisabled {
				bodyReader, err = charset.NewReader(bodyReader, req.Header.Get("Content-Type"))
				if err != nil {
					return nil, errors.Wrap(err, "Reading determined encoding error")
				}
			}
		}
	}

	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return nil, errors.Wrap(err, "Reading body error")
	}

	response := Response{
		Response: resp,
		Body:     body,
		Request:  req,
	}

	return &response, nil
}

// DoRequestChrome opens up a new chrome instance and makes request
func (c *Client) DoRequestChrome(req *Request) (*Response, error) {
	var body string
	var res *network.Response

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	if err := chromedp.Run(ctx,
		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers(ConvertHeaderToMap(req.Header))),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var reqID network.RequestID
			chromedp.ListenTarget(ctx, func(ev interface{}) {
				switch ev.(type) {
				case *network.EventRequestWillBeSent:
					reqEvent := ev.(*network.EventRequestWillBeSent)
					if _, exists := reqEvent.Request.Headers["Referer"]; !exists {
						if strings.HasPrefix(reqEvent.Request.URL, "http") {
							reqID = reqEvent.RequestID
						}
					}
				case *network.EventResponseReceived:
					if resEvent := ev.(*network.EventResponseReceived); resEvent.RequestID == reqID {
						res = resEvent.Response
					}
				}
			})
			return nil
		}),
		chromedp.Navigate(req.URL.String()),
		chromedp.WaitReady(":root"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			node, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}
			body, err = dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
			return err
		}),
	); err != nil {
		return nil, errors.Wrap(err, "Request getting rendered error")
	}

	// Update changed data
	req.Header = ConvertMapToHeader(res.RequestHeaders)
	req.URL, _ = url.Parse(res.URL)

	response := Response{
		Response: &http.Response{
			Request:    req.Request,
			StatusCode: int(res.Status),
			Proto:      res.Protocol,
			Header:     ConvertMapToHeader(res.Headers),
		},
		Body:    []byte(body),
		Request: req,
	}

	return &response, nil
}

// SetCookies handles the receipt of the cookies in a reply for the given URL
func (c *Client) SetCookies(URL string, cookies []*http.Cookie) error {
	if c.Jar == nil {
		return ErrNoCookieJar
	}
	u, err := url.Parse(URL)
	if err != nil {
		return err
	}
	c.Jar.SetCookies(u, cookies)
	return nil
}

// Cookies returns the cookies to send in a request for the given URL.
func (c *Client) Cookies(URL string) []*http.Cookie {
	if c.Jar == nil {
		return nil
	}
	parsedURL, err := url.Parse(URL)
	if err != nil {
		return nil
	}
	return c.Jar.Cookies(parsedURL)
}

// SetDefaultHeader sets header if not exists before
func SetDefaultHeader(header http.Header, key string, value string) http.Header {
	if header.Get(key) == "" {
		header.Set(key, value)
	}
	return header
}

// ConvertHeaderToMap converts http.Header to map[string]interface{}
func ConvertHeaderToMap(header http.Header) map[string]interface{} {
	m := make(map[string]interface{})
	for key, values := range header {
		for _, value := range values {
			m[key] = value
		}
	}
	return m
}

// ConvertMapToHeader converts map[string]interface{} to http.Header
func ConvertMapToHeader(m map[string]interface{}) http.Header {
	header := http.Header{}
	for k, v := range m {
		header.Set(k, v.(string))
	}
	return header
}

// NewRedirectionHandler returns maximum allowed redirection function with provided maxRedirect
func NewRedirectionHandler(maxRedirect int) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirect {
			return errors.Errorf("stopped after %d redirects", maxRedirect)
		}
		return nil
	}
}
