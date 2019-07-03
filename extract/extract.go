package extract

import "github.com/PuerkitoBio/goquery"

// Extractor interface is for extracting data from HTML document
type Extractor interface {
	Extract(sel *goquery.Selection) (interface{}, error)
}
