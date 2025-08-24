package job

import (
	"fmt"
	"hash/fnv"
)

// ShardLabel hashes memoryID to a stable small cardinality label (0-31).
func ShardLabel(memID string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(memID))
	return fmt.Sprintf("%d", h.Sum32()%32)
}
