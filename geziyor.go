package geziyor

import (
	"context"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/fpfeng/httpcache"
	"github.com/geziyor/geziyor/internal"
	"golang.org/x/net/html/charset"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

// Exporter interface is for extracting data to external resources
type Exporter interface {
	Export(exports chan interface{})
}

// Geziyor is our main scraper type
type Geziyor struct {
	Opt     *Options
	Client  *internal.Client
	Exports chan interface{}

	wg        sync.WaitGroup
	semGlobal chan struct{}
	semHosts  struct {
		sync.RWMutex
		hostSems map[string]chan struct{}
	}
	visitedURLs         sync.Map
	requestMiddlewares  []RequestMiddleware
	responseMiddlewares []ResponseMiddleware
}

func init() {
	log.SetOutput(os.Stdout)
	rand.Seed(time.Now().UnixNano())
}

// NewGeziyor creates new Geziyor with default values.
// If options provided, options
func NewGeziyor(opt *Options) *Geziyor {
	geziyor := &Geziyor{
		Client:  internal.NewClient(),
		Opt:     opt,
		Exports: make(chan interface{}),
		requestMiddlewares: []RequestMiddleware{
			allowedDomainsMiddleware,
			duplicateRequestsMiddleware,
			defaultHeadersMiddleware,
		},
		responseMiddlewares: []ResponseMiddleware{
			parseHTMLMiddleware,
		},
	}

	if opt.UserAgent == "" {
		geziyor.Opt.UserAgent = "Geziyor 1.0"
	}
	if opt.MaxBodySize == 0 {
		geziyor.Opt.MaxBodySize = 1024 * 1024 * 1024 // 1GB
	}
	if opt.Cache != nil {
		geziyor.Client.Transport = &httpcache.Transport{
			Transport: geziyor.Client.Transport, Cache: opt.Cache, MarkCachedResponses: true}
	}
	if opt.Timeout != 0 {
		geziyor.Client.Timeout = opt.Timeout
	}
	if !opt.CookiesDisabled {
		geziyor.Client.Jar, _ = cookiejar.New(nil)
	}
	if opt.ConcurrentRequests != 0 {
		geziyor.semGlobal = make(chan struct{}, opt.ConcurrentRequests)
	}
	if opt.ConcurrentRequestsPerDomain != 0 {
		geziyor.semHosts = struct {
			sync.RWMutex
			hostSems map[string]chan struct{}
		}{hostSems: make(map[string]chan struct{})}
	}
	if opt.LogDisabled {
		log.SetOutput(ioutil.Discard)
	}
	geziyor.requestMiddlewares = append(geziyor.requestMiddlewares, opt.RequestMiddlewares...)
	geziyor.responseMiddlewares = append(geziyor.responseMiddlewares, opt.ResponseMiddlewares...)

	return geziyor
}

// Start starts scraping
func (g *Geziyor) Start() {
	log.Println("Scraping Started")

	// Start Exporters
	if len(g.Opt.Exporters) != 0 {
		for _, exp := range g.Opt.Exporters {
			go exp.Export(g.Exports)
		}
	} else {
		go func() {
			for range g.Exports {
			}
		}()
	}

	// Start Requests
	if g.Opt.StartRequestsFunc == nil {
		for _, startURL := range g.Opt.StartURLs {
			g.Get(startURL, g.Opt.ParseFunc)
		}
	} else {
		g.Opt.StartRequestsFunc(g)
	}

	g.wg.Wait()
	close(g.Exports)
	log.Println("Scraping Finished")
}

// Get issues a GET to the specified URL.
func (g *Geziyor) Get(url string, callback func(g *Geziyor, r *Response)) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Request creating error %v\n", err)
		return
	}
	g.Do(&Request{Request: req}, callback)
}

// GetRendered issues GET request using headless browser
// Opens up a new Chrome instance, makes request, waits for 1 second to render HTML DOM and closed.
// Rendered requests only supported for GET requests.
func (g *Geziyor) GetRendered(url string, callback func(g *Geziyor, r *Response)) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Request creating error %v\n", err)
		return
	}
	g.Do(&Request{Request: req, Rendered: true}, callback)
}

// Head issues a HEAD to the specified URL
func (g *Geziyor) Head(url string, callback func(g *Geziyor, r *Response)) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		log.Printf("Request creating error %v\n", err)
		return
	}
	g.Do(&Request{Request: req}, callback)
}

