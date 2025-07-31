package client

import (
	"fmt"
	"hash/fnv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	entriesEnqueuedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "synapse_client",
			Name:      "entries_enqueued_total",
			Help:      "Entries accepted into the shard executor.",
		},
		[]string{"shard"},
	)

	entriesFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "synapse_client",
			Name:      "entries_enqueue_failures_total",
			Help:      "Entries whose async job returned error or panic.",
		},
		[]string{"shard"},
	)
)

// shardLabel hashes memoryID to a stable small cardinality label (0-31).
func shardLabel(memID string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(memID))
	return fmt.Sprintf("%d", h.Sum32()%32)
}
