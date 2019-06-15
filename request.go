package geziyor

import (
	"net/http"
)

// Request is a small wrapper around *http.Request that contains Metadata and Rendering option
type Request struct {
	*http.Request
	Meta     map[string]interface{}
	Rendered bool
}

func defaultHeadersMiddleware(g *Geziyor, r *Request) {
	r.Header = headerSetDefault(r.Header, "Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	r.Header = headerSetDefault(r.Header, "Accept-Charset", "utf-8")
	r.Header = headerSetDefault(r.Header, "Accept-Language", "en")
	r.Header = headerSetDefault(r.Header, "User-Agent", g.Opt.UserAgent)
}

func headerSetDefault(header http.Header, key string, value string) http.Header {
	if header.Get(key) == "" {
		header.Set(key, value)
	}
	return header
}
