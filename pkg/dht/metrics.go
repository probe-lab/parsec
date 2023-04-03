package dht

import (
	"github.com/prometheus/client_golang/prometheus"
)

var diskUsageGauge = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "parsec_datastore_disk_usage",
		Help: "Disk usage of leveldb",
	},
)

var netSizeGauge = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "parsec_network_size",
		Help: "The current network size estimate",
	},
)

func init() {
	prometheus.MustRegister(diskUsageGauge)
	prometheus.MustRegister(netSizeGauge)
}
