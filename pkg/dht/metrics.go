package dht

import (
	"github.com/prometheus/client_golang/prometheus"
)

var diskUsage = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "parsec_datastore_disk_usage",
		Help: "Disk usage of leveldb",
	},
)

func init() {
	prometheus.MustRegister(diskUsage)
}
