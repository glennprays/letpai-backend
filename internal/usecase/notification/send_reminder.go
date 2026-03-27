package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// SendReminderResponse represents the response after sending a reminder
type SendReminderResponse struct {
	Message           string `json:"message"`
	WhatsAppMessageID string `json:"whatsapp_message_id"`
	NextAvailableAt   string `json:"next_available_at"`
	RetryAfter        int64  `json:"retry_after,omitempty"`
}

// SendReminderUseCase handles sending reminder to a specific participant
type SendReminderUseCase struct {
	participantRepo ports.ParticipantRepository
	sessionRepo     ports.SessionRepository
	contactRepo     ports.ContactRepository
	whatsappSvc     *service.WhatsAppService
	rateLimitSvc    *service.RateLimitService
}

// NewSendReminderUseCase creates a new send reminder use case
func NewSendReminderUseCase(
	participantRepo ports.ParticipantRepository,
	sessionRepo ports.SessionRepository,
	contactRepo ports.ContactRepository,
	whatsappSvc *service.WhatsAppService,
	rateLimitSvc *service.RateLimitService,
) *SendReminderUseCase {
	return &SendReminderUseCase{
		participantRepo: participantRepo,
		sessionRepo:     sessionRepo,
		contactRepo:     contactRepo,
		whatsappSvc:     whatsappSvc,
		rateLimitSvc:    rateLimitSvc,
	}
}

// Execute sends a reminder to a specific participant
func (uc *SendReminderUseCase) Execute(ctx context.Context, userID, participantID string) (*SendReminderResponse, error) {
	// Get participant
	participant, err := uc.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return nil, err
	}

	// Get session to verify ownership
	session, err := uc.sessionRepo.FindByID(ctx, participant.SessionID.String(), userID)
	if err != nil {
		return nil, err
	}

	// Verify session belongs to user
	if session.UserID.String() != userID {
		return nil, domain.NewError(domain.ErrForbidden, errors.New("you can only send reminders for your own sessions"))
	}

	// Check if participant has already paid
	if participant.IsPaid() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("participant has already paid"))
	}

	// Check rate limit
	canSend, retryAfter, nextResetAt, err := uc.rateLimitSvc.GetReminderStatus(ctx, participantID)
	if err != nil {
		return nil, err
	}

	if !canSend {
		return &SendReminderResponse{
			Message:         fmt.Sprintf("Rate limit exceeded. Wait %s", service.FormatRetryAfter(retryAfter)),
			NextAvailableAt: nextResetAt.Format("2006-01-02T15:04:05Z07:00"),
			RetryAfter:      int64(retryAfter.Seconds()),
		}, domain.NewError(domain.ErrBadRequest, errors.New("rate limit exceeded"))
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
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("no WhatsApp number found for participant"))
	}

	// Format reminder message
	message := uc.formatReminderMessage(session.Title, participantName, participant.ShareAmount)

	// Send reminder and get message ID
	messageID, err := uc.whatsappSvc.SendNotification(ctx, whatsappNumber, message)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	// Record reminder for rate limiting
	_ = uc.rateLimitSvc.RecordReminder(ctx, participantID)

	return &SendReminderResponse{
		Message:           fmt.Sprintf("Reminder sent to %s", participantName),
		WhatsAppMessageID: messageID,
		NextAvailableAt:   nextResetAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// formatReminderMessage formats a reminder message
func (uc *SendReminderUseCase) formatReminderMessage(sessionName, participantName string, shareAmount float64) string {
	return fmt.Sprintf("*Letpai - Payment Reminder*\n\n"+
		"Hi %s!\n\n"+
		"This is a friendly reminder about your pending payment for: *%s*\n\n"+
		"Amount Due: *Rp%.0f*\n\n"+
		"Please submit your payment proof at your earliest convenience.\n\n"+
		"Thank you for using Letpai!",
		participantName, sessionName, shareAmount)
}
