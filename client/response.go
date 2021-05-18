package client

import (
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"net/url"
	"strings"
)

// Response type wraps http.Response
// Contains parsed response data and Geziyor functions.
type Response struct {
	*http.Response

	// Response body
	Body []byte

	// Goquery Document object. If response IsHTML, its non-nil.
	HTMLDoc *goquery.Document

	Request *Request
}

// JoinURL joins base response URL and provided relative URL.
func (r *Response) JoinURL(relativeURL string) (*url.URL, error) {
	joinedURL, err := r.Request.URL.Parse(relativeURL)
	if err != nil {
		return nil, err
	}
	return joinedURL, nil
}

// IsHTML checks if response content is HTML by looking content-type header
func (r *Response) IsHTML() bool {
	contentType := r.Header.Get("Content-Type")
	for _, htmlContentType := range []string{"text/html", "application/xhtml+xml", "application/vnd.wap.xhtml+xml"} {
		if strings.Contains(contentType, htmlContentType) {
			return true
		}
	}
	return false
}
