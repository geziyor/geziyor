package geziyor

import (
	"github.com/chromedp/chromedp"
	"github.com/geziyor/geziyor/cache"
	"github.com/geziyor/geziyor/client"
	"github.com/geziyor/geziyor/export"
	"github.com/geziyor/geziyor/metrics"
	"github.com/geziyor/geziyor/middleware"
	"net/http"
	"net/url"
	"time"
)

// Options is custom options type for Geziyor
type Options struct {
	// AllowedDomains is domains that are allowed to make requests
	// If empty, any domain is allowed
	AllowedDomains []string

	// Chrome headless browser WS endpoint.
	// If you want to run your own Chrome browser runner, provide its endpoint in here
	// For example: ws://localhost:3000
	BrowserEndpoint string

	// Cache storage backends.
	// - Memory
	// - Disk
	// - LevelDB
	Cache cache.Cache

	// Policies for caching.
	// - Dummy policy (default)
	// - RFC2616 policy
	CachePolicy cache.Policy

	// Response charset detection for decoding to UTF-8
	CharsetDetectDisabled bool

	// Concurrent requests limit
	ConcurrentRequests int

	// Concurrent requests per domain limit. Uses request.URL.Host
	// Subdomains are different than top domain
	ConcurrentRequestsPerDomain int

	// If set true, cookies won't send.
	CookiesDisabled bool

	// ErrorFunc is callback of errors.
	// If not defined, all errors will be logged.
	ErrorFunc func(g *Geziyor, r *client.Request, err error)

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

	// ProxyFunc setting proxy for each request
	ProxyFunc func(*http.Request) (*url.URL, error)

	// Rendered requests pre actions. Setting this will override the existing default.
	// And you'll need to handle all rendered actions, like navigation, waiting, response etc.
	// If you need to make custom actions in addition to the defaults, use Request.Actions instead of this.
	PreActions []chromedp.Action

	// Request delays
	RequestDelay time.Duration

	// RequestDelayRandomize uses random interval between 0.5 * RequestDelay and 1.5 * RequestDelay
	RequestDelayRandomize bool

	// Called before requests made to manipulate requests
	RequestMiddlewares []middleware.RequestProcessor

	// Called after response received
	ResponseMiddlewares []middleware.ResponseProcessor

	// RequestsPerSecond limits requests that is made per seconds. Default: No limit
	RequestsPerSecond float64

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
