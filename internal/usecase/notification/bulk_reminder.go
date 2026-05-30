package notification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
	"github.com/glennprays/letpai-backend/internal/service"
)

// BulkReminderSkipped represents a skipped reminder
type BulkReminderSkipped struct {
	ParticipantID string `json:"participant_id"`
	Reason        string `json:"reason"`
}

// BulkReminderResponse represents the response after bulk-queueing reminders
type BulkReminderResponse struct {
	Message       string                `json:"message"`
	SentCount     int                   `json:"sent_count"`
	Skipped       []BulkReminderSkipped `json:"skipped,omitempty"`
	Notifications []NotificationItem    `json:"notifications,omitempty"`
}

// BulkReminderUseCase handles bulk-queueing reminders to unpaid participants
type BulkReminderUseCase struct {
	participantRepo ports.ParticipantRepository
	sessionRepo     ports.SessionRepository
	contactRepo     ports.ContactRepository
	notifier        *service.AsyncNotifier
	renderer        *service.TemplateRenderer
	rateLimitSvc    *service.RateLimitService
	appURL          string
}

func NewBulkReminderUseCase(
	participantRepo ports.ParticipantRepository,
	sessionRepo ports.SessionRepository,
	contactRepo ports.ContactRepository,
	notifier *service.AsyncNotifier,
	renderer *service.TemplateRenderer,
	rateLimitSvc *service.RateLimitService,
	appURL string,
) *BulkReminderUseCase {
	return &BulkReminderUseCase{
		participantRepo: participantRepo,
		sessionRepo:     sessionRepo,
		contactRepo:     contactRepo,
		notifier:        notifier,
		renderer:        renderer,
		rateLimitSvc:    rateLimitSvc,
		appURL:          strings.TrimRight(appURL, "/"),
	}
}

func (uc *BulkReminderUseCase) Execute(ctx context.Context, userID, sessionID string) (*BulkReminderResponse, error) {
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

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
		if participant.IsPaid() {
			skipped = append(skipped, BulkReminderSkipped{
				ParticipantID: participant.ParticipantID.String(),
				Reason:        "Already paid",
			})
			continue
		}

		canSend, retryAfter, _, _ := uc.rateLimitSvc.GetReminderStatus(ctx, participant.ParticipantID.String())
		if !canSend {
			skipped = append(skipped, BulkReminderSkipped{
				ParticipantID: participant.ParticipantID.String(),
				Reason:        fmt.Sprintf("Rate limit: wait %s", service.FormatRetryAfter(retryAfter)),
			})
			continue
		}

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

		vars := reminderVars{
			ParticipantName: participantName,
			SessionName:     session.Title,
			Share:           formatIDR(participant.ShareAmount),
			URL:             uc.appURL + "/payment/" + participant.PublicSlug,
		}
		message, err := uc.renderer.Render(ctx, "payment_reminder", vars)
		if err != nil || message == "" {
			message = fallbackReminder(vars)
		}

		uc.notifier.Dispatch(participant.ParticipantID, valueobject.NotificationTypeReminder, whatsappNumber, message)
		_ = uc.rateLimitSvc.RecordReminder(ctx, participant.ParticipantID.String())

		notifications = append(notifications, NotificationItem{
			ParticipantID: participant.ParticipantID.String(),
			Status:        "queued",
			QueuedAt:      time.Now(),
		})

		sentCount++
	}

	return &BulkReminderResponse{
		Message:       fmt.Sprintf("%d reminders queued", sentCount),
		SentCount:     sentCount,
		Skipped:       skipped,
		Notifications: notifications,
	}, nil
}
