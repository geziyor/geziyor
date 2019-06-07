package gezer

import (
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"net/url"
)

type Response struct {
	*http.Response
	Body []byte
	Doc  *goquery.Document

	Gezer *Gezer
}

func (r *Response) JoinURL(relativeURL string) string {
	parsedRelativeURL, err := url.Parse(relativeURL)
	if err != nil {
		return ""
	}

	joinedURL := r.Response.Request.URL.ResolveReference(parsedRelativeURL)
	return joinedURL.String()
}
