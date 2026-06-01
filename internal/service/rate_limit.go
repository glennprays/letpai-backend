package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

// RateLimitService handles rate limiting for operations like login, OTP, and reminders
// Uses Redis with go-redis/redis_rate for distributed rate limiting
type RateLimitService struct {
	limiter *redis_rate.Limiter
}

// NewRateLimitService creates a new rate limit service with Redis client
func NewRateLimitService(redisClient *redis.Client) *RateLimitService {
	return &RateLimitService{
		limiter: redis_rate.NewLimiter(redisClient),
	}
}

// RateLimitResult represents the result of a rate limit check
type RateLimitResult struct {
	Allowed    bool          `json:"allowed"`
	Remaining  int           `json:"remaining"`
	ResetAfter time.Duration `json:"reset_after"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
	Limit      int           `json:"limit"`
	Window     time.Duration `json:"window"`
}

// CheckLoginRateLimit checks rate limit for login attempts
// Returns: 5 attempts per 15 minutes per IP/identifier
func (s *RateLimitService) CheckLoginRateLimit(ctx context.Context, identifier string) (*RateLimitResult, error) {
	return s.checkRateLimit(ctx, "login", identifier, 5, 15*time.Minute)
}

// CheckOTPRateLimit checks rate limit for OTP verification attempts
// Returns: 3 attempts per OTP code (single-use)
func (s *RateLimitService) CheckOTPRateLimit(ctx context.Context, otpCode string) (*RateLimitResult, error) {
	return s.checkRateLimit(ctx, "otp", otpCode, 3, 5*time.Minute)
}

// CheckReminderRateLimit checks rate limit for reminder sending
// Returns: 1 reminder per 24 hours per participant
func (s *RateLimitService) CheckReminderRateLimit(ctx context.Context, participantID string) (*RateLimitResult, error) {
	return s.checkRateLimit(ctx, "reminder", participantID, 1, 24*time.Hour)
}

// checkRateLimit performs the actual rate limit check using Redis.
//
// redis_rate's `Limit` treats `Rate` as "events allowed per Period",
// not "events per second". The previous code divided limit by
// window.Seconds() and truncated to int — for a 1-per-24h reminder
// that collapses to Rate=0, which the Lua-script underlying
// redis_rate refuses, returning an error on every call. Because the
// surrounding middleware fail-opens on errors, the limiter has been
// effectively a no-op in staging while still emitting a 500 from
// any direct (non-middleware) caller — which is what surfaced when
// the new /reminder-status read started hitting the same path.
//
// The correct call is to pass the limit value through as the rate
// over the given Period — `1 per 24h` becomes
// `Limit{Rate: 1, Burst: 1, Period: 24h}`.
func (s *RateLimitService) checkRateLimit(ctx context.Context, operation, identifier string, limit int, window time.Duration) (*RateLimitResult, error) {
	key := fmt.Sprintf("ratelimit:%s:%s", operation, identifier)

	rateLimit := redis_rate.Limit{
		Rate:   limit,
		Burst:  limit,
		Period: window,
	}

	// Check rate limit
	res, err := s.limiter.Allow(ctx, key, rateLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}

	result := &RateLimitResult{
		Allowed:    res.Allowed > 0,
		Remaining:  res.Remaining,
		ResetAfter: res.ResetAfter,
		Limit:      limit,
		Window:     window,
	}

	// Calculate retry after if not allowed
	if !result.Allowed {
		result.RetryAfter = res.RetryAfter
	}

	return result, nil
}

// ResetRateLimit resets the rate limit for a specific key
func (s *RateLimitService) ResetRateLimit(ctx context.Context, operation, identifier string) error {
	key := fmt.Sprintf("ratelimit:%s:%s", operation, identifier)
	return s.limiter.Reset(ctx, key)
}

// GetRateLimitStatus returns the current rate limit status for a key
func (s *RateLimitService) GetRateLimitStatus(ctx context.Context, operation, identifier string, limit int, window time.Duration) (*RateLimitResult, error) {
	return s.checkRateLimit(ctx, operation, identifier, limit, window)
}

// GetKey generates a rate limit key for a specific operation
func (s *RateLimitService) GetKey(operation, id string) string {
	return fmt.Sprintf("ratelimit:%s:%s", operation, id)
}

// FormatRetryAfter formats retry after duration into a human-readable string
func FormatRetryAfter(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// CalculateRetryAfterSeconds calculates retry after in seconds for HTTP headers
func CalculateRetryAfterSeconds(retryAfter time.Duration) int {
	seconds := int(math.Ceil(retryAfter.Seconds()))
	if seconds < 0 {
		seconds = 0
	}
	return seconds
}

// ReminderStatus represents the status of reminder rate limiting
type ReminderStatus struct {
	CanSend      bool   `json:"can_send"`
	RetryAfter   int64  `json:"retry_after,omitempty"`   // Seconds until next available
	NextResetAt  string `json:"next_reset_at,omitempty"` // ISO 8601 timestamp
	Remaining    int    `json:"remaining,omitempty"`     // Remaining reminders in window
	MaxReminders int    `json:"max_reminders"`           // Maximum reminders allowed
}

// GetReminderStatusResponse returns a formatted reminder status
func (s *RateLimitService) GetReminderStatusResponse(ctx context.Context, participantID string) (*ReminderStatus, error) {
	result, err := s.CheckReminderRateLimit(ctx, participantID)
	if err != nil {
		return nil, err
	}

	status := &ReminderStatus{
		CanSend:      result.Allowed,
		Remaining:    result.Remaining,
		MaxReminders: result.Limit,
	}

	if !result.Allowed {
		status.RetryAfter = int64(result.RetryAfter.Seconds())
		status.NextResetAt = time.Now().Add(result.ResetAfter).Format(time.RFC3339)
	} else {
		status.NextResetAt = time.Now().Add(result.ResetAfter).Format(time.RFC3339)
	}

	return status, nil
}

// BulkReminderCheckResult represents result of bulk reminder check
type BulkReminderCheckResult struct {
	ParticipantID string `json:"participant_id"`
	CanSend       bool   `json:"can_send"`
	Reason        string `json:"reason,omitempty"`
}

// BulkReminderCheck checks rate limits for multiple participants
func (s *RateLimitService) BulkReminderCheck(ctx context.Context, participantIDs []string) []BulkReminderCheckResult {
	results := make([]BulkReminderCheckResult, len(participantIDs))

	for i, pid := range participantIDs {
		result, err := s.CheckReminderRateLimit(ctx, pid)
		if err != nil {
			results[i] = BulkReminderCheckResult{
				ParticipantID: pid,
				CanSend:       false,
				Reason:        "Error checking rate limit",
			}
			continue
		}

		results[i] = BulkReminderCheckResult{
			ParticipantID: pid,
			CanSend:       result.Allowed,
		}

		if !result.Allowed {
			results[i].Reason = fmt.Sprintf("Rate limit: wait %s", FormatRetryAfter(result.RetryAfter))
		}
	}

	return results
}

// RateLimitHeaders represents HTTP headers for rate limiting
type RateLimitHeaders struct {
	Limit      int
	Remaining  int
	Reset      int64 // Unix timestamp
	RetryAfter int   // Seconds
}

// GetRateLimitHeaders returns HTTP headers for rate limiting response
func (s *RateLimitService) GetRateLimitHeaders(result *RateLimitResult) RateLimitHeaders {
	headers := RateLimitHeaders{
		Limit:     result.Limit,
		Remaining: result.Remaining,
		Reset:     time.Now().Add(result.ResetAfter).Unix(),
	}

	if !result.Allowed {
		headers.RetryAfter = CalculateRetryAfterSeconds(result.RetryAfter)
	}

	return headers
}

// GetReminderStatus checks if a reminder can be sent and returns status info.
//
// This is a READ — it must not consume a token, otherwise the FE's
// page-load status fetch would itself eat the participant's single
// allowed reminder for the next 24h. The underlying redis_rate Lua
// script treats `AllowN(..., n=0)` as a peek (cost=0 → no-op SET),
// returning the current `Remaining` and `RetryAfter` without
// decrementing. The send path (POST /reminder) still goes through
// the normal `Allow` (cost=1) via the middleware.
func (s *RateLimitService) GetReminderStatus(ctx context.Context, participantID string) (bool, time.Duration, time.Time, error) {
	key := fmt.Sprintf("ratelimit:%s:%s", "reminder", participantID)
	limit := 1
	window := 24 * time.Hour
	rateLimit := redis_rate.Limit{Rate: limit, Burst: limit, Period: window}

	res, err := s.limiter.AllowN(ctx, key, rateLimit, 0)
	if err != nil {
		return false, 0, time.Time{}, fmt.Errorf("failed to peek rate limit: %w", err)
	}

	canSend := res.Remaining > 0
	if canSend {
		return true, 0, time.Time{}, nil
	}
	nextResetAt := time.Now().Add(res.ResetAfter)
	return false, res.ResetAfter, nextResetAt, nil
}

// RecordReminder records that a reminder was sent for rate limiting tracking
// This consumes one count from the rate limiter
func (s *RateLimitService) RecordReminder(ctx context.Context, participantID string) error {
	// The rate limiter already counts the check in GetReminderStatus/CheckReminderRateLimit
	// So this is a no-op but kept for API compatibility
	// If we need to explicitly record, we can call Allow again
	return nil
}
