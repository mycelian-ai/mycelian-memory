package health

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

type fakeChecker struct {
	name    string
	healthy atomic.Int32
}

func (f *fakeChecker) Name() string                               { return f.name }
func (f *fakeChecker) IsHealthy() bool                            { return f.healthy.Load() == 1 }
func (f *fakeChecker) Start(ctx context.Context, _ time.Duration) { /* no-op */ }

func TestServiceHealthChecker_Transitions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := zerolog.Nop()

	a := &fakeChecker{name: "a"}
	b := &fakeChecker{name: "b"}
	a.healthy.Store(1)
	b.healthy.Store(1)

	svc := NewServiceHealthChecker(logger, a, b)
	go svc.Start(ctx, 10*time.Millisecond)

	// Initially healthy
	waitTrue(t, func() bool { return svc.IsHealthy() })

	// Flip one to unhealthy
	b.healthy.Store(0)
	waitTrue(t, func() bool { return !svc.IsHealthy() })

	// Recover
	b.healthy.Store(1)
	waitTrue(t, func() bool { return svc.IsHealthy() })
}

func waitTrue(t *testing.T, pred func() bool) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if pred() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met before timeout")
}
