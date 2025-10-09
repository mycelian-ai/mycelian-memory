package memoryservice

import "testing"

func TestCalculateStartupHealthTimeout(t *testing.T) {
	tests := []struct {
		interval int
		want     int
	}{
		{0, 60},   // minimum floor
		{10, 60},  // 2*10=20 < 60 -> 60
		{30, 60},  // 2*30=60
		{50, 100}, // 2*50=100
		{75, 150}, // 2*75=150
	}
	for _, tt := range tests {
		got := calculateStartupHealthTimeout(tt.interval)
		if got != tt.want {
			t.Fatalf("interval=%d: got %d, want %d", tt.interval, got, tt.want)
		}
	}
}
