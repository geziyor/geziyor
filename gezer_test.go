package gezer

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fpfeng/httpcache"
	"testing"
)

func TestGezer_StartURLs_Simple(t *testing.T) {
	gezer := NewGezer(Opt{
		StartURLs: []string{"http://api.ipify.org"},
		Cache:     httpcache.NewMemoryCache(),
		ParseFunc: func(r *Response) {
			fmt.Println(string(r.Body))
			r.Gezer.Get("http://api.ipify.org")
		},
	})
	gezer.Start()
}

func TestGezer_StartURLs_HTML(t *testing.T) {
	gezer := NewGezer(Opt{
		StartURLs: []string{"http://quotes.toscrape.com/"},
		ParseFunc: func(r *Response) {
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
				go r.Gezer.Get(r.JoinURL(href))
			}
		},
	})
	gezer.Start()
}

func TestGezer_Concurrent_Requests(t *testing.T) {
	gezer := NewGezer(Opt{
		AllowedDomains: []string{"quotes.toscrape.com"},
		StartURLs:      []string{"http://quotes.toscrape.com/"},
		ParseFunc: func(r *Response) {
			//r.Exports <- map[string]interface{}{"href": r.Request.URL.String()}
			r.Doc.Find("a").Each(func(i int, s *goquery.Selection) {
				if href, ok := s.Attr("href"); ok {
					go r.Gezer.Get(r.JoinURL(href))
				}
			})
		},
	})
	gezer.Start()
}
