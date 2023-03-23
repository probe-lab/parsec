package server

import "github.com/prometheus/client_golang/prometheus"

var totalRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "parsec_http_requests_total",
		Help: "Number of http requests.",
	},
	[]string{"method", "path"},
)

var latencies = prometheus.NewSummaryVec(
	prometheus.SummaryOpts{
		Name:       "parsec_durations",
		Help:       "Redis requests latencies in seconds",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	},
	[]string{"type", "success"},
)

func init() {
	prometheus.MustRegister(totalRequests)
	prometheus.MustRegister(latencies)
}
