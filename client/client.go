package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/geziyor/geziyor/internal"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

var (
	// ErrNoCookieJar is the error type for missing cookie jar
	ErrNoCookieJar = errors.New("cookie jar is not available")
)

// Client is a small wrapper around *http.Client to provide new methods.
type Client struct {
	*http.Client
	opt *Options
}

// Options is custom http.client options
type Options struct {
	MaxBodySize           int64
	CharsetDetectDisabled bool
	RetryTimes            int
	RetryHTTPCodes        []int
	RemoteAllocatorURL    string
	AllocatorOptions      []chromedp.ExecAllocatorOption
	ProxyFunc             func(*http.Request) (*url.URL, error)
	// Changing this will override the existing default PreActions for Rendered requests.
	// Geziyor Response will be nearly empty. Because we have no way to extract response without default pre actions.
	// So, if you set this, you should handle all navigation, header setting, and response handling yourself.
	// See defaultPreActions variable for the existing defaults.
	PreActions []chromedp.Action
}

// Default values for client
const (
	DefaultUserAgent        = "Geziyor 1.0"
	DefaultMaxBody    int64 = 1024 * 1024 * 1024 // 1GB
	DefaultRetryTimes       = 2
)

var (
	DefaultRetryHTTPCodes = []int{500, 502, 503, 504, 522, 524, 408}
)

// NewClient creates http.Client with modified values for typical web scraper
func NewClient(opt *Options) *Client {
	// Default proxy function is http.ProxyFunction
	var proxyFunction = http.ProxyFromEnvironment
	if opt.ProxyFunc != nil {
		proxyFunction = opt.ProxyFunc
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: proxyFunction,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          0,    // Default: 100
			MaxIdleConnsPerHost:   1000, // Default: 2
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: time.Second * 180, // Google's timeout
	}

	client := Client{
		Client: httpClient,
		opt:    opt,
	}

	return &client
}

// DoRequest selects appropriate request handler, client or Chrome
func (c *Client) DoRequest(req *Request) (resp *Response, err error) {
	if req.Rendered {
		resp, err = c.doRequestChrome(req)
	} else {
		resp, err = c.doRequestClient(req)
	}

	// Retry on Error
	if err != nil {
		if req.retryCounter < c.opt.RetryTimes {
			req.retryCounter++
			internal.Logger.Println("Retrying:", req.URL.String())
			return c.DoRequest(req)
		}
		return resp, err
	}

	// Retry on http status codes
	if internal.ContainsInt(c.opt.RetryHTTPCodes, resp.StatusCode) {
		if req.retryCounter < c.opt.RetryTimes {
			req.retryCounter++
			internal.Logger.Println("Retrying:", req.URL.String(), resp.StatusCode)
			return c.DoRequest(req)
		}
	}

	return resp, err
}

// doRequestClient is a simple wrapper to read response according to options.
func (c *Client) doRequestClient(req *Request) (*Response, error) {
	// Do request
	resp, err := c.Do(req.Request)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("response: %w", err)
	}

	// Limit response body reading
	bodyReader := io.LimitReader(resp.Body, c.opt.MaxBodySize)

	// Decode response
	if resp.Request.Method != "HEAD" && resp.ContentLength > 0 {
		if req.Encoding != "" {
			if enc, _ := charset.Lookup(req.Encoding); enc != nil {
				bodyReader = transform.NewReader(bodyReader, enc.NewDecoder())
			}
		} else {
			if !c.opt.CharsetDetectDisabled {
				contentType := req.Header.Get("Content-Type")
				bodyReader, err = charset.NewReader(bodyReader, contentType)
				if err != nil {
					return nil, fmt.Errorf("charset detection error on content-type %s: %w", contentType, err)
				}
			}
		}
	}

	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	response := Response{
		Response: resp,
		Body:     body,
		Request:  req,
	}

	return &response, nil
}

// doRequestChrome opens up a new chrome instance and makes request
func (c *Client) doRequestChrome(req *Request) (*Response, error) {
	// Set remote allocator or use local chrome instance
	var allocCtx context.Context
	var allocCancel context.CancelFunc
	if c.opt.RemoteAllocatorURL != "" {
		allocCtx, allocCancel = chromedp.NewRemoteAllocator(context.Background(), c.opt.RemoteAllocatorURL)
	} else {
		allocCtx, allocCancel = chromedp.NewExecAllocator(context.Background(), c.opt.AllocatorOptions...)
	}
	defer allocCancel()

	// Task context
	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	defer taskCancel()

	// Initiate default pre actions
	var body string
	var res *network.Response
	var defaultPreActions = []chromedp.Action{
		network.Enable(),
		network.SetExtraHTTPHeaders(ConvertHeaderToMap(req.Header)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			chromedp.ListenTarget(ctx, func(ev interface{}) {
				if event, ok := ev.(*network.EventResponseReceived); ok {
					if res == nil && event.Type == "Document" {
						res = event.Response
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
	}

	// If options has pre actions, we override the default existing one.
	if len(c.opt.PreActions) != 0 {
		defaultPreActions = c.opt.PreActions
	}

	// Append custom actions to default ones.
	defaultPreActions = append(defaultPreActions, req.Actions...)

	// Run all actions
	if err := chromedp.Run(taskCtx, defaultPreActions...); err != nil {
		return nil, fmt.Errorf("request getting rendered: %w", err)
	}

	httpResponse := &http.Response{
		Request: req.Request,
	}

	// If response is set by default pre actions
	if res != nil {
		req.Header = ConvertMapToHeader(res.RequestHeaders)
		req.URL, _ = url.Parse(res.URL)
		httpResponse.StatusCode = int(res.Status)
		httpResponse.Proto = res.Protocol
		httpResponse.Header = ConvertMapToHeader(res.Headers)
	}

	response := Response{
		Response: httpResponse,
		Body:     []byte(body),
		Request:  req,
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
			return fmt.Errorf("stopped after %d redirects", maxRedirect)
		}
		return nil
	}
}
