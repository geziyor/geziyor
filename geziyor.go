package geziyor

import (
	"github.com/chromedp/chromedp"
	"github.com/geziyor/geziyor/cache"
	"github.com/geziyor/geziyor/client"
	"github.com/geziyor/geziyor/export"
	"github.com/geziyor/geziyor/internal"
	"github.com/geziyor/geziyor/metrics"
	"github.com/geziyor/geziyor/middleware"
	"golang.org/x/time/rate"
	"io/ioutil"
	"net/http/cookiejar"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
)

// Geziyor is our main scraper type
type Geziyor struct {
	Opt     *Options
	Client  *client.Client
	Exports chan interface{}

	metrics        *metrics.Metrics
	reqMiddlewares []middleware.RequestProcessor
	resMiddlewares []middleware.ResponseProcessor
	rateLimiter    *rate.Limiter
	wgRequests     sync.WaitGroup
	wgExporters    sync.WaitGroup
	semGlobal      chan struct{}
	semHosts       struct {
		sync.RWMutex
		hostSems map[string]chan struct{}
	}
	shutdown bool
}

// NewGeziyor creates new Geziyor with default values.
// If options provided, options
func NewGeziyor(opt *Options) *Geziyor {

	// Default Options
	if opt.UserAgent == "" {
		opt.UserAgent = client.DefaultUserAgent
	}
    if opt.ProxyAdress == "" {
        opt.ProxyAdress = client.DefaultProxyAdress
    }
    if opt.ProxyLogin == "" {
        opt.ProxyLogin = client.DefaultProxyLogin
    }
    if opt.ProxyPort == "" {
        opt.ProxyPort = client.DefaultProxyPort
    }
    if opt.ProxyPassword == "" {
        opt.ProxyPassword = client.DefaultProxyPassword
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

	geziyor := &Geziyor{
		Opt:     opt,
		Exports: make(chan interface{}, 1),
		reqMiddlewares: []middleware.RequestProcessor{
			&middleware.AllowedDomains{AllowedDomains: opt.AllowedDomains},
			&middleware.DuplicateRequests{RevisitEnabled: opt.URLRevisitEnabled},
			&middleware.Headers{UserAgent: opt.UserAgent},
			middleware.NewDelay(opt.RequestDelayRandomize, opt.RequestDelay),
		},
		resMiddlewares: []middleware.ResponseProcessor{
			&middleware.ParseHTML{ParseHTMLDisabled: opt.ParseHTMLDisabled},
			&middleware.LogStats{LogDisabled: opt.LogDisabled},
		},
		metrics: metrics.NewMetrics(opt.MetricsType),
	}

	// Client
	geziyor.Client = client.NewClient(&client.Options{
		MaxBodySize:           opt.MaxBodySize,
		CharsetDetectDisabled: opt.CharsetDetectDisabled,
		RetryTimes:            opt.RetryTimes,
		RetryHTTPCodes:        opt.RetryHTTPCodes,
		RemoteAllocatorURL:    opt.BrowserEndpoint,
        ProxyPort:             opt.ProxyPort,
        ProxyAdress:           opt.ProxyAdress,
        ProxyLogin:            opt.ProxyLogin,
        ProxyPassword:         opt.ProxyPassword,
        RandomSleep:           opt.RandomSleep,
		AllocatorOptions:      chromedp.DefaultExecAllocatorOptions[:],
	})
	if opt.Cache != nil {
		geziyor.Client.Transport = &cache.Transport{
			Policy:              opt.CachePolicy,
			Transport:           geziyor.Client.Transport,
			Cache:               opt.Cache,
			MarkCachedResponses: true,
		}
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
	if opt.RequestsPerSecond != 0 {
		geziyor.rateLimiter = rate.NewLimiter(rate.Limit(opt.RequestsPerSecond), int(opt.RequestsPerSecond))
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

	// Base Middlewares
	metricsMiddleware := &middleware.Metrics{Metrics: geziyor.metrics}
	geziyor.reqMiddlewares = append(geziyor.reqMiddlewares, metricsMiddleware)
	geziyor.resMiddlewares = append(geziyor.resMiddlewares, metricsMiddleware)

	robotsMiddleware := middleware.NewRobotsTxt(geziyor.Client, geziyor.metrics, opt.RobotsTxtDisabled)
	geziyor.reqMiddlewares = append(geziyor.reqMiddlewares, robotsMiddleware)

	// Custom Middlewares
	geziyor.reqMiddlewares = append(geziyor.reqMiddlewares, opt.RequestMiddlewares...)
	geziyor.resMiddlewares = append(geziyor.resMiddlewares, opt.ResponseMiddlewares...)

	// Logging
	if opt.LogDisabled {
		internal.Logger.SetOutput(ioutil.Discard)
	} else {
		internal.Logger.SetOutput(os.Stdout)
	}

	return geziyor
}

// Start starts scraping
func (g *Geziyor) Start() {
	internal.Logger.Println("Scraping Started")

	// Metrics
	if g.Opt.MetricsType == metrics.Prometheus || g.Opt.MetricsType == metrics.ExpVar {
		metricsServer := metrics.StartMetricsServer(g.Opt.MetricsType)
		defer metricsServer.Close()
	}

	// Start Exporters
	if len(g.Opt.Exporters) != 0 {
		g.wgExporters.Add(len(g.Opt.Exporters))
		for _, exporter := range g.Opt.Exporters {
			go func(exporter export.Exporter) {
				defer g.wgExporters.Done()
				if err := exporter.Export(g.Exports); err != nil {
					internal.Logger.Printf("exporter error: %s\n", err)
				}
			}(exporter)
		}
	} else {
		g.wgExporters.Add(1)
		go func() {
			for range g.Exports {
			}
			g.wgExporters.Done()
		}()
	}

	// Wait for SIGINT (interrupt) signal.
	shutdownChan := make(chan os.Signal, 1)
	shutdownDoneChan := make(chan struct{})
	signal.Notify(shutdownChan, os.Interrupt)
	go g.interruptSignalWaiter(shutdownChan, shutdownDoneChan)

	// Start Requests
	if g.Opt.StartRequestsFunc != nil {
		g.Opt.StartRequestsFunc(g)
	} else {
		for _, startURL := range g.Opt.StartURLs {
			g.Get(startURL, g.Opt.ParseFunc)
		}
	}

	g.wgRequests.Wait()
	close(g.Exports)
	g.wgExporters.Wait()
	shutdownDoneChan <- struct{}{}
	internal.Logger.Println("Scraping Finished")
}

// Get issues a GET to the specified URL.
func (g *Geziyor) Get(url string, callback func(g *Geziyor, r *client.Response)) {
	req, err := client.NewRequest("GET", url, nil)
	if err != nil {
		internal.Logger.Printf("Request creating error %v\n", err)
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
		internal.Logger.Printf("Request creating error %v\n", err)
		return
	}
	req.Rendered = true
	g.Do(req, callback)
}

// Head issues a HEAD to the specified URL
func (g *Geziyor) Head(url string, callback func(g *Geziyor, r *client.Response)) {
	req, err := client.NewRequest("HEAD", url, nil)
	if err != nil {
		internal.Logger.Printf("Request creating error %v\n", err)
		return
	}
	g.Do(req, callback)
}

// Do sends an HTTP request
func (g *Geziyor) Do(req *client.Request, callback func(g *Geziyor, r *client.Response)) {
	if g.shutdown {
		return
	}
	g.wgRequests.Add(1)
	if req.Synchronized {
		g.do(req, callback)
	} else {
		go g.do(req, callback)
	}
}

// Do sends an HTTP request
func (g *Geziyor) do(req *client.Request, callback func(g *Geziyor, r *client.Response)) {
	g.acquireSem(req)
	defer g.releaseSem(req)
	defer g.wgRequests.Done()
	defer g.recoverMe()

	for _, middlewareFunc := range g.reqMiddlewares {
		middlewareFunc.ProcessRequest(req)
		if req.Cancelled {
			return
		}
	}

	res, err := g.Client.DoRequest(req)
	if err != nil {
		if g.Opt.ErrorFunc != nil {
			g.Opt.ErrorFunc(g, req, err)
		} else {
			internal.Logger.Println(err)
		}
		return
	}

	for _, middlewareFunc := range g.resMiddlewares {
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
	if g.rateLimiter != nil {
		_ = g.rateLimiter.Wait(req.Context())
	}
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
		g.semHosts.RLock()
		hostSem := g.semHosts.hostSems[req.Host]
		g.semHosts.RUnlock()
		<-hostSem
	}
}

// recoverMe prevents scraping being crashed.
// Logs error and stack trace
func (g *Geziyor) recoverMe() {
	if r := recover(); r != nil {
		internal.Logger.Println(r, string(debug.Stack()))
		g.metrics.PanicCounter.Add(1)
	}
}

// interruptSignalWaiter waits data from provided channels and stops scraper if shutdownChan channel receives SIGINT
func (g *Geziyor) interruptSignalWaiter(shutdownChan chan os.Signal, shutdownDoneChan chan struct{}) {
	for {
		select {
		case <-shutdownChan:
			internal.Logger.Println("Received SIGINT, shutting down gracefully. Send again to force")
			g.shutdown = true
			signal.Stop(shutdownChan)
		case <-shutdownDoneChan:
			return
		}
	}
}
