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

	// StartRequestsFunc called on scraper start
	StartRequestsFunc func(g *Geziyor)

	// ParseFunc is callback of StartURLs response.
	ParseFunc func(g *Geziyor, r *Response)

	// Timeout is global request timeout
	Timeout time.Duration

	// Set this to enable caching responses.
	// Memory Cache: httpcache.NewMemoryCache()
	// Disk Cache:   diskcache.New(".cache")
	Cache httpcache.Cache

	// Concurrent requests limit
	ConcurrentRequests int
	// Concurrent requests per domain limit
	ConcurrentRequestsPerDomain int

	// User Agent. Default: "Geziyor 1.0"
	UserAgent string

	// Request delays
	RequestDelay time.Duration
	// RequestDelayRandomize uses random interval between 0.5 * RequestDelay and 1.5 * RequestDelay
	RequestDelayRandomize bool

	// Disable logging by setting this true
	LogDisabled bool

	// For extracting data
	Exporters []Exporter

	// Called before requests made to manipulate requests
	RequestMiddlewares []RequestMiddleware

	// Max body reading size in bytes. Default: 1GB
	MaxBodySize int64

	// Charset Detection disable
	CharsetDetectDisabled bool

	// If true, HTML parsing is disabled to improve performance.
	ParseHTMLDisabled bool

	// Revisiting same URLs is disabled by default
	URLRevisitEnabled bool
}
