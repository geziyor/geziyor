package client

import (
	"context"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
	"golang.org/x/net/html/charset"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

var (
	// ErrNoCookieJar is the error type for missing cookie jar
	ErrNoCookieJar = errors.New("cookie jar is not available")
)

// Client is a small wrapper around *http.Client to provide new methods.
type Client struct {
	*http.Client
}

// NewClient creates http.Client with modified values for typical web scraper
func NewClient() *Client {
	client := &http.Client{
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
	return &Client{Client: client}
}

// DoRequest selects appropriate request handler, client or Chrome
func (c *Client) DoRequest(req *Request, maxBodySize int64, charsetDetectDisabled bool) (*Response, error) {
	if !req.Rendered {
		return c.DoRequestClient(req, maxBodySize, charsetDetectDisabled)
	} else {
		return c.DoRequestChrome(req)
	}
}

// DoRequestClient is a simple wrapper to read response according to options.
func (c *Client) DoRequestClient(req *Request, maxBodySize int64, charsetDetectDisabled bool) (*Response, error) {
	// Do request
	resp, err := c.Do(req.Request)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, errors.Wrap(err, "Response error")
	}

	// Limit response body reading
	bodyReader := io.LimitReader(resp.Body, maxBodySize)

	// Start reading body and determine encoding
	if !charsetDetectDisabled && resp.Request.Method != "HEAD" {
		bodyReader, err = charset.NewReader(bodyReader, resp.Header.Get("Content-Type"))
		if err != nil {
			return nil, errors.Wrap(err, "Determine encoding error")
		}
	}

	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return nil, errors.Wrap(err, "Reading body error")
	}

	response := Response{
		Response: resp,
		Body:     body,
		Meta:     req.Meta,
		Request:  req,
	}

	return &response, nil
}

// DoRequestChrome opens up a new chrome instance and makes request
func (c *Client) DoRequestChrome(req *Request) (*Response, error) {
	var body string
	var reqID network.RequestID
	var res *network.Response

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	if err := chromedp.Run(ctx,
		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers(ConvertHeaderToMap(req.Header))),
		chromedp.ActionFunc(func(ctx context.Context) error {
			chromedp.ListenTarget(ctx, func(ev interface{}) {
				switch ev.(type) {
				case *network.EventRequestWillBeSent:
					reqEvent := ev.(*network.EventRequestWillBeSent)
					if _, exists := reqEvent.Request.Headers["Referer"]; !exists {
						reqID = reqEvent.RequestID
					}
					//if reqEvent := ev.(*network.EventRequestWillBeSent); reqEvent.Request.URL == req.URL.String() {
					//	reqID = reqEvent.RequestID
					//}
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

	// Set new URL in case of redirection
	req.URL, _ = url.Parse(res.URL)

	response := Response{
		Response: &http.Response{
			Request:    req.Request,
			StatusCode: int(res.Status),
			Header:     ConvertMapToHeader(res.Headers),
		},
		Body:    []byte(body),
		Meta:    req.Meta,
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
