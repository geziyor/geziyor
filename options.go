package geziyor

import (
	"github.com/fpfeng/httpcache"
	"github.com/geziyor/geziyor/client"
	"github.com/geziyor/geziyor/export"
	"github.com/geziyor/geziyor/metrics"
	"github.com/geziyor/geziyor/middleware"
	"time"
)

// Options is custom options type for Geziyor
type Options struct {
	// AllowedDomains is domains that are allowed to make requests
	// If empty, any domain is allowed
	AllowedDomains []string

	// Set this to enable caching responses.
	// Memory Cache: httpcache.NewMemoryCache()
	// Disk Cache:   diskcache.New(".cache")
	Cache httpcache.Cache

	// Charset Detection disable
	CharsetDetectDisabled bool

	// Concurrent requests limit
	ConcurrentRequests int

	// Concurrent requests per domain limit
	ConcurrentRequestsPerDomain int

	// If set true, cookies won't send.
	CookiesDisabled bool

	// For extracting data
	Exporters []export.Exporter

	// Disable logging by setting this true
	LogDisabled bool

	// Max body reading size in bytes. Default: 1GB
	MaxBodySize int64

	// Maximum redirection time. Default: 10
	MaxRedirect int

	// Scraper metrics exporting type. See metrics.Type
	MetricsType metrics.Type

	// ParseFunc is callback of StartURLs response.
	ParseFunc func(g *Geziyor, r *client.Response)

	// If true, HTML parsing is disabled to improve performance.
	ParseHTMLDisabled bool

	// Request delays
	RequestDelay time.Duration

	// RequestDelayRandomize uses random interval between 0.5 * RequestDelay and 1.5 * RequestDelay
	RequestDelayRandomize bool

	// Called before requests made to manipulate requests
	RequestMiddlewares []middleware.RequestProcessor

	// Called after response received
	ResponseMiddlewares []middleware.ResponseProcessor

	// Which HTTP response codes to retry.
	// Other errors (DNS lookup issues, connections lost, etc) are always retried.
	// Default: []int{500, 502, 503, 504, 522, 524, 408}
	RetryHTTPCodes []int

	// Maximum number of times to retry, in addition to the first download.
	// Set -1 to disable retrying
	// Default: 2
	RetryTimes int

	// If true, disable robots.txt checks
	RobotsTxtDisabled bool

	// StartRequestsFunc called on scraper start
	StartRequestsFunc func(g *Geziyor)

	// First requests will made to this url array. (Concurrently)
	StartURLs []string

	// Timeout is global request timeout
	Timeout time.Duration

	// Revisiting same URLs is disabled by default
	URLRevisitEnabled bool

	// User Agent.
	// Default: "Geziyor 1.0"
	UserAgent string
}
