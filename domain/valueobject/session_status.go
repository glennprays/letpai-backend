package valueobject

import (
	"errors"
	"strings"
)

// SessionStatus represents the status of a session
type SessionStatus string

const (
	// SessionStatusActive represents an active session
	SessionStatusActive SessionStatus = "active"
	// SessionStatusCompleted represents a completed session
	SessionStatusCompleted SessionStatus = "completed"
	// SessionStatusCancelled represents a cancelled session
	SessionStatusCancelled SessionStatus = "cancelled"
)

// String returns the string representation of the session status
func (s SessionStatus) String() string {
	return string(s)
}

// IsValid checks if the session status is valid
func (s SessionStatus) IsValid() bool {
	switch s {
	case SessionStatusActive, SessionStatusCompleted, SessionStatusCancelled:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if the status can transition to another status
func (s SessionStatus) CanTransitionTo(newStatus SessionStatus) bool {
	switch s {
	case SessionStatusActive:
		return newStatus == SessionStatusCompleted || newStatus == SessionStatusCancelled
	case SessionStatusCompleted, SessionStatusCancelled:
		return false // Terminal states
	default:
		return false
	}
}

// ParseSessionStatus parses a string to a SessionStatus
func ParseSessionStatus(s string) (SessionStatus, error) {
	status := SessionStatus(strings.ToLower(s))
	if !status.IsValid() {
		return "", errors.New("invalid session status")
	}
	return status, nil
}

// IsTerminal returns true if the status is a terminal state
func (s SessionStatus) IsTerminal() bool {
	return s == SessionStatusCompleted || s == SessionStatusCancelled
}
