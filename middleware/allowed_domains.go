package middleware

import (
	"github.com/geziyor/geziyor/client"
	"github.com/geziyor/geziyor/internal"
)

// AllowedDomains checks for request host if it exists in AllowedDomains
type AllowedDomains struct {
	AllowedDomains []string
}

func (a *AllowedDomains) ProcessRequest(r *client.Request) {
	if len(a.AllowedDomains) != 0 && !internal.Contains(a.AllowedDomains, r.Host) {
		//log.Printf("Domain not allowed: %s\n", req.Host)
		r.Cancel()
		return
	}
}
