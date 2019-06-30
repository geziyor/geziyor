package geziyor

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/geziyor/geziyor/client"
	"github.com/geziyor/geziyor/internal"
	"log"
	"math/rand"
	"os"
	"reflect"
	"runtime/debug"
	"time"
)

// RequestMiddleware called before requests made.
// Set request.Cancelled = true to cancel request
type RequestMiddleware func(g *Geziyor, r *client.Request)

// ResponseMiddleware called after request response receive
type ResponseMiddleware func(g *Geziyor, r *client.Response)

func init() {
	log.SetOutput(os.Stdout)
	rand.Seed(time.Now().UnixNano())
}

// recoverMiddleware recovers scraping being crashed.
// Logs error and stack trace
func recoverMiddleware(g *Geziyor, r *client.Request) {
	if r := recover(); r != nil {
		log.Println(r, string(debug.Stack()))
		g.metrics.PanicCounter.Add(1)
	}
}

// allowedDomainsMiddleware checks for request host if it exists in AllowedDomains
func allowedDomainsMiddleware(g *Geziyor, r *client.Request) {
	if len(g.Opt.AllowedDomains) != 0 && !internal.Contains(g.Opt.AllowedDomains, r.Host) {
		//log.Printf("Domain not allowed: %s\n", req.Host)
		r.Cancel()
		return
	}
}

// duplicateRequestsMiddleware checks for already visited URLs
func duplicateRequestsMiddleware(g *Geziyor, r *client.Request) {
	if !g.Opt.URLRevisitEnabled {
		key := r.Request.URL.String() + r.Request.Method
		if _, visited := g.visitedURLs.LoadOrStore(key, struct{}{}); visited {
			//log.Printf("URL already visited %s\n", rawURL)
			r.Cancel()
		}
	}
}

// defaultHeadersMiddleware sets default request headers
func defaultHeadersMiddleware(g *Geziyor, r *client.Request) {
	r.Header = client.SetDefaultHeader(r.Header, "Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	r.Header = client.SetDefaultHeader(r.Header, "Accept-Charset", "utf-8")
	r.Header = client.SetDefaultHeader(r.Header, "Accept-Language", "en")
	r.Header = client.SetDefaultHeader(r.Header, "User-Agent", g.Opt.UserAgent)
}

// delayMiddleware delays requests
func delayMiddleware(g *Geziyor, r *client.Request) {
	if g.Opt.RequestDelayRandomize {
		min := float64(g.Opt.RequestDelay) * 0.5
		max := float64(g.Opt.RequestDelay) * 1.5
		time.Sleep(time.Duration(rand.Intn(int(max-min)) + int(min)))
	} else {
		time.Sleep(g.Opt.RequestDelay)
	}
}

// logMiddleware logs requests
func logMiddleware(g *Geziyor, r *client.Request) {
	// LogDisabled check is not necessary, but done here for performance reasons
	if !g.Opt.LogDisabled {
		log.Println("Fetching: ", r.URL.String())
	}
}

// metricsRequestMiddleware sets stats
func metricsRequestMiddleware(g *Geziyor, r *client.Request) {
	g.metrics.RequestCounter.With("method", r.Method).Add(1)
}

// parseHTMLMiddleware parses response if response is HTML
func parseHTMLMiddleware(g *Geziyor, r *client.Response) {
	if !g.Opt.ParseHTMLDisabled && r.IsHTML() {
		r.HTMLDoc, _ = goquery.NewDocumentFromReader(bytes.NewReader(r.Body))
	}
}

// metricsResponseMiddleware sets stats
func metricsResponseMiddleware(g *Geziyor, r *client.Response) {
	g.metrics.ResponseCounter.With("method", r.Request.Method).Add(1)
}

// extractorsMiddleware extracts data from loaders conf and exports it to exporters
func extractorsMiddleware(g *Geziyor, r *client.Response) {

	// Check if we have extractors and exporters
	if len(g.Opt.Extractors) != 0 && len(g.Opt.Exporters) != 0 {
		exports := map[string]interface{}{}

		for _, extractor := range g.Opt.Extractors {
			extracted := extractor.Extract(r.HTMLDoc)

			// Check extracted data type and use it accordingly
			val := reflect.ValueOf(extracted)
			switch val.Kind() {
			case reflect.Map:
				r := val.MapRange()
				for r.Next() {
					exports[fmt.Sprint(r.Key())] = r.Value().Interface()
				}
			}
		}
		g.Exports <- exports
	}
}
