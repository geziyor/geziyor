package gezer

import (
	"fmt"
	"testing"
)

func TestGezer_StartURLs_Simple(t *testing.T) {
	gezer := NewGezer(parse, "https://api.ipify.org", "https://api.ipify.org")
	gezer.Start()
}

func parse(response *Response) {
	fmt.Println(string(response.Body))
}

//func TestGezer_StartURLs_HTML(t *testing.T) {
//	gezer := NewGezer(parse, "http://quotes.toscrape.com/")
//	gezer.Start()
//	for result := range gezer.Results {
//		result.Doc.Find("div.quote").Each(func(_ int, s *goquery.Selection) {
//			fmt.Println(s.Find("span.text").Text())
//			fmt.Println(s.Find("small.author").Text())
//		})
//	}
//}
