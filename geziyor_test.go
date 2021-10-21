package geziyor_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/elazarl/goproxy"
	"github.com/fortytw2/leaktest"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/cache"
	"github.com/geziyor/geziyor/cache/diskcache"
	"github.com/geziyor/geziyor/client"
	"github.com/geziyor/geziyor/export"
	"github.com/geziyor/geziyor/internal"
	"github.com/geziyor/geziyor/metrics"
	"github.com/stretchr/testify/assert"
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

func TestUserAgent(t *testing.T) {
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"https://httpbin.org/anything"},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			var data map[string]interface{}
			err := json.Unmarshal(r.Body, &data)

			assert.NoError(t, err)
			assert.Equal(t, client.DefaultUserAgent, data["headers"].(map[string]interface{})["User-Agent"])
		},
	}).Start()
}

func TestCache(t *testing.T) {
	defer leaktest.Check(t)()
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"http://api.ipify.org"},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			fmt.Println(string(r.Body))
			g.Exports <- string(r.Body)
			g.Get("http://api.ipify.org", nil)
		},
		Cache:       diskcache.New(".cache"),
		CachePolicy: cache.RFC2616,
	}).Start()
}

func TestQuotes(t *testing.T) {
	defer leaktest.Check(t)()
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"http://quotes.toscrape.com/"},
		ParseFunc: quotesParse,
		Exporters: []export.Exporter{&export.JSONLine{FileName: "1.jsonl"}, &export.JSON{FileName: "2.json"}},
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
		absoluteURL, _ := r.Request.URL.Parse(href)
		g.Get(absoluteURL.String(), quotesParse)
	}
}

func TestAllLinks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	geziyor.NewGeziyor(&geziyor.Options{
		AllowedDomains: []string{"books.toscrape.com"},
		StartURLs:      []string{"http://books.toscrape.com/"},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			g.Exports <- []string{r.Request.URL.String()}
			r.HTMLDoc.Find("a").Each(func(i int, s *goquery.Selection) {
				if href, ok := s.Attr("href"); ok {
					absoluteURL, _ := r.Request.URL.Parse(href)
					g.Get(absoluteURL.String(), g.Opt.ParseFunc)
				}
			})
		},
		Exporters:   []export.Exporter{&export.CSV{}},
		MetricsType: metrics.Prometheus,
	}).Start()
}

func TestStartRequestsFunc(t *testing.T) {
	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			g.Get("http://quotes.toscrape.com/", nil)
		},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			r.HTMLDoc.Find("a").Each(func(_ int, s *goquery.Selection) {
				g.Exports <- s.AttrOr("href", "")
			})
		},
		Exporters: []export.Exporter{&export.JSON{}},
	}).Start()
}

func TestGetRendered(t *testing.T) {
	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			g.GetRendered("https://httpbin.org/anything", g.Opt.ParseFunc)
		},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			fmt.Println(string(r.Body))
			fmt.Println(r.Request.URL.String(), r.Header)
		},
		//URLRevisitEnabled: true,
	}).Start()
}

func TestGetRenderedCookie(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.Header.Get("Cookie")))
	}))
	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			req, err := client.NewRequest("GET", testServer.URL, nil)
			if err != nil {
				internal.Logger.Printf("Request creating error %v\n", err)
				return
			}
			req.Header.Set("Cookie", "key=value")
			req.Rendered = true
			g.Do(req, g.Opt.ParseFunc)
		},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			assert.Contains(t, string(r.Body), "key=value")
		},
	}).Start()
}

// Run chrome headless instance to test this
//func TestGetRenderedRemoteAllocator(t *testing.T) {
//	geziyor.NewGeziyor(&geziyor.Options{
//		StartRequestsFunc: func(g *geziyor.Geziyor) {
//			g.GetRendered("https://httpbin.org/anything", g.Opt.ParseFunc)
//		},
//		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
//			fmt.Println(string(r.Body))
//			fmt.Println(r.Request.URL.String(), r.Header)
//		},
//		BrowserEndpoint: "ws://localhost:3000",
//	}).Start()
//}

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

type PostBody struct {
	UserName string `json:"user_name"`
	Message  string `json:"message"`
}

