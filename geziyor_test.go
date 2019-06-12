package geziyor_test

import (
	"encoding/json"
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
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			req, _ := http.NewRequest("GET", "http://quotes.toscrape.com/", nil)
			g.Requests <- &geziyor.Request{Request: req}
		},
		ParseFunc: func(r *geziyor.Response) {
			r.Exports <- []string{r.Status}
		},
		Exporters: []geziyor.Exporter{exporter.CSVExporter{}},
	}).Start()
}

func TestAlmaany(t *testing.T) {
	alphabet := "ab"

	geziyor.NewGeziyor(geziyor.Options{
		AllowedDomains: []string{"www.almaany.com"},
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			base := "http://www.almaany.com/suggest.php?term=%c%c&lang=turkish&t=d"
			for _, c1 := range alphabet {
				for _, c2 := range alphabet {
					req, _ := http.NewRequest("GET", fmt.Sprintf(base, c1, c2), nil)
					g.Requests <- &geziyor.Request{Request: req, Meta: map[string]interface{}{"word": string(c1) + string(c2)}}
				}
			}
		},
		ConcurrentRequests: 10,
		ParseFunc:          parseAlmaany,
		Exporters:          []geziyor.Exporter{exporter.CSVExporter{}},
	}).Start()

}

func parseAlmaany(r *geziyor.Response) {
	var words []string
	_ = json.Unmarshal(r.Body, &words)
	r.Exports <- words

	if len(words) == 20 {
		alphabet := "abcde"
		base := "http://www.almaany.com/suggest.php?term=%s%c&lang=turkish&t=d"

		for _, c := range alphabet {
			req, _ := http.NewRequest("GET", fmt.Sprintf(base, r.Meta["word"], c), nil)
			go r.Geziyor.Do(&geziyor.Request{Request: req, Meta: map[string]interface{}{"word": r.Meta["word"].(string) + string(c)}}, parseAlmaany)
		}
	}
}
