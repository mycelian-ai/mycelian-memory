package storage

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// pingable fake Store that also implements HealthPing
type fakePingStore struct {
	mu  sync.RWMutex
	err error
}

// Implement Store interface with nil sub-stores (not used in tests)
func (f *fakePingStore) Vaults() Vaults     { return nil }
func (f *fakePingStore) Memories() Memories { return nil }
func (f *fakePingStore) Entries() Entries   { return nil }
func (f *fakePingStore) Contexts() Contexts { return nil }

// HealthPing satisfies health.HealthPinger via method presence.
func (f *fakePingStore) HealthPing(ctx context.Context) error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.err
}

func (f *fakePingStore) setHealthy() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = nil
}
func (f *fakePingStore) setUnhealthy() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = errors.New("db down")
}

func waitFor(t *testing.T, cond func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}

func TestStoreHealthChecker_Name(t *testing.T) {
	hc := NewStoreHealthChecker(&fakePingStore{}, zerolog.New(io.Discard), 20*time.Millisecond)
	if hc.Name() != "store" {
		t.Fatalf("Name: got %q, want %q", hc.Name(), "store")
	}
}

func TestStoreHealthChecker_TogglesWithHealthPing(t *testing.T) {
	store := &fakePingStore{}
	log := zerolog.New(io.Discard)
	hc := NewStoreHealthChecker(store, log, 20*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start unhealthy
	store.setUnhealthy()
	go hc.Start(ctx, 30*time.Millisecond)

	// Should become unhealthy after first probe
	waitFor(t, func() bool { return hc.IsHealthy() == false }, 500*time.Millisecond)

	// Become healthy and verify flip
	store.setHealthy()
	waitFor(t, func() bool { return hc.IsHealthy() == true }, 500*time.Millisecond)

	// Become unhealthy again and verify
	store.setUnhealthy()
	waitFor(t, func() bool { return hc.IsHealthy() == false }, 500*time.Millisecond)
}
