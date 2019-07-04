package geziyor

import (
	"github.com/fpfeng/httpcache"
	"github.com/geziyor/geziyor/client"
	"github.com/geziyor/geziyor/metrics"
	"github.com/geziyor/geziyor/middleware"
	"io/ioutil"
	"log"
	"net/http/cookiejar"
	"os"
	"runtime/debug"
	"sync"
)

// Geziyor is our main scraper type
type Geziyor struct {
	Opt     *Options
	Client  *client.Client
	Exports chan interface{}

	metrics             *metrics.Metrics
	requestMiddlewares  []middleware.RequestProcessor
	responseMiddlewares []middleware.ResponseProcessor
	wgRequests          sync.WaitGroup
	wgExporters         sync.WaitGroup
	semGlobal           chan struct{}
	semHosts            struct {
		sync.RWMutex
		hostSems map[string]chan struct{}
	}
}

// NewGeziyor creates new Geziyor with default values.
// If options provided, options
func NewGeziyor(opt *Options) *Geziyor {
	geziyor := &Geziyor{
		Opt:     opt,
		Exports: make(chan interface{}, 1),
		requestMiddlewares: []middleware.RequestProcessor{
			&middleware.AllowedDomains{AllowedDomains: opt.AllowedDomains},
			&middleware.DuplicateRequests{RevisitEnabled: opt.URLRevisitEnabled},
			&middleware.Headers{UserAgent: opt.UserAgent},
			middleware.NewDelay(opt.RequestDelayRandomize, opt.RequestDelay),
		},
		responseMiddlewares: []middleware.ResponseProcessor{
			&middleware.ParseHTML{ParseHTMLDisabled: opt.ParseHTMLDisabled},
			&middleware.LogStats{LogDisabled: opt.LogDisabled},
		},
		metrics: metrics.NewMetrics(opt.MetricsType),
	}

	metricsMiddleware := &middleware.Metrics{Metrics: geziyor.metrics}
	geziyor.requestMiddlewares = append(geziyor.requestMiddlewares, metricsMiddleware)
	geziyor.responseMiddlewares = append(geziyor.responseMiddlewares, metricsMiddleware)

	// Default
	if opt.UserAgent == "" {
		opt.UserAgent = client.DefaultUserAgent
	}
	if opt.MaxBodySize == 0 {
		opt.MaxBodySize = client.DefaultMaxBody
	}
	if opt.RetryTimes == 0 {
		opt.RetryTimes = client.DefaultRetryTimes
	}
	if len(opt.RetryHTTPCodes) == 0 {
		opt.RetryHTTPCodes = client.DefaultRetryHTTPCodes
	}
	// Client
	geziyor.Client = client.NewClient(opt.MaxBodySize, opt.CharsetDetectDisabled, opt.RetryTimes, opt.RetryHTTPCodes)
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
	if opt.MaxRedirect != 0 {
		geziyor.Client.CheckRedirect = client.NewRedirectionHandler(opt.MaxRedirect)
	}
	// Concurrency
	if opt.ConcurrentRequests != 0 {
		geziyor.semGlobal = make(chan struct{}, opt.ConcurrentRequests)
	}
	if opt.ConcurrentRequestsPerDomain != 0 {
		geziyor.semHosts = struct {
			sync.RWMutex
			hostSems map[string]chan struct{}
		}{hostSems: make(map[string]chan struct{})}
	}
	// Middlewares
	geziyor.requestMiddlewares = append(geziyor.requestMiddlewares, opt.RequestMiddlewares...)
	geziyor.responseMiddlewares = append(geziyor.responseMiddlewares, opt.ResponseMiddlewares...)
	// Logging
	if opt.LogDisabled {
		log.SetOutput(ioutil.Discard)
	} else {
		log.SetOutput(os.Stdout)
	}

	return geziyor
}

// Start starts scraping
func (g *Geziyor) Start() {
	log.Println("Scraping Started")

	// Metrics
	if g.Opt.MetricsType == metrics.Prometheus || g.Opt.MetricsType == metrics.ExpVar {
		metricsServer := metrics.StartMetricsServer(g.Opt.MetricsType)
		defer metricsServer.Close()
	}

	// Start Exporters
	if len(g.Opt.Exporters) != 0 {
		g.wgExporters.Add(len(g.Opt.Exporters))
		for _, exp := range g.Opt.Exporters {
			go func() {
				defer g.wgExporters.Done()
				exp.Export(g.Exports)
			}()
		}
	} else {
		g.wgExporters.Add(1)
		go func() {
			for range g.Exports {
			}
			g.wgExporters.Done()
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

	g.wgRequests.Wait()
	close(g.Exports)
	g.wgExporters.Wait()
	log.Println("Scraping Finished")
}

// Get issues a GET to the specified URL.
func (g *Geziyor) Get(url string, callback func(g *Geziyor, r *client.Response)) {
	req, err := client.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Request creating error %v\n", err)
		return
	}
	g.Do(req, callback)
}

// GetRendered issues GET request using headless browser
// Opens up a new Chrome instance, makes request, waits for rendering HTML DOM and closed.
// Rendered requests only supported for GET requests.
func (g *Geziyor) GetRendered(url string, callback func(g *Geziyor, r *client.Response)) {
	req, err := client.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Request creating error %v\n", err)
		return
	}
	req.Rendered = true
	g.Do(req, callback)
}

// Head issues a HEAD to the specified URL
func (g *Geziyor) Head(url string, callback func(g *Geziyor, r *client.Response)) {
	req, err := client.NewRequest("HEAD", url, nil)
	if err != nil {
		log.Printf("Request creating error %v\n", err)
		return
	}
	g.Do(req, callback)
}

// Do sends an HTTP request
func (g *Geziyor) Do(req *client.Request, callback func(g *Geziyor, r *client.Response)) {
	if req.Synchronized {
		g.do(req, callback)
	} else {
		g.wgRequests.Add(1)
		go g.do(req, callback)
	}
}

// Do sends an HTTP request
func (g *Geziyor) do(req *client.Request, callback func(g *Geziyor, r *client.Response)) {
	g.acquireSem(req)
	defer g.releaseSem(req)
	if !req.Synchronized {
		defer g.wgRequests.Done()
	}
	defer g.recoverMe()

	for _, middlewareFunc := range g.requestMiddlewares {
		middlewareFunc.ProcessRequest(req)
		if req.Cancelled {
			return
		}
	}

	res, err := g.Client.DoRequest(req)
	if err != nil {
		log.Println(err)
		return
	}

	for _, middlewareFunc := range g.responseMiddlewares {
		middlewareFunc.ProcessResponse(res)
	}

	// Callbacks
	if callback != nil {
		callback(g, res)
	} else {
		if g.Opt.ParseFunc != nil {
			g.Opt.ParseFunc(g, res)
		}
	}
}

func (g *Geziyor) acquireSem(req *client.Request) {
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

func (g *Geziyor) releaseSem(req *client.Request) {
	if g.Opt.ConcurrentRequests != 0 {
		<-g.semGlobal
	}
	if g.Opt.ConcurrentRequestsPerDomain != 0 {
		<-g.semHosts.hostSems[req.Host]
	}
}

// recoverMe prevents scraping being crashed.
// Logs error and stack trace
func (g *Geziyor) recoverMe() {
	if r := recover(); r != nil {
		log.Println(r, string(debug.Stack()))
		g.metrics.PanicCounter.Add(1)
	}
}
