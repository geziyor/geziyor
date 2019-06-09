package geziyor

import (
	"bytes"
	"github.com/PuerkitoBio/goquery"
	"github.com/fpfeng/httpcache"
	"golang.org/x/net/html/charset"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

// Geziyor is our main scraper type
type Geziyor struct {
	client *http.Client
	wg     sync.WaitGroup
	opt    Options

	visitedURLS []string
	semGlobal   chan struct{}
	semHosts    struct {
		sync.RWMutex
		hostSems map[string]chan struct{}
	}
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
			Timeout: time.Second * 60,
		},
		opt: opt,
	}

	if opt.Cache != nil {
		geziyor.client.Transport = httpcache.NewTransport(opt.Cache)
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
		geziyor.opt.UserAgent = "Geziyor 1.0"
	}
	if opt.LogDisabled {
		log.SetOutput(ioutil.Discard)
	}

	return geziyor
}

// Start starts scraping
func (g *Geziyor) Start() {
	for _, startURL := range g.opt.StartURLs {
		go g.Get(startURL)
	}

	time.Sleep(time.Millisecond)
	g.wg.Wait()
}

// Get issues a GET to the specified URL.
func (g *Geziyor) Get(url string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Request creating error %v\n", err)
		return
	}
	g.Do(req)
}

// Head issues a HEAD to the specified URL
func (g *Geziyor) Head(url string) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		log.Printf("Request creating error %v\n", err)
		return
	}
	g.Do(req)
}

// Do sends an HTTP request
func (g *Geziyor) Do(req *http.Request) {
	g.wg.Add(1)
	defer g.wg.Done()

	if !g.checkURL(req.URL) {
		return
	}

	// Modify Request
	req.Header.Set("Accept-Charset", "utf-8")
	req.Header.Set("User-Agent", g.opt.UserAgent)

	// Acquire Semaphore
	g.acquireSem(req)

	// Request Delay
	if g.opt.RequestDelayRandomize {
		min := float64(g.opt.RequestDelay) * 0.5
		max := float64(g.opt.RequestDelay) * 1.5
		time.Sleep(time.Duration(rand.Intn(int(max-min)) + int(min)))
	} else {
		time.Sleep(g.opt.RequestDelay)
	}

	// Log
	log.Println("Fetching: ", req.URL.String())

	// Do request
	resp, err := g.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Printf("Response error: %v\n", err)
		g.releaseSem(req)
		return
	}

	// Start reading body and determine encoding
	reader, err := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
	if err != nil {
		log.Printf("Determine encoding error: %v\n", err)
		g.releaseSem(req)
		return
	}

	// Continue reading body
	body, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Printf("Reading Body error: %v\n", err)
		g.releaseSem(req)
		return
	}

	// Release Semaphore
	g.releaseSem(req)

	// Create Document
	doc, _ := goquery.NewDocumentFromReader(bytes.NewReader(body))

	// Create response
	response := Response{
		Response: resp,
		Body:     body,
		Doc:      doc,
		Geziyor:  g,
		Exports:  make(chan interface{}, 1),
	}

	// Export Function
	go Export(&response)

	// ParseFunc response
	g.opt.ParseFunc(&response)
	time.Sleep(time.Millisecond)
}

func (g *Geziyor) acquireSem(req *http.Request) {
	if g.opt.ConcurrentRequests != 0 {
		g.semGlobal <- struct{}{}
	}

	if g.opt.ConcurrentRequestsPerDomain != 0 {
		g.semHosts.RLock()
		hostSem, exists := g.semHosts.hostSems[req.Host]
		g.semHosts.RUnlock()
		if !exists {
			hostSem = make(chan struct{}, g.opt.ConcurrentRequestsPerDomain)
			g.semHosts.Lock()
			g.semHosts.hostSems[req.Host] = hostSem
			g.semHosts.Unlock()
		}
		hostSem <- struct{}{}
	}
}

func (g *Geziyor) releaseSem(req *http.Request) {
	if g.opt.ConcurrentRequests != 0 {
		<-g.semGlobal
	}
	if g.opt.ConcurrentRequestsPerDomain != 0 {
		<-g.semHosts.hostSems[req.Host]
	}
}

func (g *Geziyor) checkURL(parsedURL *url.URL) bool {
	rawURL := parsedURL.String()
	// Check for allowed domains
	if len(g.opt.AllowedDomains) != 0 && !contains(g.opt.AllowedDomains, parsedURL.Host) {
		//log.Printf("Domain not allowed: %s\n", parsedURL.Host)
		return false
	}

	// Check for duplicate requests
	if contains(g.visitedURLS, rawURL) {
		//log.Printf("URL already visited %s\n", rawURL)
		return false
	}
	g.visitedURLS = append(g.visitedURLS, rawURL)

	return true
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
