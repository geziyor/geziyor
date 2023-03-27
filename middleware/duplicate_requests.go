package middleware

import (
	"github.com/hohner2008/geziyor/client"
	"github.com/hohner2008/geziyor/internal"
	"sync"
)

// DuplicateRequests checks for already visited URLs
type DuplicateRequests struct {
	RevisitEnabled bool
	visitedURLs    sync.Map
	logOnlyOnce    sync.Map
}

func (a *DuplicateRequests) ProcessRequest(r *client.Request) {
	if !a.RevisitEnabled && r.Request.Method == "GET" {
		requestURL := r.Request.URL.String()
		if _, visited := a.visitedURLs.LoadOrStore(requestURL, struct{}{}); visited {
			if _, logged := a.logOnlyOnce.LoadOrStore(requestURL, struct{}{}); !logged {
				internal.Logger.Printf("URL already visited %s\n", requestURL)
			}
			r.Cancel()
		}
	}
}
