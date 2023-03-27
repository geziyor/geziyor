package middleware

import (
	"bytes"
	"github.com/PuerkitoBio/goquery"
	"github.com/hohner2008/geziyor/client"
	"github.com/hohner2008/geziyor/internal"
)

// ParseHTML parses response if response is HTML
type ParseHTML struct {
	ParseHTMLDisabled bool
}

func (p *ParseHTML) ProcessResponse(r *client.Response) {
	if !p.ParseHTMLDisabled && r.IsHTML() {
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(r.Body))
		if err != nil {
			internal.Logger.Println(err.Error())
			return
		}
		r.HTMLDoc = doc
	}
}
