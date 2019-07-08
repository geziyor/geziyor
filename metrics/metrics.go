package metrics

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"
	"github.com/go-kit/kit/metrics/expvar"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

// Type represents metrics Types
type Type int

const (
	// Discard discards any metrics.
	Discard Type = iota

	// Prometheus starts server at :2112 and exports metrics data to /metrics
	Prometheus

	// ExpVar uses built-in expvar package
	ExpVar
)

// Metrics type stores metrics
type Metrics struct {
	RequestCounter            metrics.Counter
	ResponseCounter           metrics.Counter
	PanicCounter              metrics.Counter
	RobotsTxtRequestCounter   metrics.Counter
	RobotsTxtResponseCounter  metrics.Counter
	RobotsTxtForbiddenCounter metrics.Counter
}

// NewMetrics creates new metrics with given metrics.Type
func NewMetrics(metricsType Type) *Metrics {
	switch metricsType {
	case Discard:
		return &Metrics{
			RequestCounter:            discard.NewCounter(),
			ResponseCounter:           discard.NewCounter(),
			PanicCounter:              discard.NewCounter(),
			RobotsTxtRequestCounter:   discard.NewCounter(),
			RobotsTxtResponseCounter:  discard.NewCounter(),
			RobotsTxtForbiddenCounter: discard.NewCounter(),
		}
	case ExpVar:
		return &Metrics{
			RequestCounter:            expvar.NewCounter("request_count"),
			ResponseCounter:           expvar.NewCounter("response_count"),
			PanicCounter:              expvar.NewCounter("panic_count"),
			RobotsTxtRequestCounter:   expvar.NewCounter("robotstxt_request_count"),
			RobotsTxtResponseCounter:  expvar.NewCounter("robotstxt_response_count"),
			RobotsTxtForbiddenCounter: expvar.NewCounter("robotstxt_forbidden_count"),
		}
	case Prometheus:
		return &Metrics{
			RequestCounter: prometheus.NewCounterFrom(stdprometheus.CounterOpts{
				Namespace: "geziyor",
				Name:      "request_count",
				Help:      "Request count",
			}, []string{"method"}),
			ResponseCounter: prometheus.NewCounterFrom(stdprometheus.CounterOpts{
				Namespace: "geziyor",
				Name:      "response_count",
				Help:      "Response count",
			}, []string{"status"}),
			PanicCounter: prometheus.NewCounterFrom(stdprometheus.CounterOpts{
				Namespace: "geziyor",
				Name:      "panic_count",
				Help:      "Panic count",
			}, []string{}),
			RobotsTxtRequestCounter: prometheus.NewCounterFrom(stdprometheus.CounterOpts{
				Namespace: "geziyor",
				Name:      "robotstxt_request_count",
				Help:      "Robotstxt request count",
			}, []string{}),
			RobotsTxtResponseCounter: prometheus.NewCounterFrom(stdprometheus.CounterOpts{
				Namespace: "geziyor",
				Name:      "robotstxt_response_count",
				Help:      "Robotstxt response count",
			}, []string{"status"}),
			RobotsTxtForbiddenCounter: prometheus.NewCounterFrom(stdprometheus.CounterOpts{
				Namespace: "geziyor",
				Name:      "robotstxt_forbidden_count",
				Help:      "Robotstxt forbidden count",
			}, []string{"method"}),
		}
	default:
		return nil
	}
}

// StartMetricsServer starts server that handles metrics
// Prometheus: http://localhost:2112/metrics
// Expvar    : http://localhost:2112/debug/vars
func StartMetricsServer(metricsType Type) *http.Server {
	if metricsType == Prometheus {
		http.Handle("/metrics", promhttp.Handler())
	}
	server := &http.Server{Addr: ":2112"}
	go func() {
		server.ListenAndServe()
	}()
	return server
}
