package service

import (
	"testing"
	"time"
)

func TestNotificationBackoffSchedule(t *testing.T) {
	// The base schedule (ignoring jitter) should ramp 30s, 2m, 10m, 1h, 6h
	// and then hold at the 6h cap. attempt is the number of attempts already
	// made (1 = first failure).
	base := []time.Duration{
		30 * time.Second,
		2 * time.Minute,
		10 * time.Minute,
		1 * time.Hour,
		6 * time.Hour,
	}
	for i, want := range base {
		attempt := i + 1
		got := backoffBase(attempt)
		if got != want {
			t.Errorf("backoffBase(%d) = %v, want %v", attempt, got, want)
		}
	}
	if got := backoffBase(6); got != 6*time.Hour {
		t.Errorf("backoffBase(6) = %v, want cap 6h", got)
	}
	if got := backoffBase(99); got != 6*time.Hour {
		t.Errorf("backoffBase(99) = %v, want cap 6h", got)
	}
}

func TestNextBackoffStaysWithinJitterBand(t *testing.T) {
	// NextBackoff applies +/-20% jitter around the base; assert it lands in
	// band for a representative attempt across many draws.
	base := backoffBase(2) // 2m
	lo := time.Duration(float64(base) * 0.8)
	hi := time.Duration(float64(base) * 1.2)
	for i := 0; i < 200; i++ {
		d := NextBackoff(2)
		if d < lo || d > hi {
			t.Fatalf("NextBackoff(2) = %v, want within [%v,%v]", d, lo, hi)
		}
	}
}
