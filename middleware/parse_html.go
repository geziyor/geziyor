package middleware

import (
	"bytes"
	"github.com/PuerkitoBio/goquery"
	"github.com/geziyor/geziyor/client"
)

// ParseHTML parses response if response is HTML
type ParseHTML struct {
	ParseHTMLDisabled bool
}

func (p *ParseHTML) ProcessResponse(r *client.Response) {
	if !p.ParseHTMLDisabled && r.IsHTML() {
		r.HTMLDoc, _ = goquery.NewDocumentFromReader(bytes.NewReader(r.Body))
	}
}
