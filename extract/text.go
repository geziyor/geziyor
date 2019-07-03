package extract

import (
	"github.com/PuerkitoBio/goquery"
	"strings"
)

// Text returns the combined text contents of provided selector.
type Text struct {
	Name      string
	Selector  string
	TrimSpace bool
}

// Extract returns the combined text contents of provided selector.
func (e Text) Extract(sel *goquery.Selection) (interface{}, error) {
	text := sel.Find(e.Selector).Text()
	if e.TrimSpace {
		text = strings.TrimSpace(text)
	}
	return map[string]string{e.Name: text}, nil
}
