package geziyor

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

// Metrics type stores metrics
type Metrics struct {
	requestCount  metrics.Counter
	responseCount metrics.Counter
}

func newMetrics() *Metrics {
	m := Metrics{
		requestCount: prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "geziyor",
			Name:      "request_count",
			Help:      "Request count",
		}, []string{"method"}),
		responseCount: prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "geziyor",
			Name:      "response_count",
			Help:      "Response count",
		}, []string{"method"}),
	}

	return &m
}
