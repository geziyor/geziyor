package geziyor

import (
	"bytes"
	"github.com/PuerkitoBio/goquery"
	"github.com/geziyor/geziyor/internal"
	"log"
	"runtime/debug"
)

// RequestMiddleware called before requests made.
// Set request.Cancelled = true to cancel request
type RequestMiddleware func(g *Geziyor, r *Request)

// ResponseMiddleware called after request response receive
type ResponseMiddleware func(g *Geziyor, r *Response)

// recoverMiddleware recovers scraping being crashed.
// Logs error and stack trace
func recoverMiddleware() {
	if r := recover(); r != nil {
		log.Println(r, string(debug.Stack()))
	}
}

// allowedDomainsMiddleware checks for request host if it exists in AllowedDomains
func allowedDomainsMiddleware(g *Geziyor, r *Request) {
	if len(g.Opt.AllowedDomains) != 0 && !internal.Contains(g.Opt.AllowedDomains, r.Host) {
		//log.Printf("Domain not allowed: %s\n", req.Host)
		r.Cancelled = true
		return
	}
}

// duplicateRequestsMiddleware checks for already visited URLs
func duplicateRequestsMiddleware(g *Geziyor, r *Request) {
	if !g.Opt.URLRevisitEnabled {
		key := r.Request.URL.String() + r.Request.Method
		if _, visited := g.visitedURLs.LoadOrStore(key, struct{}{}); visited {
			//log.Printf("URL already visited %s\n", rawURL)
			r.Cancelled = true
		}
	}
}

// defaultHeadersMiddleware sets default request headers
func defaultHeadersMiddleware(g *Geziyor, r *Request) {
	r.Header = internal.SetDefaultHeader(r.Header, "Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	r.Header = internal.SetDefaultHeader(r.Header, "Accept-Charset", "utf-8")
	r.Header = internal.SetDefaultHeader(r.Header, "Accept-Language", "en")
	r.Header = internal.SetDefaultHeader(r.Header, "User-Agent", g.Opt.UserAgent)
}

// parseHTMLMiddleware parses response if response is HTML
func parseHTMLMiddleware(g *Geziyor, r *Response) {
	if !g.Opt.ParseHTMLDisabled && r.isHTML() {
		r.DocHTML, _ = goquery.NewDocumentFromReader(bytes.NewReader(r.Body))
	}
}
