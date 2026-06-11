package service

import (
	"math/rand"
	"time"
)

// backoffSchedule is the base retry delay ladder for failed notification
// sends, indexed by attempt number (1 = first failure). Delays past the end
// of the ladder hold at the final value.
var backoffSchedule = []time.Duration{
	30 * time.Second,
	2 * time.Minute,
	10 * time.Minute,
	1 * time.Hour,
	6 * time.Hour,
}

// backoffBase returns the un-jittered delay for the given attempt number
// (1-based), capped at the final ladder entry.
func backoffBase(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > len(backoffSchedule) {
		attempt = len(backoffSchedule)
	}
	return backoffSchedule[attempt-1]
}

// NextBackoff returns the delay before the next retry for a row that has just
// made its attempt-th attempt, with +/-20% jitter to avoid thundering-herd
// retries when many sends fail at once (e.g. the gateway briefly going down).
func NextBackoff(attempt int) time.Duration {
	base := backoffBase(attempt)
	// jitter factor in [0.8, 1.2]
	factor := 0.8 + rand.Float64()*0.4
	return time.Duration(float64(base) * factor)
}
