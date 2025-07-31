package shardqueue

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics are intentionally simple.  queueDepth is *only* updated in the worker
// goroutine, guaranteeing a single writer and eliminating race/skew concerns.
var (
	submissionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "synapse",
			Subsystem: "shardqueue",
			Name:      "submissions_total",
			Help:      "Jobs successfully accepted for execution.",
		},
		[]string{"shard"},
	)

	queueFullTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "synapse",
			Subsystem: "shardqueue",
			Name:      "queue_full_total",
			Help:      "Enqueue attempts that timed out (per‑shard queue full).",
		},
		[]string{"shard"},
	)

	runDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "synapse",
			Subsystem: "shardqueue",
			Name:      "run_duration_seconds",
			Help:      "Job execution latency.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"shard"},
	)

	queueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "synapse",
			Subsystem: "shardqueue",
			Name:      "queue_depth",
			Help:      "Current depth of each shard queue.",
		},
		[]string{"shard"},
	)
)

func labelFor(i int) string { return strconv.Itoa(i) }
