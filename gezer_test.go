package gezer

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"testing"
)

func TestGezer_StartURLs_Simple(t *testing.T) {
	gezer := NewGezer(Opt{
		StartURLs: []string{"https://api.ipify.org", "https://api.ipify.org"},
		ParseFunc: func(r *Response) {
			fmt.Println(string(r.Body))
		},
	})
	gezer.Start()
}

func TestGezer_StartURLs_HTML(t *testing.T) {
	gezer := NewGezer(Opt{
		StartURLs: []string{"http://quotes.toscrape.com/"},
		ParseFunc: func(r *Response) {
			r.Doc.Find("div.quote").Each(func(i int, s *goquery.Selection) {
				fmt.Println(i, s.Find("span.text").Text(), s.Find("small.author").Text())
			})
			if href, ok := r.Doc.Find("li.next > a").Attr("href"); ok {
				r.Gezer.Get(r.JoinURL(href))
			}
		},
	})
	gezer.Start()
}
