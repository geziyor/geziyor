package geziyor

import (
	"bytes"
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"github.com/fpfeng/httpcache"
	"golang.org/x/net/html/charset"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

// Exporter interface is for extracting data to external resources
type Exporter interface {
	Export(r *Response)
}

// RequestMiddleware called before requests made.
// Set request.Cancelled = true to cancel request
type RequestMiddleware func(g *Geziyor, r *Request)

// Geziyor is our main scraper type
type Geziyor struct {
	Opt Options

	client    *http.Client
	wg        sync.WaitGroup
	semGlobal chan struct{}
	semHosts  struct {
		sync.RWMutex
		hostSems map[string]chan struct{}
	}
	visitedURLS struct {
		sync.RWMutex
		visitedURLS []string
	}
	requestMiddlewaresBase []RequestMiddleware
}

func init() {
	log.SetOutput(os.Stdout)
	rand.Seed(time.Now().UnixNano())
}

// NewGeziyor creates new Geziyor with default values.
// If options provided, options
func NewGeziyor(opt Options) *Geziyor {
	geziyor := &Geziyor{
		client: &http.Client{
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
		},
		Opt:                    opt,
		requestMiddlewaresBase: []RequestMiddleware{defaultHeadersMiddleware},
	}

	if opt.Cache != nil {
		geziyor.client.Transport = &httpcache.Transport{
			Transport: geziyor.client.Transport, Cache: opt.Cache, MarkCachedResponses: true}
	}
	if opt.Timeout != 0 {
		geziyor.client.Timeout = opt.Timeout
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
	if opt.UserAgent == "" {
		geziyor.Opt.UserAgent = "Geziyor 1.0"
	}
	if opt.LogDisabled {
		log.SetOutput(ioutil.Discard)
	}
	if opt.MaxBodySize == 0 {
		geziyor.Opt.MaxBodySize = 1024 * 1024 * 1024 // 1GB
	}

	return geziyor
}

// Start starts scraping
func (g *Geziyor) Start() {
	log.Println("Scraping Started")

	if g.Opt.StartRequestsFunc == nil {
		for _, startURL := range g.Opt.StartURLs {
			g.Get(startURL, g.Opt.ParseFunc)
		}
	} else {
		g.Opt.StartRequestsFunc(g)
	}

	g.wg.Wait()

	log.Println("Scraping Finished")
}

// Get issues a GET to the specified URL.
func (g *Geziyor) Get(url string, callback func(resp *Response)) {
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
func (g *Geziyor) GetRendered(url string, callback func(resp *Response)) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Request creating error %v\n", err)
		return
	}
	g.Do(&Request{Request: req, Rendered: true}, callback)
}

// Head issues a HEAD to the specified URL
func (g *Geziyor) Head(url string, callback func(resp *Response)) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		log.Printf("Request creating error %v\n", err)
		return
	}
	g.Do(&Request{Request: req}, callback)
}

// Do sends an HTTP request
func (g *Geziyor) Do(req *Request, callback func(resp *Response)) {
	g.wg.Add(1)
	go g.do(req, callback)
}

// Do sends an HTTP request
func (g *Geziyor) do(req *Request, callback func(resp *Response)) {
	defer g.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Println(r, string(debug.Stack()))
		}
	}()

	if !g.checkURL(req.URL) {
		return
	}

	// Request Middlewares
	for _, middlewareFunc := range g.requestMiddlewaresBase {
		middlewareFunc(g, req)
	}
	for _, middlewareFunc := range g.Opt.RequestMiddlewares {
		middlewareFunc(g, req)
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

	if !g.Opt.ParseHTMLDisabled && response.isHTML() {
		response.DocHTML, _ = goquery.NewDocumentFromReader(bytes.NewReader(response.Body))
	}

	// Exporter functions
	for _, exp := range g.Opt.Exporters {
		go exp.Export(response)
	}

	// Drain exports chan if no exporter functions added
	if len(g.Opt.Exporters) == 0 {
		go func() {
			for range response.Exports {
			}
		}()
	}

	// Callbacks
	if callback != nil {
		callback(response)
	} else {
		if g.Opt.ParseFunc != nil {
			g.Opt.ParseFunc(response)
		}
	}

	// Close exports chan to prevent goroutine leak
	close(response.Exports)
}

func (g *Geziyor) doRequestClient(req *Request) (*Response, error) {
	g.acquireSem(req)
	defer g.releaseSem(req)

	g.delay()

	log.Println("Fetching: ", req.URL.String())

	// Do request
	resp, err := g.client.Do(req.Request)
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
	if !g.Opt.CharsetDetectDisabled {
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
		Geziyor:  g,
		Exports:  make(chan interface{}),
	}

	return &response, nil
}

func (g *Geziyor) doRequestChrome(req *Request) (*Response, error) {
	g.acquireSem(req)
	defer g.releaseSem(req)

	g.delay()

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var res string

	if err := chromedp.Run(ctx,
		chromedp.Navigate(req.URL.String()),
		chromedp.Sleep(1*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			node, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}
			res, err = dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
			return err
		}),
	); err != nil {
		log.Printf("Request getting rendered error: %v\n", err)
		return nil, err
	}

	response := &Response{
		//Response: resp,
		Body:    []byte(res),
		Meta:    req.Meta,
		Geziyor: g,
		Exports: make(chan interface{}),
	}

	return response, nil
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

func (g *Geziyor) checkURL(parsedURL *url.URL) bool {
	rawURL := parsedURL.String()
	// Check for allowed domains
	if len(g.Opt.AllowedDomains) != 0 && !contains(g.Opt.AllowedDomains, parsedURL.Host) {
		//log.Printf("Domain not allowed: %s\n", parsedURL.Host)
		return false
	}

	// Check for duplicate requests
	if !g.Opt.URLRevisitEnabled {
		g.visitedURLS.RLock()
		if contains(g.visitedURLS.visitedURLS, rawURL) {
			g.visitedURLS.RUnlock()
			//log.Printf("URL already visited %s\n", rawURL)
			return false
		}
		g.visitedURLS.RUnlock()
		g.visitedURLS.Lock()
		g.visitedURLS.visitedURLS = append(g.visitedURLS.visitedURLS, rawURL)
		g.visitedURLS.Unlock()
	}

	return true
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

// contains checks whether []string contains string
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
