package geziyor_test

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fpfeng/httpcache"
	"github.com/geziyor/geziyor"
	"testing"
)

func TestGeziyor_Simple(t *testing.T) {
	geziyor.NewGeziyor(geziyor.Options{
		StartURLs: []string{"http://api.ipify.org"},
		ParseFunc: func(r *geziyor.Response) {
			fmt.Println(r.Doc.Text())
		},
	}).Start()
}

func TestGeziyor_IP(t *testing.T) {
	gez := geziyor.NewGeziyor(geziyor.Options{
		StartURLs: []string{"http://api.ipify.org"},
		Cache:     httpcache.NewMemoryCache(),
		ParseFunc: func(r *geziyor.Response) {
			fmt.Println(string(r.Body))
			r.Geziyor.Get("http://api.ipify.org")
		},
	})
	gez.Start()
}

func TestGeziyor_HTML(t *testing.T) {
	gez := geziyor.NewGeziyor(geziyor.Options{
		StartURLs: []string{"http://quotes.toscrape.com/"},
		ParseFunc: func(r *geziyor.Response) {
			r.Doc.Find("div.quote").Each(func(i int, s *goquery.Selection) {
				// Export Data
				r.Exports <- map[string]interface{}{
					"text":   s.Find("span.text").Text(),
					"author": s.Find("small.author").Text(),
					"tags": s.Find("div.tags > a.tag").Map(func(_ int, s *goquery.Selection) string {
						return s.Text()
					}),
				}
			})

			// Next Page
			if href, ok := r.Doc.Find("li.next > a").Attr("href"); ok {
				go r.Geziyor.Get(r.JoinURL(href))
			}
		},
	})
	gez.Start()
}

func TestGeziyor_Concurrent_Requests(t *testing.T) {
	gez := geziyor.NewGeziyor(geziyor.Options{
		AllowedDomains: []string{"quotes.toscrape.com"},
		StartURLs:      []string{"http://quotes.toscrape.com/"},
		ParseFunc: func(r *geziyor.Response) {
			//r.Exports <- map[string]interface{}{"href": r.Request.URL.String()}
			r.Doc.Find("a").Each(func(i int, s *goquery.Selection) {
				if href, ok := s.Attr("href"); ok {
					go r.Geziyor.Get(r.JoinURL(href))
				}
			})
		},
	})
	gez.Start()
}
