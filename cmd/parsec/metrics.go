package main

import (
	"context"
	"time"

	leveldb "github.com/ipfs/go-ds-leveldb"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

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

var diskUsageGauge = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "parsec_datastore_disk_usage",
		Help: "Disk usage of leveldb",
	},
)

func measureDiskUsage(ctx context.Context, ds *leveldb.Datastore) {
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		usage, err := ds.DiskUsage(ctx)
		if err != nil {
			log.WithError(err).Warnln("Failed getting disk usage")
			continue
		}

		diskUsageGauge.Set(float64(usage))
	}
}
