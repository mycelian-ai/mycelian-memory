package internal_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestHTTPServerPerRequestGoroutine validates that the standard net/http
// server services concurrent requests in parallel goroutines. It launches N
// simultaneous requests against a handler that sleeps, and asserts that the
// maximum observed in-flight count exceeds 1, proving overlap.
func TestHTTPServerPerRequestGoroutine(t *testing.T) {
	var inFlight int32
	var maxObserved int32

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := atomic.AddInt32(&inFlight, 1)
		// record max
		for {
			m := atomic.LoadInt32(&maxObserved)
			if cur > m {
				if atomic.CompareAndSwapInt32(&maxObserved, m, cur) {
					break
				}
				continue
			}
			break
		}
		time.Sleep(30 * time.Millisecond) // hold the goroutine for a bit
		atomic.AddInt32(&inFlight, -1)
		w.WriteHeader(http.StatusOK)
	})

	ts := httptest.NewServer(h)
	defer ts.Close()

	const N = 10
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			if _, err := http.Get(ts.URL); err != nil {
				t.Errorf("request failed: %v", err)
			}
		}()
	}
	wg.Wait()

	if maxObserved <= 1 {
		t.Fatalf("expected concurrent request handling; max in-flight=%d", maxObserved)
	}
}