// Do sends an HTTP request
func (g *Geziyor) Do(req *Request, callback func(g *Geziyor, r *Response)) {
	g.wg.Add(1)
	go g.do(req, callback)
}

// Do sends an HTTP request
func (g *Geziyor) do(req *Request, callback func(g *Geziyor, r *Response)) {
	defer g.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Println(r, string(debug.Stack()))
		}
	}()

	for _, middlewareFunc := range g.requestMiddlewares {
		middlewareFunc(g, req)
		if req.Cancelled {
			return
		}
	}

	// Do request normal or Chrome and read response
	var response *Response
	var err error
	if !req.Rendered {
		response, err = g.doRequestClient(req)
	} else {
		response, err = g.doRequestChrome(req)
	}
	if err != nil {
		return
	}

	for _, middlewareFunc := range g.responseMiddlewares {
		middlewareFunc(g, response)
	}

	// Callbacks
	if callback != nil {
		callback(g, response)
	} else {
		if g.Opt.ParseFunc != nil {
			g.Opt.ParseFunc(g, response)
		}
	}
}

func (g *Geziyor) doRequestClient(req *Request) (*Response, error) {
	g.acquireSem(req)
	defer g.releaseSem(req)

	g.delay()

	log.Println("Fetching: ", req.URL.String())

	// Do request
	resp, err := g.Client.Do(req.Request)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Printf("Response error: %v\n", err)
		return nil, err
	}

	// Limit response body reading
	bodyReader := io.LimitReader(resp.Body, g.Opt.MaxBodySize)

	// Start reading body and determine encoding
	if !g.Opt.CharsetDetectDisabled && resp.Request.Method != "HEAD" {
		bodyReader, err = charset.NewReader(bodyReader, resp.Header.Get("Content-Type"))
		if err != nil {
			log.Printf("Determine encoding error: %v\n", err)
			return nil, err
		}
	}

	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		log.Printf("Reading Body error: %v\n", err)
		return nil, err
	}

	response := Response{
		Response: resp,
		Body:     body,
		Meta:     req.Meta,
		Request:  req,
	}

	return &response, nil
}

func (g *Geziyor) doRequestChrome(req *Request) (*Response, error) {
	g.acquireSem(req)
	defer g.releaseSem(req)

	g.delay()

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var body string
	var res *network.Response

	if err := chromedp.Run(ctx,
		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers(internal.ConvertHeaderToMap(req.Header))),
		chromedp.ActionFunc(func(ctx context.Context) error {
			chromedp.ListenTarget(ctx, func(ev interface{}) {
				switch ev.(type) {
				case *network.EventResponseReceived:
					res = ev.(*network.EventResponseReceived).Response
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
		log.Printf("Request getting rendered error: %v\n", err)
		return nil, err
	}

	// Set new URL in case of redirection
	req.URL, _ = url.Parse(res.URL)

	response := Response{
		Response: &http.Response{
			Request:    req.Request,
			StatusCode: int(res.Status),
		},
		Body:    []byte(body),
		Meta:    req.Meta,
		Request: req,
	}

	return &response, nil
}

func (g *Geziyor) acquireSem(req *Request) {
	if g.Opt.ConcurrentRequests != 0 {
		g.semGlobal <- struct{}{}
	}
	if g.Opt.ConcurrentRequestsPerDomain != 0 {
		g.semHosts.RLock()
		hostSem, exists := g.semHosts.hostSems[req.Host]
		g.semHosts.RUnlock()
		if !exists {
			hostSem = make(chan struct{}, g.Opt.ConcurrentRequestsPerDomain)
			g.semHosts.Lock()
			g.semHosts.hostSems[req.Host] = hostSem
			g.semHosts.Unlock()
		}
		hostSem <- struct{}{}
	}
}

func (g *Geziyor) releaseSem(req *Request) {
	if g.Opt.ConcurrentRequests != 0 {
		<-g.semGlobal
	}
	if g.Opt.ConcurrentRequestsPerDomain != 0 {
		<-g.semHosts.hostSems[req.Host]
	}
}

func (g *Geziyor) delay() {
	if g.Opt.RequestDelayRandomize {
		min := float64(g.Opt.RequestDelay) * 0.5
		max := float64(g.Opt.RequestDelay) * 1.5
		time.Sleep(time.Duration(rand.Intn(int(max-min)) + int(min)))
	} else {
		time.Sleep(g.Opt.RequestDelay)
	}
}
