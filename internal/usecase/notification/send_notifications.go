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

// NotificationItem represents a queued notification (the gateway send
// runs on a goroutine so we report status:"queued" — the actual
// delivery result lands on the notification_logs row asynchronously).
type NotificationItem struct {
	ParticipantID string    `json:"participant_id"`
	Status        string    `json:"status"`
	QueuedAt      time.Time `json:"queued_at"`
}

// SendNotificationsResponse represents the response after queueing notifications
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
	notifier        *service.AsyncNotifier
	renderer        *service.TemplateRenderer
	appURL          string
}

// NewSendNotificationsUseCase creates a new send notifications use case
func NewSendNotificationsUseCase(
	sessionRepo ports.SessionRepository,
	participantRepo ports.ParticipantRepository,
	contactRepo ports.ContactRepository,
	billItemRepo ports.BillItemRepository,
	notifier *service.AsyncNotifier,
	renderer *service.TemplateRenderer,
	appURL string,
) *SendNotificationsUseCase {
	return &SendNotificationsUseCase{
		sessionRepo:     sessionRepo,
		participantRepo: participantRepo,
		contactRepo:     contactRepo,
		billItemRepo:    billItemRepo,
		notifier:        notifier,
		renderer:        renderer,
		appURL:          strings.TrimRight(appURL, "/"),
	}
}

// notificationVars is the data context passed to the
// `session_notification` template. Field names must match the
// {{.Field}} placeholders the admin authored in the body.
type notificationVars struct {
	ParticipantName string
	SessionName     string
	Total           string
	Share           string
	URL             string
}

// Execute queues WhatsApp notifications to all participants in a
// session and returns immediately. The gateway sends run on
// background goroutines so a slow upstream gateway can't time out
// the HTTP request.
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

	queued := make([]NotificationItem, 0, len(participants))

	for _, participant := range participants {
		whatsappNumber := participant.GetWhatsAppNumber()
		participantName := participant.GetName()

		vars := notificationVars{
			ParticipantName: participantName,
			SessionName:     session.Title,
			Total:           formatIDR(totalAmount),
			Share:           formatIDR(participant.ShareAmount),
			URL:             uc.appURL + "/payment/" + participant.ParticipantID.String(),
		}

		message, err := uc.renderer.Render(ctx, "session_notification", vars)
		if err != nil || message == "" {
			// Template fetch/parse failed: fall back to a hardcoded
			// body so a broken template doesn't block the send.
			message = fallbackSessionNotification(vars)
		}

		uc.notifier.Dispatch(participant.ParticipantID, valueobject.NotificationTypeInitial, whatsappNumber, message)

		queued = append(queued, NotificationItem{
			ParticipantID: participant.ParticipantID.String(),
			Status:        "queued",
			QueuedAt:      time.Now(),
		})
	}

	return &SendNotificationsResponse{
		Message:       fmt.Sprintf("%d notifications queued", len(queued)),
		SentCount:     len(queued),
		Notifications: queued,
	}, nil
}

// formatIDR renders an IDR amount as a thousand-separated whole-rupiah
// string (no decimals) suitable for plain-text WhatsApp messages.
func formatIDR(amount float64) string {
	whole := int64(amount)
	if whole < 0 {
		return "-" + formatIDR(float64(-whole))
	}
	if whole < 1000 {
		return fmt.Sprintf("%d", whole)
	}
	s := fmt.Sprintf("%d", whole)
	// Insert "." every three digits from the right (Indonesian format).
	n := len(s)
	out := make([]byte, 0, n+n/3)
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, byte(c))
	}
	return string(out)
}

func fallbackSessionNotification(v notificationVars) string {
	return fmt.Sprintf("*Letpai - Bill Split*\n\nHi %s!\n\nYou've been added to a bill split session: *%s*\n\nTotal Amount: *Rp%s*\nYour Share: *Rp%s*\n\nView your bill and upload proof here:\n%s\n\nThank you for using Letpai!",
		v.ParticipantName, v.SessionName, v.Total, v.Share, v.URL)
}
