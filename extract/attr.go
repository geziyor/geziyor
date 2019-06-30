package extract

import (
	"errors"
	"github.com/PuerkitoBio/goquery"
)

var ErrAttrNotExists = errors.New("attribute not exist")

// Attr returns HTML attribute value of provided selector
type Attr struct {
	Name     string
	Selector string
	Attr     string
}

// Extract returns HTML attribute value of provided selector
func (e *Attr) Extract(doc *goquery.Document) (interface{}, error) {
	attr, exists := doc.Find(e.Selector).Attr(e.Attr)
	if !exists {
		return nil, ErrAttrNotExists
	}
	return map[string]string{e.Name: attr}, nil
}
