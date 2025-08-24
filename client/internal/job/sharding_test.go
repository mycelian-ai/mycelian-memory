package job

import (
	"strconv"
	"testing"
)

func TestShardLabel_DeterministicAndRange(t *testing.T) {
	t.Parallel()
	ids := []string{"", "m1", "m2", "m3", "some-longer-id"}
	for _, id := range ids {
		got1 := ShardLabel(id)
		got2 := ShardLabel(id)
		if got1 != got2 {
			t.Fatalf("ShardLabel not deterministic for %q: %s vs %s", id, got1, got2)
		}
		// Ensure numeric in [0,31]
		n, err := strconv.Atoi(got1)
		if err != nil || n < 0 || n > 31 {
			t.Fatalf("ShardLabel out of range for %q: %s", id, got1)
		}
	}
}
