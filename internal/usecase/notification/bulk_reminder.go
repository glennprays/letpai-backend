package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// BulkReminderSkipped represents a skipped reminder
type BulkReminderSkipped struct {
	ParticipantID string `json:"participant_id"`
	Reason        string `json:"reason"`
}

// BulkReminderResponse represents the response after bulk sending reminders
type BulkReminderResponse struct {
	Message       string                `json:"message"`
	SentCount     int                   `json:"sent_count"`
	Skipped       []BulkReminderSkipped `json:"skipped,omitempty"`
	Notifications []NotificationItem    `json:"notifications,omitempty"`
}

// BulkReminderUseCase handles sending bulk reminders to unpaid participants
type BulkReminderUseCase struct {
	participantRepo ports.ParticipantRepository
	sessionRepo     ports.SessionRepository
	contactRepo     ports.ContactRepository
	whatsappSvc     *service.WhatsAppService
	rateLimitSvc    *service.RateLimitService
}

// NewBulkReminderUseCase creates a new bulk reminder use case
func NewBulkReminderUseCase(
	participantRepo ports.ParticipantRepository,
	sessionRepo ports.SessionRepository,
	contactRepo ports.ContactRepository,
	whatsappSvc *service.WhatsAppService,
	rateLimitSvc *service.RateLimitService,
) *BulkReminderUseCase {
	return &BulkReminderUseCase{
		participantRepo: participantRepo,
		sessionRepo:     sessionRepo,
		contactRepo:     contactRepo,
		whatsappSvc:     whatsappSvc,
		rateLimitSvc:    rateLimitSvc,
	}
}

// Execute sends reminders to all unpaid participants in a session
func (uc *BulkReminderUseCase) Execute(ctx context.Context, userID, sessionID string) (*BulkReminderResponse, error) {
	// Verify session exists and belongs to user
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Get all participants for the session
	participants, err := uc.participantRepo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if len(participants) == 0 {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("no participants found in this session"))
	}

	sentCount := 0
	skipped := make([]BulkReminderSkipped, 0)
	notifications := make([]NotificationItem, 0)

	for _, participant := range participants {
		// Skip paid participants
		if participant.IsPaid() {
			skipped = append(skipped, BulkReminderSkipped{
				ParticipantID: participant.ParticipantID.String(),
				Reason:        "Already paid",
			})
			continue
		}

		// Check rate limit
		canSend, retryAfter, _, _ := uc.rateLimitSvc.GetReminderStatus(ctx, participant.ParticipantID.String())
		if !canSend {
			skipped = append(skipped, BulkReminderSkipped{
				ParticipantID: participant.ParticipantID.String(),
				Reason:        fmt.Sprintf("Rate limit: wait %s", service.FormatRetryAfter(retryAfter)),
			})
			continue
		}

		// Get WhatsApp number
		whatsappNumber := ""
		participantName := ""
		if participant.ContactID != nil {
			contact, err := uc.contactRepo.FindByID(ctx, participant.ContactID.String(), userID)
			if err == nil {
				whatsappNumber = contact.WhatsAppNumber
				participantName = contact.Name
			}
		} else {
			whatsappNumber = participant.CustomWhatsApp
			participantName = participant.CustomName
		}

		if whatsappNumber == "" {
			skipped = append(skipped, BulkReminderSkipped{
				ParticipantID: participant.ParticipantID.String(),
				Reason:        "No WhatsApp number found",
			})
			continue
		}

		// Format reminder message
		message := uc.formatReminderMessage(session.Title, participantName, participant.ShareAmount)

		// Send reminder and get message ID
		messageID, err := uc.whatsappSvc.SendNotification(ctx, whatsappNumber, message)
		if err != nil {
			skipped = append(skipped, BulkReminderSkipped{
				ParticipantID: participant.ParticipantID.String(),
				Reason:        "Failed to send",
			})
			continue
		}

		// Record reminder for rate limiting
		_ = uc.rateLimitSvc.RecordReminder(ctx, participant.ParticipantID.String())

		notifications = append(notifications, NotificationItem{
			ParticipantID:     participant.ParticipantID.String(),
			WhatsAppMessageID: messageID,
			SentAt:            time.Now(),
		})

		sentCount++
	}

	return &BulkReminderResponse{
		Message:       fmt.Sprintf("%d reminders sent", sentCount),
		SentCount:     sentCount,
		Skipped:       skipped,
		Notifications: notifications,
	}, nil
}

// formatReminderMessage formats a reminder message
func (uc *BulkReminderUseCase) formatReminderMessage(sessionName, participantName string, shareAmount float64) string {
	return fmt.Sprintf("*Letpai - Payment Reminder*\n\n"+
		"Hi %s!\n\n"+
		"This is a friendly reminder about your pending payment for: *%s*\n\n"+
		"Amount Due: *Rp%.0f*\n\n"+
		"Please submit your payment proof at your earliest convenience.\n\n"+
		"Thank you for using Letpai!",
		participantName, sessionName, shareAmount)
}
