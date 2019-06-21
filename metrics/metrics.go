package metrics

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"
	"github.com/go-kit/kit/metrics/expvar"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
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
	RequestCounter  metrics.Counter
	ResponseCounter metrics.Counter
}

// NewMetrics creates new metrics with given metrics.Type
func NewMetrics(metricsType Type) *Metrics {
	switch metricsType {
	case Discard:
		return &Metrics{
			RequestCounter:  discard.NewCounter(),
			ResponseCounter: discard.NewCounter(),
		}
	case ExpVar:
		return &Metrics{
			RequestCounter:  expvar.NewCounter("request_count"),
			ResponseCounter: expvar.NewCounter("response_count"),
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
			}, []string{"method"}),
		}
	default:
		return nil
	}
}
