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

// RetryNotificationResponse mirrors the queued shape used by the
// bulk send path so the FE can render the same chip transitions.
type RetryNotificationResponse struct {
	Message   string `json:"message"`
	Status    string `json:"status"`
	Type      string `json:"type"`
	QueuedAt  string `json:"queued_at"`
}

// RetryNotificationUseCase re-dispatches the most recent notification
// for a participant. Powers the per-participant "Retry" button next
// to a Failed / Stuck delivery chip.
//
// NOT subject to the session-level dirty-for-notify gate (the gate
// exists to stop a host from re-firing the whole batch repeatedly;
// an individual retry of a failed delivery is exactly the case we
// want to allow). The reminder rate-limit still applies to
// notifications of type=reminder.
type RetryNotificationUseCase struct {
	participantRepo  ports.ParticipantRepository
	sessionRepo      ports.SessionRepository
	contactRepo      ports.ContactRepository
	billItemRepo     ports.BillItemRepository
	logRepo          ports.NotificationLogRepository
	userRepo         ports.UserRepository
	notifier         *service.AsyncNotifier
	renderer         *service.TemplateRenderer
	rateLimitService *service.RateLimitService
	appURL           string
}

func NewRetryNotificationUseCase(
	participantRepo ports.ParticipantRepository,
	sessionRepo ports.SessionRepository,
	contactRepo ports.ContactRepository,
	billItemRepo ports.BillItemRepository,
	logRepo ports.NotificationLogRepository,
	userRepo ports.UserRepository,
	notifier *service.AsyncNotifier,
	renderer *service.TemplateRenderer,
	rateLimitService *service.RateLimitService,
	appURL string,
) *RetryNotificationUseCase {
	return &RetryNotificationUseCase{
		participantRepo:  participantRepo,
		sessionRepo:      sessionRepo,
		contactRepo:      contactRepo,
		billItemRepo:     billItemRepo,
		logRepo:          logRepo,
		userRepo:         userRepo,
		notifier:         notifier,
		renderer:         renderer,
		rateLimitService: rateLimitService,
		appURL:           strings.TrimRight(appURL, "/"),
	}
}

// Execute re-fires the participant's most-recent notification of the
// matching type. If there's no prior log we default to the initial
// notification body (covers the "Not sent yet" → "Retry" edge case
// even though the UI normally would route those through the bulk
// send endpoint).
func (uc *RetryNotificationUseCase) Execute(ctx context.Context, userID, participantID string) (*RetryNotificationResponse, error) {
	participant, err := uc.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return nil, err
	}

	session, err := uc.sessionRepo.FindByID(ctx, participant.SessionID.String(), userID)
	if err != nil {
		return nil, err
	}

	if participant.IsPaid() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("participant has already paid"))
	}

	whatsappNumber := participant.GetWhatsAppNumber()
	participantName := participant.GetName()
	if whatsappNumber == "" {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("no WhatsApp number on file"))
	}

	// Default to "initial" if there's no prior log, otherwise mirror
	// the most-recent log's type so a failed reminder retries as a
	// reminder (and is rate-limit-checked accordingly).
	notifType := valueobject.NotificationTypeInitial
	if last, _ := uc.logRepo.FindLatestByParticipantID(ctx, participantID); last != nil {
		notifType = last.NotificationType
	}

	// Reminder rate-limit still applies on a reminder retry —
	// otherwise the retry button is a trivial bypass for the 24h
	// cap. The initial-notification retry does NOT consult the
	// reminder limiter (different operation).
	if notifType == valueobject.NotificationTypeReminder {
		result, rlErr := uc.rateLimitService.CheckReminderRateLimit(ctx, participantID)
		if rlErr == nil && !result.Allowed {
			retryAfterSec := int(result.RetryAfter.Seconds())
			next := time.Now().Add(result.RetryAfter).UTC().Format(time.RFC3339)
			rateErr := fmt.Errorf("reminder cooldown active; try again at %s", next)
			_ = retryAfterSec
			return nil, domain.NewError(domain.ErrConflict, rateErr)
		}
	}

	// Re-render against the live template + current totals (the
	// stored MessageContent from the last log is a snapshot — we
	// want today's URL, today's share amount).
	totalAmount := session.TotalAmount
	if totalAmount == 0 {
		items, _ := uc.billItemRepo.FindBySessionID(ctx, participant.SessionID.String())
		for _, b := range items {
			totalAmount += b.Amount
		}
	}

	// Resolve the session host's display name for MakerName.
	makerName := "the host"
	if host, err := uc.userRepo.FindByID(ctx, session.UserID.String()); err == nil && host.FullName != "" {
		makerName = host.FullName
	}

	url := uc.appURL + "/payment/" + participant.PublicSlug
	var message string
	switch notifType {
	case valueobject.NotificationTypeReminder:
		vars := reminderVars{
			ParticipantName: participantName,
			SessionName:     session.Title,
			MakerName:       makerName,
			Share:           formatIDR(participant.ShareAmount),
			URL:             url,
		}
		rendered, rErr := uc.renderer.Render(ctx, "payment_reminder", vars)
		if rErr != nil || rendered == "" {
			rendered = fallbackReminder(vars)
		}
		message = rendered
	default:
		vars := notificationVars{
			ParticipantName: participantName,
			SessionName:     session.Title,
			MakerName:       makerName,
			Total:           formatIDR(totalAmount),
			Share:           formatIDR(participant.ShareAmount),
			URL:             url,
		}
		rendered, rErr := uc.renderer.Render(ctx, "session_notification", vars)
		if rErr != nil || rendered == "" {
			rendered = fallbackSessionNotification(vars)
		}
		message = rendered
	}

	uc.notifier.Dispatch(participant.ParticipantID, notifType, whatsappNumber, message)

	return &RetryNotificationResponse{
		Message:  "Retry queued",
		Status:   "queued",
		Type:     notifType.String(),
		QueuedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}
