package middleware

import (
	"github.com/geziyor/geziyor/client"
	"github.com/geziyor/geziyor/internal"
	"log"
	"sync"
)

// AllowedDomains checks for request host if it exists in AllowedDomains
type AllowedDomains struct {
	AllowedDomains []string
	logOnlyOnce    sync.Map
}

func (a *AllowedDomains) ProcessRequest(r *client.Request) {
	if len(a.AllowedDomains) != 0 && !internal.Contains(a.AllowedDomains, r.Host) {
		if _, logged := a.logOnlyOnce.LoadOrStore(r.Host, struct{}{}); !logged {
			log.Printf("Domain not allowed: %s\n", r.Host)
		}
		r.Cancel()
		return
	}
}
