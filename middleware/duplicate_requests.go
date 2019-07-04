package middleware

import (
	"github.com/geziyor/geziyor/client"
	"sync"
)

// DuplicateRequests checks for already visited URLs
type DuplicateRequests struct {
	RevisitEnabled bool
	visitedURLs    sync.Map
}

func (a *DuplicateRequests) ProcessRequest(r *client.Request) {
	if !a.RevisitEnabled && r.Request.Method == "GET" {
		if _, visited := a.visitedURLs.LoadOrStore(r.Request.URL.String(), struct{}{}); visited {
			//log.Printf("URL already visited %s\n")
			r.Cancel()
		}
	}
}
