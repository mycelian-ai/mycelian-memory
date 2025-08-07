package client

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	entriesEnqueuedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mycelian_client",
			Name:      "entries_enqueued_total",
			Help:      "Entries accepted into the shard executor.",
		},
		[]string{"shard"},
	)

	entriesFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mycelian_client",
			Name:      "entries_enqueue_failures_total",
			Help:      "Entries whose async job returned error or panic.",
		},
		[]string{"shard"},
	)
)
