package geziyor

import (
	"net/http"
)

// Request is a small wrapper around *http.Request that contains Metadata and Rendering option
type Request struct {
	*http.Request
	Meta      map[string]interface{}
	Rendered  bool
	Cancelled bool
}

func allowedDomainsMiddleware(g *Geziyor, r *Request) {
	if len(g.Opt.AllowedDomains) != 0 && !contains(g.Opt.AllowedDomains, r.Host) {
		//log.Printf("Domain not allowed: %s\n", req.Host)
		r.Cancelled = true
		return
	}
}

func duplicateRequestsMiddleware(g *Geziyor, r *Request) {
	if !g.Opt.URLRevisitEnabled {
		if _, visited := g.visitedURLs.LoadOrStore(r.Request.URL.String(), struct{}{}); visited {
			//log.Printf("URL already visited %s\n", rawURL)
			r.Cancelled = true
		}
	}
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

// contains checks whether []string contains string
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
