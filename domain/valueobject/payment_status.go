package valueobject

import (
	"errors"
	"strings"
)

// PaymentStatus represents the payment status of a participant
type PaymentStatus string

const (
	// PaymentStatusPending represents a pending payment
	PaymentStatusPending PaymentStatus = "pending"
	// PaymentStatusSubmitted represents a submitted payment
	PaymentStatusSubmitted PaymentStatus = "submitted"
	// PaymentStatusPaid represents a paid payment
	PaymentStatusPaid PaymentStatus = "paid"
	// PaymentStatusRejected represents a rejected payment
	PaymentStatusRejected PaymentStatus = "rejected"
)

// String returns the string representation of the payment status
func (p PaymentStatus) String() string {
	return string(p)
}

// IsValid checks if the payment status is valid
func (p PaymentStatus) IsValid() bool {
	switch p {
	case PaymentStatusPending, PaymentStatusSubmitted, PaymentStatusPaid, PaymentStatusRejected:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if the status can transition to another status
func (p PaymentStatus) CanTransitionTo(newStatus PaymentStatus) bool {
	switch p {
	case PaymentStatusPending:
		return newStatus == PaymentStatusSubmitted || newStatus == PaymentStatusRejected
	case PaymentStatusSubmitted:
		return newStatus == PaymentStatusPaid || newStatus == PaymentStatusRejected
	case PaymentStatusPaid, PaymentStatusRejected:
		return false // Terminal states
	default:
		return false
	}
}

// ParsePaymentStatus parses a string to a PaymentStatus
func ParsePaymentStatus(s string) (PaymentStatus, error) {
	status := PaymentStatus(strings.ToLower(s))
	if !status.IsValid() {
		return "", errors.New("invalid payment status")
	}
	return status, nil
}

// IsTerminal returns true if the status is a terminal state
func (p PaymentStatus) IsTerminal() bool {
	return p == PaymentStatusPaid || p == PaymentStatusRejected
}
