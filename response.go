package geziyor

import (
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"net/url"
)

// Response type wraps http.Response
// Contains parsed response data and Geziyor functions.
type Response struct {
	*http.Response
	Body []byte
	Doc  *goquery.Document

	Geziyor *Geziyor
	Exports chan interface{}
}

// JoinURL joins base response URL and provided relative URL.
func (r *Response) JoinURL(relativeURL string) string {
	parsedRelativeURL, err := url.Parse(relativeURL)
	if err != nil {
		return ""
	}

	joinedURL := r.Response.Request.URL.ResolveReference(parsedRelativeURL)
	return joinedURL.String()
}
