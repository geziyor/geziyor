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

type Geziyor struct {
	client *http.Client
	wg     sync.WaitGroup
	opt    Opt

	visitedURLS []string
}

type Opt struct {
	AllowedDomains []string
	StartURLs      []string
	ParseFunc      func(response *Response)
	Cache          httpcache.Cache
}

func init() {
	log.SetOutput(os.Stdout)
}

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

func (g *Geziyor) Start() {
	for _, startURL := range g.opt.StartURLs {
		go g.Get(startURL)
	}

	time.Sleep(time.Millisecond)
	g.wg.Wait()
}

func (g *Geziyor) Get(rawURL string) {
	g.wg.Add(1)
	defer g.wg.Done()

	if !checkURL(rawURL, g) {
		return
	}

	// Log
	log.Println("Fetching: ", rawURL)

	// Get request
	resp, err := g.client.Get(rawURL)
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

func checkURL(rawURL string, g *Geziyor) bool {

	// Parse URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "url parsing error: %v\n", err)
		return false
	}

	// Check for allowed domains
	if len(g.opt.AllowedDomains) != 0 && !Contains(g.opt.AllowedDomains, parsedURL.Host) {
		log.Printf("Domain not allowed: %s\n", parsedURL.Host)
		return false
	}

	// Check for duplicate requests
	if Contains(g.visitedURLS, rawURL) {
		log.Printf("URL already visited %s\n", rawURL)
		return false
	}
	g.visitedURLS = append(g.visitedURLS, rawURL)

	return true
}

// Contains checks whether []string contains string
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
