package extract

import "github.com/PuerkitoBio/goquery"

// Text returns the combined text contents of provided selector.
type Text struct {
	Name     string
	Selector string
}

// Extract returns the combined text contents of provided selector.
func (e *Text) Extract(doc *goquery.Document) (interface{}, error) {
	return map[string]string{e.Name: doc.Find(e.Selector).Text()}, nil
}
