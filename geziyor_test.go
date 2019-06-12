package geziyor_test

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fpfeng/httpcache"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/exporter"
	"math/rand"
	"net/http"
	"testing"
	"time"
)

func TestSimple(t *testing.T) {
	geziyor.NewGeziyor(geziyor.Options{
		StartURLs: []string{"http://api.ipify.org"},
		ParseFunc: func(r *geziyor.Response) {
			fmt.Println(string(r.Body))
		},
	}).Start()
}

func TestSimpleCache(t *testing.T) {
	gez := geziyor.NewGeziyor(geziyor.Options{
		StartURLs: []string{"http://api.ipify.org"},
		Cache:     httpcache.NewMemoryCache(),
		ParseFunc: func(r *geziyor.Response) {
			fmt.Println(string(r.Body))
			r.Exports <- string(r.Body)
			r.Geziyor.Get("http://api.ipify.org", nil)
		},
	})
	gez.Start()
}

func TestQuotes(t *testing.T) {
	geziyor.NewGeziyor(geziyor.Options{
		StartURLs: []string{"http://quotes.toscrape.com/"},
		ParseFunc: quotesParse,
		Exporters: []geziyor.Exporter{exporter.JSONExporter{}},
	}).Start()
}

func quotesParse(r *geziyor.Response) {
	r.DocHTML.Find("div.quote").Each(func(i int, s *goquery.Selection) {
		// Export Data
		r.Exports <- map[string]interface{}{
			"number": i,
			"text":   s.Find("span.text").Text(),
			"author": s.Find("small.author").Text(),
			"tags": s.Find("div.tags > a.tag").Map(func(_ int, s *goquery.Selection) string {
				return s.Text()
			}),
		}
		//r.Exports <- []string{s.Find("span.text").Text(), s.Find("small.author").Text()}
	})

	// Next Page
	if href, ok := r.DocHTML.Find("li.next > a").Attr("href"); ok {
		go r.Geziyor.Get(r.JoinURL(href), quotesParse)
	}
}

func TestLinks(t *testing.T) {
	geziyor.NewGeziyor(geziyor.Options{
		AllowedDomains: []string{"books.toscrape.com"},
		StartURLs:      []string{"http://books.toscrape.com/"},
		ParseFunc:      linksParse,
		Exporters:      []geziyor.Exporter{exporter.CSVExporter{}},
	}).Start()
}

func linksParse(r *geziyor.Response) {
	r.Exports <- []string{r.Request.URL.String()}
	r.DocHTML.Find("a").Each(func(i int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			go r.Geziyor.Get(r.JoinURL(href), linksParse)
		}
	})
}

func TestRandomDelay(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	delay := time.Millisecond * 1000
	min := float64(delay) * 0.5
	max := float64(delay) * 1.5
	randomDelay := rand.Intn(int(max-min)) + int(min)
	fmt.Println(time.Duration(randomDelay))
}

func TestStartRequestsFunc(t *testing.T) {
	geziyor.NewGeziyor(geziyor.Options{
		StartRequestsFunc: func() []*http.Request {
			req, _ := http.NewRequest("GET", "http://quotes.toscrape.com/", nil)
			return []*http.Request{req}
		},
		ParseFunc: func(r *geziyor.Response) {
			r.Exports <- []string{r.Status}
		},
		Exporters: []geziyor.Exporter{exporter.CSVExporter{}},
	}).Start()
}
