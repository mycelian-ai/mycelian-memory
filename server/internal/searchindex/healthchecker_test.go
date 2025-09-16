package searchindex

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/model"
	"github.com/rs/zerolog"
)

// fakeIndex implements Index (and HealthPinger) for tests.
type fakeIndex struct{ pingErr error }

func (f fakeIndex) Search(context.Context, string, string, string, []float32, int, float32, bool) ([]model.SearchHit, error) {
	return nil, nil
}
func (f fakeIndex) LatestContext(context.Context, string, string) (string, time.Time, error) {
	return "", time.Time{}, nil
}
func (f fakeIndex) SearchContexts(ctx context.Context, actorID, memoryID, query string, vec []float32, topKC int, alpha float32) ([]model.ContextHit, error) {
	return []model.ContextHit{}, nil
}
func (f fakeIndex) UpsertEntry(context.Context, string, []float32, map[string]interface{}) error {
	return nil
}
func (f fakeIndex) UpsertContext(context.Context, string, []float32, map[string]interface{}) error {
	return nil
}
func (f fakeIndex) DeleteEntry(context.Context, string, string) error             { return nil }
func (f fakeIndex) DeleteContext(context.Context, string, string) error           { return nil }
func (f fakeIndex) DeleteMemory(context.Context, string, string) error            { return nil }
func (f fakeIndex) DeleteVault(ctx context.Context, userID, vaultID string) error { return nil }
func (f fakeIndex) HealthPing(ctx context.Context) error                          { return f.pingErr }

// fallbackIdx implements Index WITHOUT HealthPinger.
type fallbackIdx struct{ delErr error }

func (f fallbackIdx) Search(context.Context, string, string, string, []float32, int, float32, bool) ([]model.SearchHit, error) {
	return nil, nil
}
func (f fallbackIdx) LatestContext(context.Context, string, string) (string, time.Time, error) {
	return "", time.Time{}, nil
}
func (f fallbackIdx) SearchContexts(ctx context.Context, actorID, memoryID, query string, vec []float32, topKC int, alpha float32) ([]model.ContextHit, error) {
	return nil, f.delErr
}
func (f fallbackIdx) UpsertEntry(context.Context, string, []float32, map[string]interface{}) error {
	return nil
}
func (f fallbackIdx) UpsertContext(context.Context, string, []float32, map[string]interface{}) error {
	return nil
}
func (f fallbackIdx) DeleteEntry(context.Context, string, string) error             { return nil }
func (f fallbackIdx) DeleteContext(context.Context, string, string) error           { return nil }
func (f fallbackIdx) DeleteMemory(context.Context, string, string) error            { return nil }
func (f fallbackIdx) DeleteVault(ctx context.Context, userID, vaultID string) error { return f.delErr }

func TestSearchIndexHealthChecker_WithHealthPinger(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := zerolog.Nop()

	// Healthy
	hc := NewSearchIndexHealthChecker(fakeIndex{pingErr: nil}, logger, 50*time.Millisecond)
	go hc.Start(ctx, 20*time.Millisecond)
	waitTrue(t, func() bool { return hc.IsHealthy() })

	// Unhealthy
	hc2 := NewSearchIndexHealthChecker(fakeIndex{pingErr: errors.New("down")}, logger, 50*time.Millisecond)
	go hc2.Start(ctx, 20*time.Millisecond)
	waitTrue(t, func() bool { return !hc2.IsHealthy() })
}

func TestSearchIndexHealthChecker_FallbackDeleteVault(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := zerolog.Nop()

	// Healthy via fallback (no HealthPinger available)
	var onlyIndex Index = fallbackIdx{delErr: nil}
	hc := NewSearchIndexHealthChecker(onlyIndex, logger, 50*time.Millisecond)
	go hc.Start(ctx, 20*time.Millisecond)
	waitTrue(t, func() bool { return hc.IsHealthy() })

	// Unhealthy via fallback
	var onlyIndexBad Index = fallbackIdx{delErr: errors.New("fail")}
	hc2 := NewSearchIndexHealthChecker(onlyIndexBad, logger, 50*time.Millisecond)
	go hc2.Start(ctx, 20*time.Millisecond)
	waitTrue(t, func() bool { return !hc2.IsHealthy() })
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
