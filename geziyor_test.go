package geziyor_test

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fortytw2/leaktest"
	"github.com/fpfeng/httpcache"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
	"github.com/geziyor/geziyor/export"
	"github.com/geziyor/geziyor/extract"
	"github.com/geziyor/geziyor/metrics"
	"net/http"
	"net/http/httptest"
	"testing"
	"unicode/utf8"
)

func TestSimple(t *testing.T) {
	defer leaktest.Check(t)()
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"http://api.ipify.org"},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			fmt.Println(string(r.Body))
		},
	}).Start()
}

func TestSimpleCache(t *testing.T) {
	defer leaktest.Check(t)()
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"http://api.ipify.org"},
		Cache:     httpcache.NewMemoryCache(),
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
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
		Exporters: []geziyor.Exporter{&export.JSON{}},
	}).Start()
}

func quotesParse(g *geziyor.Geziyor, r *client.Response) {
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
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	defer leaktest.Check(t)()

	geziyor.NewGeziyor(&geziyor.Options{
		AllowedDomains: []string{"books.toscrape.com"},
		StartURLs:      []string{"http://books.toscrape.com/"},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			g.Exports <- []string{r.Request.URL.String()}
			r.HTMLDoc.Find("a").Each(func(i int, s *goquery.Selection) {
				if href, ok := s.Attr("href"); ok {
					g.Get(r.JoinURL(href), g.Opt.ParseFunc)
				}
			})
		},
		Exporters:   []geziyor.Exporter{&export.CSV{}},
		MetricsType: metrics.Prometheus,
	}).Start()
}

func TestStartRequestsFunc(t *testing.T) {
	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			g.Get("http://quotes.toscrape.com/", g.Opt.ParseFunc)
		},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			r.HTMLDoc.Find("a").Each(func(_ int, s *goquery.Selection) {
				g.Exports <- s.AttrOr("href", "")
			})
		},
		Exporters: []geziyor.Exporter{&export.JSON{}},
	}).Start()
}

func TestGetRendered(t *testing.T) {
	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			g.GetRendered("https://httpbin.org/anything", g.Opt.ParseFunc)
		},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
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
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			fmt.Println(string(r.Body))
		},
	}).Start()
}

func TestCookies(t *testing.T) {
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"http://quotes.toscrape.com/login"},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			if len(g.Client.Cookies(r.Request.URL.String())) == 0 {
				t.Fatal("Cookies is Empty")
			}
		},
	}).Start()

	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"http://quotes.toscrape.com/login"},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			if len(g.Client.Cookies(r.Request.URL.String())) != 0 {
				t.Fatal("Cookies exist")
			}
		},
		CookiesDisabled: true,
	}).Start()
}

func TestBasicAuth(t *testing.T) {
	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			req, _ := client.NewRequest("GET", "https://httpbin.org/anything", nil)
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
			&extract.HTML{Name: "entry_html", Selector: ".c-entry-hero__content"},
			&extract.Text{Name: "title", Selector: ".c-page-title"},
			&extract.OuterHTML{Name: "title_html", Selector: ".c-page-title"},
			&extract.Text{Name: "author", Selector: ".c-byline__item:nth-child(1) > a"},
			&extract.Attr{Name: "author_url", Selector: ".c-byline__item:nth-child(1) > a", Attr: "href"},
			&extract.Text{Name: "summary", Selector: ".c-entry-summary"},
			&extract.Text{Name: "content", Selector: ".c-entry-content"},
		},
		Exporters: []geziyor.Exporter{&export.JSON{}},
	}).Start()
}

func TestCharsetDetection(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "\xf0ültekin")
	}))
	defer ts.Close()

	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{ts.URL},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			if !utf8.Valid(r.Body) {
				t.Fatal()
			}
		},
		CharsetDetectDisabled: false,
	}).Start()

	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{ts.URL},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			if utf8.Valid(r.Body) {
				t.Fatal()
			}
		},
		CharsetDetectDisabled: true,
	}).Start()
}

// Make sure to increase open file descriptor limits before running
func BenchmarkGeziyor_Do(b *testing.B) {

	// Create Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, client")
	}))
	ts.Client().Transport = client.NewClient().Transport
	defer ts.Close()

	// As we don't benchmark creating a server, reset timer.
	b.ResetTimer()

	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			// Create Synchronized request to benchmark requests accurately.
			req, _ := client.NewRequest("GET", ts.URL, nil)
			req.Synchronized = true

			// We only bench here !
			for i := 0; i < b.N; i++ {
				g.Do(req, nil)
			}
		},
		URLRevisitEnabled: true,
		LogDisabled:       true,
	}).Start()
}
