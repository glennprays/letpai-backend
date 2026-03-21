package service

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RateLimitService handles rate limiting for operations like reminders
// For MVP v1, this is a stub implementation using in-memory storage
// In production, this should use Redis
type RateLimitService struct {
	// In production, this would be a Redis client
	storage map[string]*rateLimitEntry
}

// rateLimitEntry represents a rate limit entry in storage
type rateLimitEntry struct {
	Count       int
	LastResetAt time.Time
	ExpiresAt   time.Time
}

// NewRateLimitService creates a new rate limit service
func NewRateLimitService() *RateLimitService {
	return &RateLimitService{
		storage: make(map[string]*rateLimitEntry),
	}
}

// ReminderRateLimitConfig represents rate limit configuration for reminders
type ReminderRateLimitConfig struct {
	MaxReminders  int           // Maximum reminders allowed
	TimeWindow    time.Duration // Time window for the limit
	CooldownPeriod time.Duration // Cooldown period between reminders
}

// DefaultReminderRateLimit returns the default rate limit for reminders
// 1 reminder per 24 hours
func DefaultReminderRateLimit() *ReminderRateLimitConfig {
	return &ReminderRateLimitConfig{
		MaxReminders:   1,
		TimeWindow:     24 * time.Hour,
		CooldownPeriod: 24 * time.Hour,
	}
}

// CheckRateLimit checks if an action is allowed under rate limiting
// Returns (allowed, retryAfter, error)
func (s *RateLimitService) CheckRateLimit(ctx context.Context, key string, config *ReminderRateLimitConfig) (bool, time.Duration, error) {
	now := time.Now()

	// Clean up expired entries
	s.cleanupExpired(now)

	// Get or create entry
	entry, exists := s.storage[key]
	if !exists {
		// First request, allow it
		s.storage[key] = &rateLimitEntry{
			Count:       1,
			LastResetAt: now,
			ExpiresAt:   now.Add(config.TimeWindow),
		}
		return true, 0, nil
	}

	// Check if entry has expired
	if now.After(entry.ExpiresAt) {
		// Reset count
		entry.Count = 1
		entry.LastResetAt = now
		entry.ExpiresAt = now.Add(config.TimeWindow)
		return true, 0, nil
	}

	// Check if count exceeds limit
	if entry.Count >= config.MaxReminders {
		// Calculate retry after duration
		retryAfter := entry.ExpiresAt.Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter, nil
	}

	// Check cooldown period
	timeSinceLastReset := now.Sub(entry.LastResetAt)
	if timeSinceLastReset < config.CooldownPeriod {
		retryAfter := config.CooldownPeriod - timeSinceLastReset
		return false, retryAfter, nil
	}

	// Increment count
	entry.Count++
	entry.LastResetAt = now
	return true, 0, nil
}

// RecordAction records an action for rate limiting
func (s *RateLimitService) RecordAction(ctx context.Context, key string, config *ReminderRateLimitConfig) error {
	now := time.Now()

	// Get or create entry
	entry, exists := s.storage[key]
	if !exists {
		s.storage[key] = &rateLimitEntry{
			Count:       1,
			LastResetAt: now,
			ExpiresAt:   now.Add(config.TimeWindow),
		}
		return nil
	}

	// Check if entry has expired
	if now.After(entry.ExpiresAt) {
		entry.Count = 1
		entry.LastResetAt = now
		entry.ExpiresAt = now.Add(config.TimeWindow)
		return nil
	}

	// Increment count
	entry.Count++
	return nil
}

// ResetRateLimit resets the rate limit for a key
func (s *RateLimitService) ResetRateLimit(ctx context.Context, key string) error {
	delete(s.storage, key)
	return nil
}

// GetRemainingCount returns the remaining count for a key
func (s *RateLimitService) GetRemainingCount(ctx context.Context, key string, config *ReminderRateLimitConfig) (int, time.Time, error) {
	now := time.Now()

	entry, exists := s.storage[key]
	if !exists {
		return config.MaxReminders, now.Add(config.TimeWindow), nil
	}

	// Check if entry has expired
	if now.After(entry.ExpiresAt) {
		return config.MaxReminders, now.Add(config.TimeWindow), nil
	}

	remaining := config.MaxReminders - entry.Count
	if remaining < 0 {
		remaining = 0
	}

	return remaining, entry.ExpiresAt, nil
}

