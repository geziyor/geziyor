package gezer

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	httpcacheDumb "github.com/fpfeng/httpcache"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

type Gezer struct {
	client *http.Client
	wg     sync.WaitGroup
	opt    Opt
}

type Opt struct {
	AllowedDomains []string
	StartURLs      []string
	ParseFunc      func(response *Response)
}

func NewGezer(opt Opt) *Gezer {
	log.SetOutput(os.Stdout)
	return &Gezer{
		client: &http.Client{
			Timeout:   time.Second * 10,
			Transport: httpcacheDumb.NewMemoryCacheTransport(),
		},
		opt: opt,
	}
}

func (g *Gezer) Start() {
	for _, startURL := range g.opt.StartURLs {
		go g.Get(startURL)
	}

	time.Sleep(time.Millisecond)
	g.wg.Wait()
}

func (g *Gezer) Get(rawURL string) {
	g.wg.Add(1)
	defer g.wg.Done()

	if !checkURL(rawURL, g.opt.AllowedDomains) {
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
		Gezer:    g,
		Exports:  make(chan map[string]interface{}, 1),
	}

	// Export Function
	go Export(&response)

	// ParseFunc response
	g.opt.ParseFunc(&response)
	time.Sleep(time.Millisecond)
}

func checkURL(rawURL string, allowedDomains []string) bool {

	// Parse URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "url parsing error: %v\n", err)
		return false
	}

	// Check for allowed domains
	if len(allowedDomains) != 0 && !Contains(allowedDomains, parsedURL.Host) {
		log.Printf("Domain not allowed: %s\n", parsedURL.Host)
		return false
	}

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
