package indexer

import (
	"os"
	"testing"
	"time"
)

func TestStateLoadSave(t *testing.T) {
	tmp := t.TempDir()
	s, err := NewState(tmp)
	if err != nil {
		t.Fatalf("init state: %v", err)
	}

	user, mem := "u1", "m1"
	ts := time.Now().UTC().Truncate(time.Microsecond)

	// initial load should be zero
	got, err := s.Load(user, mem)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !got.IsZero() {
		t.Fatalf("expected zero time, got %v", got)
	}

	if err := s.Save(user, mem, ts); err != nil {
		t.Fatalf("save: %v", err)
	}

	got2, err := s.Load(user, mem)
	if err != nil {
		t.Fatalf("load2: %v", err)
	}
	if !got2.Equal(ts) {
		t.Fatalf("expected %v, got %v", ts, got2)
	}

	// ensure file exists
	files, _ := os.ReadDir(tmp)
	if len(files) == 0 {
		t.Fatalf("expected state file to be created")
	}
}
