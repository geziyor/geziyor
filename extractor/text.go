package extractor

import "github.com/PuerkitoBio/goquery"

// Text extracts texts from selected nodes
type Text struct {
	Name     string
	Selector string
}

// Extract extracts texts from selected nodes
func (e *Text) Extract(doc *goquery.Document) interface{} {
	return map[string]string{e.Name: doc.Find(e.Selector).Text()}
}