// CheckReminderRateLimit checks rate limit specifically for reminders
// Uses participant ID as the key
func (s *RateLimitService) CheckReminderRateLimit(ctx context.Context, participantID string) (bool, time.Duration, error) {
	key := fmt.Sprintf("reminder:%s", participantID)
	config := DefaultReminderRateLimit()
	return s.CheckRateLimit(ctx, key, config)
}

// RecordReminder records a reminder being sent
func (s *RateLimitService) RecordReminder(ctx context.Context, participantID string) error {
	key := fmt.Sprintf("reminder:%s", participantID)
	config := DefaultReminderRateLimit()
	return s.RecordAction(ctx, key, config)
}

// GetReminderStatus returns the reminder status for a participant
func (s *RateLimitService) GetReminderStatus(ctx context.Context, participantID string) (canSend bool, retryAfter time.Duration, nextResetAt time.Time, err error) {
	key := fmt.Sprintf("reminder:%s", participantID)
	config := DefaultReminderRateLimit()

	remaining, expiresAt, err := s.GetRemainingCount(ctx, key, config)
	if err != nil {
		return false, 0, time.Time{}, err
	}

	canSend = remaining > 0
	if !canSend {
		now := time.Now()
		if expiresAt.After(now) {
			retryAfter = expiresAt.Sub(now)
		}
	}

	return canSend, retryAfter, expiresAt, nil
}

// cleanupExpired removes expired entries from storage
func (s *RateLimitService) cleanupExpired(now time.Time) {
	for key, entry := range s.storage {
		if now.After(entry.ExpiresAt) {
			delete(s.storage, key)
		}
	}
}

// GetKey generates a rate limit key for a specific operation
func (s *RateLimitService) GetKey(operation, id string) string {
	return fmt.Sprintf("%s:%s", operation, id)
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
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
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
	CanSend       bool      `json:"can_send"`
	RetryAfter    int64     `json:"retry_after,omitempty"`    // Seconds until next available
	NextResetAt   string    `json:"next_reset_at,omitempty"`  // ISO 8601 timestamp
	Remaining     int       `json:"remaining,omitempty"`      // Remaining reminders in window
	MaxReminders  int       `json:"max_reminders"`            // Maximum reminders allowed
}

// GetReminderStatusResponse returns a formatted reminder status
func (s *RateLimitService) GetReminderStatusResponse(ctx context.Context, participantID string) (*ReminderStatus, error) {
	canSend, retryAfter, nextResetAt, err := s.GetReminderStatus(ctx, participantID)
	if err != nil {
		return nil, err
	}

	config := DefaultReminderRateLimit()
	remaining, _, _ := s.GetRemainingCount(ctx, s.GetKey("reminder", participantID), config)

	status := &ReminderStatus{
		CanSend:      canSend,
		NextResetAt:  nextResetAt.Format(time.RFC3339),
		Remaining:    remaining,
		MaxReminders: config.MaxReminders,
	}

	if retryAfter > 0 {
		status.RetryAfter = int64(retryAfter.Seconds())
	}

	return status, nil
}

// BulkReminderCheck checks multiple participants for reminder eligibility
type BulkReminderCheckResult struct {
	ParticipantID string `json:"participant_id"`
	CanSend       bool   `json:"can_send"`
	Reason        string `json:"reason,omitempty"`
}

// BulkReminderCheck checks rate limits for multiple participants
func (s *RateLimitService) BulkReminderCheck(ctx context.Context, participantIDs []string) []BulkReminderCheckResult {
	results := make([]BulkReminderCheckResult, len(participantIDs))

	for i, pid := range participantIDs {
		canSend, retryAfter, _, _ := s.GetReminderStatus(ctx, pid)
		results[i] = BulkReminderCheckResult{
			ParticipantID: pid,
			CanSend:       canSend,
		}

		if !canSend {
			results[i].Reason = fmt.Sprintf("Rate limit: wait %s", FormatRetryAfter(retryAfter))
		}
	}

	return results
}
