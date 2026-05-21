package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
	"github.com/glennprays/letpai-backend/internal/service"
	"github.com/google/uuid"
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
	sessionRepo         ports.SessionRepository
	participantRepo     ports.ParticipantRepository
	contactRepo         ports.ContactRepository
	billItemRepo        ports.BillItemRepository
	whatsappSvc         *service.WhatsAppService
	notificationLogRepo ports.NotificationLogRepository
}

// NewSendNotificationsUseCase creates a new send notifications use case
func NewSendNotificationsUseCase(
	sessionRepo ports.SessionRepository,
	participantRepo ports.ParticipantRepository,
	contactRepo ports.ContactRepository,
	billItemRepo ports.BillItemRepository,
	whatsappSvc *service.WhatsAppService,
	notificationLogRepo ports.NotificationLogRepository,
) *SendNotificationsUseCase {
	return &SendNotificationsUseCase{
		sessionRepo:         sessionRepo,
		participantRepo:     participantRepo,
		contactRepo:         contactRepo,
		billItemRepo:        billItemRepo,
		whatsappSvc:         whatsappSvc,
		notificationLogRepo: notificationLogRepo,
	}
}

// Execute sends WhatsApp notifications to all participants in a session
func (uc *SendNotificationsUseCase) Execute(ctx context.Context, userID, sessionID string) (*SendNotificationsResponse, error) {
	// Verify session exists and belongs to user
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Fetch participants with contact info joined in — one query, no N+1
	// per-participant contact lookup.
	participants, err := uc.participantRepo.FindBySessionIDWithContactInfo(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if len(participants) == 0 {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("no participants found in this session"))
	}

	// Use the session's stored TotalAmount; bill-list sum is a fallback
	// for sessions that haven't been recalculated yet.
	totalAmount := session.TotalAmount
	if totalAmount == 0 {
		billItems, err := uc.billItemRepo.FindBySessionID(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		for _, bill := range billItems {
			totalAmount += bill.Amount
		}
	}

	notifications := make([]NotificationItem, 0, len(participants))

	for _, participant := range participants {
		whatsappNumber := participant.GetWhatsAppNumber()
		participantName := participant.GetName()

		if whatsappNumber == "" {
			continue
		}

		// Format notification message
		message := uc.formatSessionNotification(session.Title, participantName, participant.ShareAmount, totalAmount)

		// Send notification and get message ID
		messageID, err := uc.whatsappSvc.SendNotification(ctx, whatsappNumber, message)
		if err != nil {
			// Create failed log entry
			errMsg := err.Error()
			log := &entity.NotificationLog{
				LogID:            uuid.New(),
				ParticipantID:    participant.ParticipantID,
				NotificationType: valueobject.NotificationTypeInitial,
				MessageContent:   message,
				SentAt:           time.Now(),
				Status:           entity.NotificationStatusFailed,
				ErrorMessage:     &errMsg,
			}
			uc.notificationLogRepo.Create(ctx, log)
			continue
		}

		// Create successful log entry
		log := &entity.NotificationLog{
			LogID:             uuid.New(),
			ParticipantID:     participant.ParticipantID,
			NotificationType:  valueobject.NotificationTypeInitial,
			WhatsAppMessageID: &messageID,
			MessageContent:    message,
			SentAt:            time.Now(),
			Status:            entity.NotificationStatusQueued,
		}
		if err := uc.notificationLogRepo.Create(ctx, log); err != nil {
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
		"You've been added to a bill split session: *%s*\n\n"+
		"Total Amount: *Rp%.0f*\n"+
		"Your Share: *Rp%.0f*\n\n"+
		"Please submit your payment proof at your earliest convenience.\n\n"+
		"Thank you for using Letpai!",
		participantName, sessionName, totalAmount, shareAmount)
}
