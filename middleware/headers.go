package middleware

import (
	"github.com/hohner2008/geziyor/client"
)

// Headers sets default request headers
type Headers struct {
	UserAgent string
}

func (a *Headers) ProcessRequest(r *client.Request) {
	r.Header = client.SetDefaultHeader(r.Header, "Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	r.Header = client.SetDefaultHeader(r.Header, "Accept-Charset", "utf-8")
	r.Header = client.SetDefaultHeader(r.Header, "Accept-Language", "en")
	r.Header = client.SetDefaultHeader(r.Header, "User-Agent", a.UserAgent)
}
