package geziyor_test

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fortytw2/leaktest"
	"github.com/fpfeng/httpcache"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/exporter"
	"math/rand"
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
	defer leaktest.Check(t)()
	gez := geziyor.NewGeziyor(geziyor.Options{
		StartURLs: []string{"http://api.ipify.org"},
		Cache:     httpcache.NewMemoryCache(),
		ParseFunc: func(r *geziyor.Response) {
			fmt.Println(string(r.Body))
			r.Geziyor.Exports <- string(r.Body)
			r.Geziyor.Get("http://api.ipify.org", nil)
		},
	})
	gez.Start()
}

func TestQuotes(t *testing.T) {
	defer leaktest.Check(t)()
	geziyor.NewGeziyor(geziyor.Options{
		StartURLs: []string{"http://quotes.toscrape.com/"},
		ParseFunc: quotesParse,
		Exporters: []geziyor.Exporter{&exporter.JSONExporter{}},
	}).Start()
}

func quotesParse(r *geziyor.Response) {
	r.DocHTML.Find("div.quote").Each(func(i int, s *goquery.Selection) {
		// Export Data
		r.Geziyor.Exports <- map[string]interface{}{
			"number": i,
			"text":   s.Find("span.text").Text(),
			"author": s.Find("small.author").Text(),
			"tags": s.Find("div.tags > a.tag").Map(func(_ int, s *goquery.Selection) string {
				return s.Text()
			}),
		}
	})

	// Next Page
	if href, ok := r.DocHTML.Find("li.next > a").Attr("href"); ok {
		r.Geziyor.Get(r.JoinURL(href), quotesParse)
	}
}

func TestAllLinks(t *testing.T) {
	defer leaktest.Check(t)()

	geziyor.NewGeziyor(geziyor.Options{
		AllowedDomains: []string{"books.toscrape.com"},
		StartURLs:      []string{"http://books.toscrape.com/"},
		ParseFunc: func(r *geziyor.Response) {
			r.Geziyor.Exports <- []string{r.Request.URL.String()}
			r.DocHTML.Find("a").Each(func(i int, s *goquery.Selection) {
				if href, ok := s.Attr("href"); ok {
					r.Geziyor.Get(r.JoinURL(href), r.Geziyor.Opt.ParseFunc)
				}
			})
		},
		Exporters: []geziyor.Exporter{&exporter.CSVExporter{}},
	}).Start()
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
			g.Get("http://quotes.toscrape.com/", g.Opt.ParseFunc)
		},
		ParseFunc: func(r *geziyor.Response) {
			r.DocHTML.Find("a").Each(func(_ int, s *goquery.Selection) {
				r.Geziyor.Exports <- s.AttrOr("href", "")
			})
		},
		Exporters: []geziyor.Exporter{&exporter.JSONExporter{}},
	}).Start()
}

func TestGetRendered(t *testing.T) {
	geziyor.NewGeziyor(geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			g.GetRendered("https://httpbin.org/anything", g.Opt.ParseFunc)
		},
		ParseFunc: func(r *geziyor.Response) {
			fmt.Println(string(r.Body))
		},
		//URLRevisitEnabled: true,
	}).Start()
}
