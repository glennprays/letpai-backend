package valueobject

import (
	"errors"
	"strings"
)

// NotificationType represents the type of notification sent
type NotificationType string

const (
	// NotificationTypeInitial represents the initial payment request
	NotificationTypeInitial NotificationType = "initial"
	// NotificationTypeReminder represents a payment reminder
	NotificationTypeReminder NotificationType = "reminder"
	// NotificationTypeRejection represents a payment rejection notification
	NotificationTypeRejection NotificationType = "rejection"
)

// String returns the string representation of the notification type
func (n NotificationType) String() string {
	return string(n)
}

// IsValid checks if the notification type is valid
func (n NotificationType) IsValid() bool {
	switch n {
	case NotificationTypeInitial, NotificationTypeReminder, NotificationTypeRejection:
		return true
	default:
		return false
	}
}

// ParseNotificationType parses a string to a NotificationType
func ParseNotificationType(s string) (NotificationType, error) {
	notifType := NotificationType(strings.ToLower(s))
	if !notifType.IsValid() {
		return "", errors.New("invalid notification type")
	}
	return notifType, nil
}
