package notification

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
	"github.com/glennprays/letpai-backend/internal/service"
)

// SendReminderResponse represents the response after queueing a reminder
type SendReminderResponse struct {
	Message         string `json:"message"`
	Status          string `json:"status"`
	NextAvailableAt string `json:"next_available_at"`
	RetryAfter      int64  `json:"retry_after,omitempty"`
}

// SendReminderUseCase handles sending reminder to a specific participant
type SendReminderUseCase struct {
	participantRepo ports.ParticipantRepository
	sessionRepo     ports.SessionRepository
	contactRepo     ports.ContactRepository
	userRepo        ports.UserRepository
	notifier        *service.AsyncNotifier
	renderer        *service.TemplateRenderer
	rateLimitSvc    *service.RateLimitService
	appURL          string
}

// NewSendReminderUseCase creates a new send reminder use case
func NewSendReminderUseCase(
	participantRepo ports.ParticipantRepository,
	sessionRepo ports.SessionRepository,
	contactRepo ports.ContactRepository,
	userRepo ports.UserRepository,
	notifier *service.AsyncNotifier,
	renderer *service.TemplateRenderer,
	rateLimitSvc *service.RateLimitService,
	appURL string,
) *SendReminderUseCase {
	return &SendReminderUseCase{
		participantRepo: participantRepo,
		sessionRepo:     sessionRepo,
		contactRepo:     contactRepo,
		userRepo:        userRepo,
		notifier:        notifier,
		renderer:        renderer,
		rateLimitSvc:    rateLimitSvc,
		appURL:          strings.TrimRight(appURL, "/"),
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

	// Rate-limit gating lives in the Fiber middleware (ReminderRateLimiter
	// in internal/middleware/rate_limit.go), which returns the canonical
	// 429 + Retry-After + next_available_at response. Re-checking here
	// would double-decrement the limiter and ship a divergent 400 payload
	// when the use-case ran first under a race. The status helper below
	// remains for /reminder-status reads.
	_, _, nextResetAt, err := uc.rateLimitSvc.GetReminderStatus(ctx, participantID)
	if err != nil {
		return nil, err
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

	// Resolve the session host's display name for MakerName.
	makerName := "the host"
	if host, err := uc.userRepo.FindByID(ctx, session.UserID.String()); err == nil && host.FullName != "" {
		makerName = host.FullName
	}

	// Render reminder via the admin-managed template; fall back to a
	// hardcoded body if the template is missing/broken.
	vars := reminderVars{
		ParticipantName: participantName,
		SessionName:     session.Title,
		MakerName:       makerName,
		Share:           formatIDR(participant.ShareAmount),
		URL:             uc.appURL + "/payment/" + participant.PublicSlug,
	}
	message, err := uc.renderer.Render(ctx, "payment_reminder", vars)
	if err != nil || message == "" {
		message = fallbackReminder(vars)
	}

	// Queue the gateway send; record rate-limit + counter immediately.
	uc.notifier.Dispatch(participant.ParticipantID, valueobject.NotificationTypeReminder, whatsappNumber, message)
	_ = uc.rateLimitSvc.RecordReminder(ctx, participantID)
	participant.BumpNotificationCount()
	_ = uc.participantRepo.Update(ctx, participant)

	return &SendReminderResponse{
		Message:         fmt.Sprintf("Reminder queued for %s", participantName),
		Status:          "queued",
		NextAvailableAt: nextResetAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// reminderVars is the template data context for the payment_reminder
// template. The fields must match the {{.Field}} placeholders in the
// admin-authored body.
type reminderVars struct {
	ParticipantName string
	SessionName     string
	MakerName       string
	Share           string
	URL             string
}

func fallbackReminder(v reminderVars) string {
	return fmt.Sprintf("Hey %s, just a reminder! 👋\n\n%s is waiting for your payment on: %s\n\nAmount Due: Rp%s\n\nUpload your payment proof here:\n%s\n\nThanks!\n— Letpai\nhttps://letpai.app",
		v.ParticipantName, v.MakerName, v.SessionName, v.Share, v.URL)
}
