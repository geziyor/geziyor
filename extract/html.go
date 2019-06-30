package extract

import (
	"bytes"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// HTML extracts and returns the HTML from inside each element of the given selection.
type HTML struct {
	Name     string
	Selector string
}

// Extract extracts and returns the HTML from inside each element of the given selection.
func (e *HTML) Extract(doc *goquery.Document) (interface{}, error) {
	var ret, h string
	var err error

	doc.Find(e.Selector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		h, err = s.Html()
		if err != nil {
			return false
		}

		ret += h
		return true
	})

	if err != nil {
		return nil, err
	}

	return map[string]string{e.Name: ret}, nil
}

// OuterHTML extracts and returns the HTML of each element of the given selection.
type OuterHTML struct {
	Name     string
	Selector string
}

// Extract extracts and returns the HTML of each element of the given selection.
func (e *OuterHTML) Extract(doc *goquery.Document) (interface{}, error) {
	output := bytes.NewBufferString("")
	for _, node := range doc.Find(e.Selector).Nodes {
		if err := html.Render(output, node); err != nil {
			return nil, err
		}
	}

	return map[string]string{e.Name: output.String()}, nil
}
