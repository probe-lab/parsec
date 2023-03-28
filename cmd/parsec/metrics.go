package main

import "github.com/prometheus/client_golang/prometheus"

var activeNodes = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "parsec_active_nodes",
		Help: "Number of nodes this scheduler controls.",
	},
)

var issuedProvides = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "parsec_issued_provides",
		Help: "Number of started provide operations.",
	},
	[]string{"success"},
)

var issuedRetrievals = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "parsec_issued_retrievals",
		Help: "Number of started retrieval operations.",
	},
	[]string{"success"},
)

func init() {
	prometheus.MustRegister(activeNodes)
	prometheus.MustRegister(issuedProvides)
	prometheus.MustRegister(issuedRetrievals)
}
