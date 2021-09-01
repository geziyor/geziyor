package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/geziyor/geziyor/internal"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
    "net/http/httptest"
    "net/http/httputil"
    "github.com/chromedp/cdproto/fetch"
	"log"
    "math/rand"
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
    ProxyPort             string
    ProxyAdress           string
    ProxyLogin            string
    ProxyPassword         string
    RandomSleep           bool
	AllocatorOptions      []chromedp.ExecAllocatorOption
}

// Default values for client
const (
    DefaultRandomSleep      = "none"
    DefaultProxyAdress      = "none"
    DefaultProxyPort        = "none"
    DefaultProxyLogin       = "none"
    DefaultProxyPassword    = "none"
	DefaultUserAgent        = "Geziyor 1.0"
	DefaultMaxBody    int64 = 1024 * 1024 * 1024 // 1GB
	DefaultRetryTimes       = 2
)

var (
	DefaultRetryHTTPCodes = []int{500, 502, 503, 504, 522, 524, 408}
)

// NewClient creates http.Client with modified values for typical web scraper
func NewClient(opt *Options) *Client {
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

    if c.opt.ProxyAdress != "none" || c.opt.ProxyPort != "none"  {
        p := httptest.NewServer(newProxy())
        defer p.Close()

        c.opt.AllocatorOptions = append(chromedp.DefaultExecAllocatorOptions[:],
            chromedp.ProxyServer(c.opt.ProxyAdress + ":" + c.opt.ProxyPort),
        )
    }
	var body string
	var res *network.Response

	// Set remote allocator or use local chrome instance
	var ctx context.Context
	var cancel context.CancelFunc
	if c.opt.RemoteAllocatorURL != "" {
		ctx, cancel = chromedp.NewRemoteAllocator(ctx, c.opt.RemoteAllocatorURL)
	} else {
		ctx, cancel = chromedp.NewExecAllocator(context.Background(), c.opt.AllocatorOptions...)
	}
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

        lctx, lcancel := context.WithCancel(ctx)
        chromedp.ListenTarget(lctx, func(ev interface{}) {
            switch ev := ev.(type) {
            case *fetch.EventRequestPaused:
                go func() {
                    _ = chromedp.Run(ctx, fetch.ContinueRequest(ev.RequestID))
                }()
            case *fetch.EventAuthRequired:
                if ev.AuthChallenge.Source == fetch.AuthChallengeSourceProxy {
                    go func() {
                        _ = chromedp.Run(ctx,
                            fetch.ContinueWithAuth(ev.RequestID, &fetch.AuthChallengeResponse{
                                Response: fetch.AuthChallengeResponseResponseProvideCredentials,
                                Username: c.opt.ProxyLogin,
                                Password: c.opt.ProxyPassword,
                            }),
                            // Chrome will remember the credential for the current instance,
                            // so we can disable the fetch domain once credential is provided.
                            // Please file an issue if Chrome does not work in this way.
                            fetch.Disable(),
                        )
                        // and cancel the event handler too.
                        lcancel()
                    }()
                }
            }
        })

    var randSleep int
        //fmt.Println(c.opt.RandomSleep)
    if c.opt.RandomSleep == true {
        rand.Seed(time.Now().Unix())
        randSleep = rand.Intn(240)
        fmt.Println(randSleep)
    }else{
        randSleep = 0
        fmt.Println("false")
    }

    if c.opt.ProxyAdress != "none" || c.opt.ProxyPort != "none"  {
        if err := chromedp.Run(ctx,
            fetch.Enable().WithHandleAuthRequests(true),
            chromedp.Sleep(time.Duration(randSleep)*time.Second),
            network.Enable(),
            network.SetExtraHTTPHeaders(ConvertHeaderToMap(req.Header)),
            chromedp.ActionFunc(func(ctx context.Context) error {
                var reqID network.RequestID
                chromedp.ListenTarget(ctx, func(ev interface{}) {
                    switch ev.(type) {
                    // Save main request ID to get response of it
                    case *network.EventRequestWillBeSent:
                        reqEvent := ev.(*network.EventRequestWillBeSent)
                        if _, exists := reqEvent.Request.Headers["Referer"]; !exists {
                            if strings.HasPrefix(reqEvent.Request.URL, "http") {
                                reqID = reqEvent.RequestID
                            }
                        }
                    // Save response using main request ID
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
            return nil, fmt.Errorf("request getting rendered: %w", err)
        }
    }else{
        if err := chromedp.Run(ctx,
            network.Enable(),
            network.SetExtraHTTPHeaders(ConvertHeaderToMap(req.Header)),
            chromedp.ActionFunc(func(ctx context.Context) error {
                var reqID network.RequestID
                chromedp.ListenTarget(ctx, func(ev interface{}) {
                    switch ev.(type) {
                    // Save main request ID to get response of it
                    case *network.EventRequestWillBeSent:
                        reqEvent := ev.(*network.EventRequestWillBeSent)
                        if _, exists := reqEvent.Request.Headers["Referer"]; !exists {
                            if strings.HasPrefix(reqEvent.Request.URL, "http") {
                                reqID = reqEvent.RequestID
                            }
                        }
                    // Save response using main request ID
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
            return nil, fmt.Errorf("request getting rendered: %w", err)
        }
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

func newProxy() *httputil.ReverseProxy {
    return &httputil.ReverseProxy{
        Director: func(r *http.Request) {
            if dump, err := httputil.DumpRequest(r, true); err == nil {
                log.Printf("%s", dump)
            }
            // hardcode username/password "u:p" (base64 encoded: dTpw ) to make it simple
            if auth := r.Header.Get("Proxy-Authorization"); auth != "Basic dTpw" {
                r.Header.Set("X-Failed", "407")
            }
        },
        Transport: &transport{http.DefaultTransport},
        ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
            if err.Error() == "407" {
                log.Println("proxy: not authorized")
                w.Header().Add("Proxy-Authenticate", `Basic realm="Proxy Authorization"`)
                w.WriteHeader(407)
            } else {
                w.WriteHeader(http.StatusBadGateway)
            }
        },
    }
}

type transport struct {
    http.RoundTripper
}

func (t *transport) RoundTrip(r *http.Request) (*http.Response, error) {
    if h := r.Header.Get("X-Failed"); h != "" {
        return nil, fmt.Errorf(h)
    }
    return t.RoundTripper.RoundTrip(r)
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
