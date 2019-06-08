package geziyor

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fpfeng/httpcache"
	"io/ioutil"
	"log"
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
	opt    Opt

	visitedURLS []string
}

// Opt is custom options type for Geziyor
type Opt struct {
	AllowedDomains []string
	StartURLs      []string
	ParseFunc      func(response *Response)
	Cache          httpcache.Cache
}

func init() {
	log.SetOutput(os.Stdout)
}

// NewGeziyor creates new Geziyor with default values.
// If options provided, options
func NewGeziyor(opt Opt) *Geziyor {
	geziyor := &Geziyor{
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		opt: opt,
	}

	if opt.Cache != nil {
		geziyor.client.Transport = httpcache.NewTransport(opt.Cache)
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

// Do sends an HTTP request
func (g *Geziyor) Do(req *http.Request) {
	g.wg.Add(1)
	defer g.wg.Done()

	if !checkURL(req.URL, g) {
		return
	}

	// Log
	log.Println("Fetching: ", req.URL.String())

	// Do request
	resp, err := g.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return
	}

	// Read body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading body error: %v\n", err)
		return
	}

	// Create Document
	doc, _ := goquery.NewDocumentFromReader(bytes.NewReader(body))

	// Create response
	response := Response{
		Response: resp,
		Body:     body,
		Doc:      doc,
		Geziyor:  g,
		Exports:  make(chan map[string]interface{}, 1),
	}

	// Export Function
	go Export(&response)

	// ParseFunc response
	g.opt.ParseFunc(&response)
	time.Sleep(time.Millisecond)
}

func checkURL(parsedURL *url.URL, g *Geziyor) bool {
	rawURL := parsedURL.String()

	// Check for allowed domains
	if len(g.opt.AllowedDomains) != 0 && !contains(g.opt.AllowedDomains, parsedURL.Host) {
		log.Printf("Domain not allowed: %s\n", parsedURL.Host)
		return false
	}

	// Check for duplicate requests
	if contains(g.visitedURLS, rawURL) {
		log.Printf("URL already visited %s\n", rawURL)
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