func TestPostJson(t *testing.T) {
	postBody := &PostBody{
		UserName: "Juan Valdez",
		Message:  "Best coffee in town",
	}
	payloadBuf := new(bytes.Buffer)
	json.NewEncoder(payloadBuf).Encode(postBody)

	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			g.Post("https://reqbin.com/echo/post/json", payloadBuf, g.Opt.ParseFunc)
		},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			fmt.Println(string(r.Body))
			g.Exports <- string(r.Body)
		},
		Exporters: []export.Exporter{&export.JSON{FileName: "post_json.json"}},
	}).Start()
}

func TestPostFormUrlEncoded(t *testing.T) {
	postForm := url.Values{}
	postForm.Set("user_name", "Juan Valdez")
	postForm.Set("message", "Enjoy a good coffee!")

	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			g.Post("https://reqbin.com/echo/post/form", strings.NewReader(postForm.Encode()), g.Opt.ParseFunc)
		},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			fmt.Println(string(r.Body))
			g.Exports <- map[string]interface{}{
				"host":            r.Request.Host,
				"h1":              r.HTMLDoc.Find("h1").Text(),
				"entire_response": string(r.Body),
			}
		},
		Exporters: []export.Exporter{&export.JSON{FileName: "post_form.json"}},
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

func TestRedirect(t *testing.T) {
	defer leaktest.Check(t)()
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"https://httpbin.org/absolute-redirect/1"},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			//t.Fail()
		},
		MaxRedirect: -1,
	}).Start()

	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"https://httpbin.org/absolute-redirect/1"},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			if r.StatusCode == 302 {
				t.Fail()
			}
		},
		MaxRedirect: 0,
	}).Start()
}

func TestConcurrentRequests(t *testing.T) {
	defer leaktest.Check(t)()
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs:                   []string{"https://httpbin.org/delay/1", "https://httpbin.org/delay/2"},
		ConcurrentRequests:          1,
		ConcurrentRequestsPerDomain: 1,
	}).Start()
}

func TestRobots(t *testing.T) {
	defer leaktest.Check(t)()
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{"https://httpbin.org/deny"},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			t.Error("/deny should be blocked by robots.txt middleware")
		},
	}).Start()
}

func TestPassMetadata(t *testing.T) {
	geziyor.NewGeziyor(&geziyor.Options{
		StartRequestsFunc: func(g *geziyor.Geziyor) {
			req, _ := client.NewRequest("GET", "https://httpbin.org/anything", nil)
			req.Meta["key"] = "value"
			g.Do(req, g.Opt.ParseFunc)
		},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			assert.Equal(t, r.Request.Meta["key"], "value")
		},
	}).Start()
}

func TestProxy(t *testing.T) {
	// Setup fake proxy server
	testHeaderKey := "Geziyor"
	testHeaderVal := "value"
	proxy := goproxy.NewProxyHttpServer()
	proxy.OnRequest().DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		r.Header.Set(testHeaderKey, testHeaderVal)
		return r, nil
	})
	ts := httptest.NewServer(proxy)
	defer ts.Close()

	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs:         []string{"http://httpbin.org/anything"},
		ProxyFunc:         client.RoundRobinProxy(ts.URL),
		RobotsTxtDisabled: true,
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			var data map[string]interface{}
			err := json.Unmarshal(r.Body, &data)
			assert.NoError(t, err)
			// Check header set
			assert.Equal(t, testHeaderVal, data["headers"].(map[string]interface{})[testHeaderKey])
		},
	}).Start()
}

// Make sure to increase open file descriptor limits before running
func BenchmarkRequests(b *testing.B) {

	// Create Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, client")
	}))
	ts.Client().Transport = client.NewClient(&client.Options{
		MaxBodySize:    client.DefaultMaxBody,
		RetryTimes:     client.DefaultRetryTimes,
		RetryHTTPCodes: client.DefaultRetryHTTPCodes,
	}).Transport
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

func BenchmarkWhole(b *testing.B) {
	for i := 0; i < b.N; i++ {
		geziyor.NewGeziyor(&geziyor.Options{
			AllowedDomains: []string{"quotes.toscrape.com"},
			StartURLs:      []string{"http://quotes.toscrape.com/"},
			ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
				g.Exports <- []string{r.Request.URL.String()}
				r.HTMLDoc.Find("a").Each(func(i int, s *goquery.Selection) {
					if href, ok := s.Attr("href"); ok {
						absoluteURL, _ := r.Request.URL.Parse(href)
						g.Get(absoluteURL.String(), g.Opt.ParseFunc)
					}
				})
			},
			Exporters: []export.Exporter{&export.CSV{}},
			//MetricsType: metrics.Prometheus,
			LogDisabled: true,
		}).Start()
	}
}
