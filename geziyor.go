package geziyor

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/fpfeng/httpcache"
	"github.com/geziyor/geziyor/client"
	"github.com/geziyor/geziyor/metrics"
	"io/ioutil"
	"log"
	"net/http/cookiejar"
	"sync"
)

// Extractor interface is for extracting data from HTML document
type Extractor interface {
	Extract(doc *goquery.Document) (interface{}, error)
}

// Exporter interface is for extracting data to external resources.
// Geziyor calls every extractors Export functions before any scraping starts.
// Export functions should wait for new data from exports chan.
type Exporter interface {
	Export(exports chan interface{})
}

// Geziyor is our main scraper type
type Geziyor struct {
	Opt     *Options
	Client  *client.Client
	Exports chan interface{}

	metrics             *metrics.Metrics
	requestMiddlewares  []RequestMiddleware
	responseMiddlewares []ResponseMiddleware
	wgRequests          sync.WaitGroup
	wgExporters         sync.WaitGroup
	semGlobal           chan struct{}
	semHosts            struct {
		sync.RWMutex
		hostSems map[string]chan struct{}
	}
	visitedURLs sync.Map
}

// NewGeziyor creates new Geziyor with default values.
// If options provided, options
func NewGeziyor(opt *Options) *Geziyor {
	geziyor := &Geziyor{
		Client:  client.NewClient(),
		Opt:     opt,
		Exports: make(chan interface{}),
		requestMiddlewares: []RequestMiddleware{
			allowedDomainsMiddleware,
			duplicateRequestsMiddleware,
			defaultHeadersMiddleware,
			delayMiddleware,
			logMiddleware,
			metricsRequestMiddleware,
		},
		responseMiddlewares: []ResponseMiddleware{
			parseHTMLMiddleware,
			metricsResponseMiddleware,
			extractorsMiddleware,
		},
		metrics: metrics.NewMetrics(opt.MetricsType),
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
// Opens up a new Chrome instance, makes request, waits for 1 second to render HTML DOM and closed.
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
	defer recoverMiddleware(g, req)

	for _, middlewareFunc := range g.requestMiddlewares {
		middlewareFunc(g, req)
		if req.Cancelled {
			return
		}
	}

	res, err := g.Client.DoRequest(req, g.Opt.MaxBodySize, g.Opt.CharsetDetectDisabled)
	if err != nil {
		log.Println(err)
		return
	}

	for _, middlewareFunc := range g.responseMiddlewares {
		middlewareFunc(g, res)
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
