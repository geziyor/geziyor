package geziyor_test

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fortytw2/leaktest"
	"github.com/fpfeng/httpcache"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/exporter"
	"github.com/geziyor/geziyor/extractor"
	"github.com/geziyor/geziyor/metrics"
	"testing"
)

func TestSimple(t *testing.T) {
	defer leaktest.Check(t)()
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"http://api.ipify.org"},
		ParseFunc: func(g *geziyor.Geziyor, r *geziyor.Response) {
			fmt.Println(string(r.Body))
		},
	}).Start()
}

func TestSimpleCache(t *testing.T) {
	defer leaktest.Check(t)()
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"http://api.ipify.org"},
		Cache:     httpcache.NewMemoryCache(),
		ParseFunc: func(g *geziyor.Geziyor, r *geziyor.Response) {
			fmt.Println(string(r.Body))
			g.Exports <- string(r.Body)
			g.Get("http://api.ipify.org", nil)
		},
	}).Start()
}

func TestQuotes(t *testing.T) {
	defer leaktest.Check(t)()
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"http://quotes.toscrape.com/"},
		ParseFunc: quotesParse,
		Exporters: []geziyor.Exporter{&exporter.JSON{}},
	}).Start()
}

func quotesParse(g *geziyor.Geziyor, r *geziyor.Response) {
	r.HTMLDoc.Find("div.quote").Each(func(i int, s *goquery.Selection) {
		// Export Data
		g.Exports <- map[string]interface{}{
			"number": i,
			"text":   s.Find("span.text").Text(),
			"author": s.Find("small.author").Text(),
			"tags": s.Find("div.tags > a.tag").Map(func(_ int, s *goquery.Selection) string {
				return s.Text()
			}),
		}
	})

	// Next Page
	if href, ok := r.HTMLDoc.Find("li.next > a").Attr("href"); ok {
		g.Get(r.JoinURL(href), quotesParse)
	}
}

func TestAllLinks(t *testing.T) {
	defer leaktest.Check(t)()

	geziyor.NewGeziyor(&geziyor.Options{
		AllowedDomains: []string{"books.toscrape.com"},
		StartURLs:      []string{"http://books.toscrape.com/"},
		ParseFunc: func(g *geziyor.Geziyor, r *geziyor.Response) {
			g.Exports <- []string{r.Request.URL.String()}
			r.HTMLDoc.Find("a").Each(func(i int, s *goquery.Selection) {
				if href, ok := s.Attr("href"); ok {
					g.Get(r.JoinURL(href), g.Opt.ParseFunc)
				}
			})
		},
		Exporters:   []geziyor.Exporter{&exporter.CSV{}},
		MetricsType: metrics.Prometheus,
	}).Start()
}

func TestStartRequestsFunc(t *testing.T) {
	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			g.Get("http://quotes.toscrape.com/", g.Opt.ParseFunc)
		},
		ParseFunc: func(g *geziyor.Geziyor, r *geziyor.Response) {
			r.HTMLDoc.Find("a").Each(func(_ int, s *goquery.Selection) {
				g.Exports <- s.AttrOr("href", "")
			})
		},
		Exporters: []geziyor.Exporter{&exporter.JSON{}},
	}).Start()
}

func TestGetRendered(t *testing.T) {
	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			g.GetRendered("https://httpbin.org/anything", g.Opt.ParseFunc)
		},
		ParseFunc: func(g *geziyor.Geziyor, r *geziyor.Response) {
			fmt.Println(string(r.Body))
			fmt.Println(r.Header)
		},
		//URLRevisitEnabled: true,
	}).Start()
}

func TestHEADRequest(t *testing.T) {
	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			g.Head("https://httpbin.org/anything", g.Opt.ParseFunc)
		},
		ParseFunc: func(g *geziyor.Geziyor, r *geziyor.Response) {
			fmt.Println(string(r.Body))
		},
	}).Start()
}

func TestCookies(t *testing.T) {
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"http://quotes.toscrape.com/login"},
		ParseFunc: func(g *geziyor.Geziyor, r *geziyor.Response) {
			if len(g.Client.Cookies("http://quotes.toscrape.com/login")) == 0 {
				t.Fatal("Cookies is Empty")
			}
		},
	}).Start()

	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"http://quotes.toscrape.com/login"},
		ParseFunc: func(g *geziyor.Geziyor, r *geziyor.Response) {
			if len(g.Client.Cookies("http://quotes.toscrape.com/login")) != 0 {
				t.Fatal("Cookies exist")
			}
		},
		CookiesDisabled: true,
	}).Start()
}

func TestBasicAuth(t *testing.T) {
	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			req, _ := geziyor.NewRequest("GET", "https://httpbin.org/anything", nil)
			req.SetBasicAuth("username", "password")
			g.Do(req, nil)
		},
		MetricsType: metrics.ExpVar,
	}).Start()
}

func TestExtractor(t *testing.T) {
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"https://www.theverge.com/2019/6/27/18760384/facebook-libra-currency-cryptocurrency-money-transfer-bank-problems-india-china"},
		Extractors: []geziyor.Extractor{
			&extractor.Text{Name: "title", Selector: ".c-page-title"},
			&extractor.Text{Name: "byline", Selector: ".c-byline__item:nth-child(1) > a"},
			&extractor.Text{Name: "summary", Selector: ".c-entry-summary"},
			&extractor.Text{Name: "content", Selector: ".c-entry-content"},
		},
		Exporters: []geziyor.Exporter{&exporter.JSON{}},
	}).Start()
}
