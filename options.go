package geziyor

import (
	"github.com/fpfeng/httpcache"
	"time"
)

// Options is custom options type for Geziyor
type Options struct {
	// AllowedDomains is domains that are allowed to make requests
	// If empty, any domain is allowed
	AllowedDomains []string
	// First requests will made to this url array. (Concurrently)
	StartURLs []string
	// ParseFunc is callback of StartURLs response.
	ParseFunc func(response *Response)
	// Timeout is global request timeout
	Timeout time.Duration

	// Set this to enable caching responses.
	// Memory Cache: httpcache.NewMemoryCache()
	// Disk Cache:   diskcache.New(".cache")
	Cache httpcache.Cache
}
