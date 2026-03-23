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

// NotificationItem represents a sent notification
type NotificationItem struct {
	ParticipantID     string    `json:"participant_id"`
	WhatsAppMessageID string    `json:"whatsapp_message_id"`
	SentAt            time.Time `json:"sent_at"`
}

// SendNotificationsResponse represents the response after sending notifications
type SendNotificationsResponse struct {
	Message       string             `json:"message"`
	SentCount     int                `json:"sent_count"`
	Notifications []NotificationItem `json:"notifications"`
}

// SendNotificationsUseCase handles sending notifications to all session participants
type SendNotificationsUseCase struct {
	sessionRepo     ports.SessionRepository
	participantRepo ports.ParticipantRepository
	contactRepo     ports.ContactRepository
	billItemRepo    ports.BillItemRepository
	whatsappSvc     *service.WhatsAppService
}

// NewSendNotificationsUseCase creates a new send notifications use case
func NewSendNotificationsUseCase(
	sessionRepo ports.SessionRepository,
	participantRepo ports.ParticipantRepository,
	contactRepo ports.ContactRepository,
	billItemRepo ports.BillItemRepository,
	whatsappSvc *service.WhatsAppService,
) *SendNotificationsUseCase {
	return &SendNotificationsUseCase{
		sessionRepo:     sessionRepo,
		participantRepo: participantRepo,
		contactRepo:     contactRepo,
		billItemRepo:    billItemRepo,
		whatsappSvc:     whatsappSvc,
	}
}

// Execute sends WhatsApp notifications to all participants in a session
func (uc *SendNotificationsUseCase) Execute(ctx context.Context, userID, sessionID string) (*SendNotificationsResponse, error) {
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

	// Get bill items to calculate total
	billItems, _ := uc.billItemRepo.FindBySessionID(ctx, sessionID)
	totalAmount := session.TotalAmount
	if totalAmount == 0 && len(billItems) > 0 {
		for _, bill := range billItems {
			totalAmount += bill.Amount
		}
	}

	notifications := make([]NotificationItem, 0, len(participants))

	for _, participant := range participants {
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
			continue
		}

		// Format notification message
		message := uc.formatSessionNotification(session.Title, participantName, participant.ShareAmount, totalAmount)

		// Send notification and get message ID
		messageID, err := uc.whatsappSvc.SendNotification(ctx, whatsappNumber, message)
		if err != nil {
			// Log error but continue with other participants
			continue
		}

		notifications = append(notifications, NotificationItem{
			ParticipantID:     participant.ParticipantID.String(),
			WhatsAppMessageID: messageID,
			SentAt:            time.Now(),
		})
	}

	return &SendNotificationsResponse{
		Message:       fmt.Sprintf("%d notifications sent", len(notifications)),
		SentCount:     len(notifications),
		Notifications: notifications,
	}, nil
}

// formatSessionNotification formats a session notification message
func (uc *SendNotificationsUseCase) formatSessionNotification(sessionName, participantName string, shareAmount, totalAmount float64) string {
	return fmt.Sprintf("*Letpai - Bill Split*\n\n"+
		"Hi %s!\n\n"+
		"You've been added to the bill split session: *%s*\n\n"+
		"Total Amount: *Rp%.0f*\n"+
		"Your Share: *Rp%.0f*\n\n"+
		"Please submit your payment proof at your earliest convenience.\n\n"+
		"Thank you for using Letpai!",
		participantName, sessionName, totalAmount, shareAmount)
}
