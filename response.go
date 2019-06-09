package geziyor

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
	Body    []byte
	DocHTML *goquery.Document

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

func (r *Response) isHTML() bool {
	contentType := r.Header.Get("Content-Type")
	for _, htmlContentType := range []string{"text/html", "application/xhtml+xml", "application/vnd.wap.xhtml+xml"} {
		if strings.Contains(contentType, htmlContentType) {
			return true
		}
	}
	return false
}
