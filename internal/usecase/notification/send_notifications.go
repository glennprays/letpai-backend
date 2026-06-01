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
	userRepo        ports.UserRepository
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
	userRepo ports.UserRepository,
	notifier *service.AsyncNotifier,
	renderer *service.TemplateRenderer,
	appURL string,
) *SendNotificationsUseCase {
	return &SendNotificationsUseCase{
		sessionRepo:     sessionRepo,
		participantRepo: participantRepo,
		contactRepo:     contactRepo,
		billItemRepo:    billItemRepo,
		userRepo:        userRepo,
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
	MakerName       string
	Total           string
	Share           string
	URL             string
}

// ErrSessionNotDirty is the canonical not-dirty sentinel. Handlers
// translate this into 409 SESSION_NOT_DIRTY so the FE can render its
// "you already sent these — resend anyway?" confirm modal.
var ErrSessionNotDirty = errors.New("session has no changes since last notification")

// NotDirtyError wraps ErrSessionNotDirty with the timestamp the FE
// needs to render its confirm modal ("Already sent 12 min ago"). The
// handler unwraps to populate the 409 body without an extra DB round
// trip.
type NotDirtyError struct {
	LastNotifiedAt time.Time
}

func (e *NotDirtyError) Error() string { return ErrSessionNotDirty.Error() }
func (e *NotDirtyError) Unwrap() error { return ErrSessionNotDirty }

// Execute queues WhatsApp notifications to all participants in a
// session and returns immediately. The gateway sends run on
// background goroutines so a slow upstream gateway can't time out
// the HTTP request.
//
// `force=true` bypasses the dirty-for-notify check (host explicitly
// chose to resend via the /resend escape hatch). `force=false` is
// the default UX path and returns ErrSessionNotDirty when the
// session has been notified and nothing has changed since.
func (uc *SendNotificationsUseCase) Execute(ctx context.Context, userID, sessionID string, force bool) (*SendNotificationsResponse, error) {
	// Verify session exists and belongs to user
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Count unpaid participants up front. Two reasons:
	//   1. Feeds IsDirtyForNotify so "mark everyone paid" doesn't
	//      keep the gate open via the trigger-driven updated_at bump.
	//   2. Lets us short-circuit when there's literally no one left
	//      to notify (saves a goroutine spawn and a gateway round
	//      trip per zero-fanout call).
	pendingCount, err := uc.participantRepo.CountBySessionIDAndStatus(ctx, sessionID, "pending")
	if err != nil {
		return nil, err
	}
	submittedCount, err := uc.participantRepo.CountBySessionIDAndStatus(ctx, sessionID, "submitted")
	if err != nil {
		return nil, err
	}
	rejectedCount, err := uc.participantRepo.CountBySessionIDAndStatus(ctx, sessionID, "rejected")
	if err != nil {
		return nil, err
	}
	hasUnpaid := (pendingCount + submittedCount + rejectedCount) > 0

	if !force && !session.IsDirtyForNotify(hasUnpaid) {
		nde := &NotDirtyError{}
		if session.LastNotifiedAt != nil {
			nde.LastNotifiedAt = *session.LastNotifiedAt
		}
		return nil, domain.NewError(domain.ErrConflict, nde)
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

	// Resolve the session host's display name for MakerName.
	makerName := "the host"
	if host, err := uc.userRepo.FindByID(ctx, session.UserID.String()); err == nil && host.FullName != "" {
		makerName = host.FullName
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
			MakerName:       makerName,
			Total:           formatIDR(totalAmount),
			Share:           formatIDR(participant.ShareAmount),
			URL:             uc.appURL + "/payment/" + participant.PublicSlug,
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

	// Stamp last_notified_at exactly once, after all dispatches are
	// queued. We mark even if some individual goroutines later fail —
	// the host's "send now" intent was honoured; per-participant
	// delivery state surfaces via the status chip + retry button.
	if err := uc.sessionRepo.MarkNotified(ctx, sessionID, time.Now()); err != nil {
		// Best-effort: a failure here means the gate doesn't latch
		// but the messages still went out, which is the safer
		// failure direction (host can re-fire if they want; the
		// rate-limit on individual reminders still applies).
		_ = err
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
	return fmt.Sprintf("Hello, %s 👋\n\n%s has added you to a bill split: %s\n\nBill Summary\n• Total Amount: Rp%s\n• Your Share: Rp%s\n\nReview the bill and upload your payment proof:\n%s\n\nIf you have any questions, contact %s directly.\n\n— Letpai · Bill Split\nhttps://letpai.app",
		v.ParticipantName, v.MakerName, v.SessionName, v.Total, v.Share, v.URL, v.MakerName)
}
