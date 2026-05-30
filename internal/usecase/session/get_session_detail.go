package session

import (
	"context"
	"time"

	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/usecase/idresolve"
)

// LastNotification mirrors the most recent notification_logs row for
// a participant. `FriendlyStatus` is mapped at the use-case boundary
// (not on the entity) because the "queued > 2 minutes → Stuck" rule
// is a presentation policy, not a domain invariant.
type LastNotification struct {
	LogID          string  `json:"log_id"`
	Type           string  `json:"type"`
	Status         string  `json:"status"`
	SentAt         string  `json:"sent_at"`
	ErrorMessage   *string `json:"error_message,omitempty"`
	FriendlyStatus string  `json:"friendly_status"`
}

// ParticipantItem represents a participant in session detail.
//
// PaidManually surfaces whether the host closed this participant out
// without a proof upload. NotificationCount / LastNotificationAt mirror
// the participant row so the host UI can show "Reminded 2× — Last 4h
// ago" and derive cooldown state without an extra round trip.
type ParticipantItem struct {
	ParticipantID      string            `json:"participant_id"`
	PublicSlug         string            `json:"public_slug"`
	ContactID          *string           `json:"contact_id,omitempty"`
	Name               string            `json:"name"`
	WhatsAppNumber     string            `json:"whatsapp_number"`
	AvatarURL          *string           `json:"avatar_url,omitempty"`
	ShareAmount        float64           `json:"share_amount"`
	PaymentStatus      string            `json:"payment_status"`
	PaymentProofURL    *string           `json:"payment_proof_url,omitempty"`
	PaidManually       bool              `json:"paid_manually"`
	NotificationCount  int               `json:"notification_count"`
	LastNotificationAt *time.Time        `json:"last_notification_at,omitempty"`
	LastNotification   *LastNotification `json:"last_notification,omitempty"`
}

// BillItemItem represents a bill item in session detail.
//
// ParticipantIDs is the per-bill participant assignment; an empty slice
// means "applies to everyone in the session" (legacy default).
type BillItemItem struct {
	BillItemID     string   `json:"bill_item_id"`
	Description    string   `json:"description"`
	Amount         float64  `json:"amount"`
	Category       *string  `json:"category,omitempty"`
	ParticipantIDs []string `json:"participant_ids"`
}

// GetSessionDetailResponse represents the response for getting session details
type GetSessionDetailResponse struct {
	SessionID         string             `json:"session_id"`
	PublicSlug        string             `json:"public_slug"`
	Title             string             `json:"title"`
	Description       string             `json:"description"`
	Status            string             `json:"status"`
	TotalAmount       float64            `json:"total_amount"`
	Currency          string             `json:"currency"`
	SessionDate       *string            `json:"session_date,omitempty"`
	BankName          *string            `json:"bank_name,omitempty"`
	BankAccountNumber *string            `json:"bank_account_number,omitempty"`
	BankAccountHolder *string            `json:"bank_account_holder,omitempty"`
	BankAccounts      []*BankAccountItem `json:"bank_accounts"`
	LastNotifiedAt    *string            `json:"last_notified_at,omitempty"`
	IsDirty           bool               `json:"is_dirty"`
	CreatedAt         string             `json:"created_at"`
	UpdatedAt         string             `json:"updated_at"`
	ParticipantCount  int                `json:"participant_count"`
	BillItemCount     int                `json:"bill_item_count"`
	PaidCount         int                `json:"paid_count"`
	Participants      []*ParticipantItem `json:"participants"`
	Bills             []*BillItemItem    `json:"bills"`
}

// GetSessionDetailUseCase handles retrieving session details
type GetSessionDetailUseCase struct {
	sessionRepo         ports.SessionRepository
	participantRepo     ports.ParticipantRepository
	billItemRepo        ports.BillItemRepository
	notificationLogRepo ports.NotificationLogRepository
	bankAccountRepo     ports.SessionBankAccountRepository
}

// NewGetSessionDetailUseCase creates a new get session detail use case
func NewGetSessionDetailUseCase(
	sessionRepo ports.SessionRepository,
	participantRepo ports.ParticipantRepository,
	billItemRepo ports.BillItemRepository,
	notificationLogRepo ports.NotificationLogRepository,
	bankAccountRepo ports.SessionBankAccountRepository,
) *GetSessionDetailUseCase {
	return &GetSessionDetailUseCase{
		sessionRepo:         sessionRepo,
		participantRepo:     participantRepo,
		billItemRepo:        billItemRepo,
		notificationLogRepo: notificationLogRepo,
		bankAccountRepo:     bankAccountRepo,
	}
}

// hasLegacyBank reports whether any of the three legacy
// sessions.bank_* fields is non-nil and non-blank. Used as the
// read-fallback predicate when session_bank_accounts has no rows
// (pre-migration data the backfill missed).
func hasLegacyBank(name, number, holder *string) bool {
	for _, p := range []*string{name, number, holder} {
		if p != nil && *p != "" {
			return true
		}
	}
	return false
}

// friendlyNotificationStatus renders the gateway-jargon log status as
// a UI-stable label. Kept on the use case rather than the entity
// because the 2-minute stuck threshold and the "tap to retry"
// language are presentation policy, not domain truth.
func friendlyNotificationStatus(status entity.NotificationStatus, age time.Duration) string {
	switch status {
	case entity.NotificationStatusSent:
		return "Sent"
	case entity.NotificationStatusQueued:
		if age > 2*time.Minute {
			return "Stuck — tap to retry"
		}
		return "Sending…"
	case entity.NotificationStatusFailed:
		return "Failed — tap to retry"
	}
	return "Unknown"
}

// Execute retrieves session details with participants and bills
func (uc *GetSessionDetailUseCase) Execute(ctx context.Context, userID, sessionID string) (*GetSessionDetailResponse, error) {
	// Fetch session
	// Accept either the legacy UUID or the new public_slug; old links
	// in the wild stay resolvable through the compat window.
	session, err := idresolve.ResolveSession(ctx, uc.sessionRepo, sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Fetch participants with contact info
	resolvedSessionID := session.SessionID.String()

	participants, err := uc.participantRepo.FindBySessionIDWithContactInfo(ctx, resolvedSessionID)
	if err != nil {
		return nil, err
	}

	// Fetch bills
	bills, err := uc.billItemRepo.FindBySessionID(ctx, resolvedSessionID)
	if err != nil {
		return nil, err
	}

	// Count paid participants
	paidCount, err := uc.participantRepo.CountBySessionIDAndStatus(ctx, resolvedSessionID, "paid")
	if err != nil {
		paidCount = 0
	}

	// Latest notification per participant. Best-effort: a failure here
	// just leaves last_notification absent on the response (the chip
	// falls back to "Not sent yet").
	latestLogs, _ := uc.notificationLogRepo.FindLatestPerParticipantBySessionID(ctx, resolvedSessionID)

	// Per-session bank accounts. Read from the new table; if it's
	// empty AND the legacy single-row columns have any value, fall
	// back to synthesising a single ordinal=0 entry from the legacy
	// fields. That handles the "session created pre-migration and
	// never edited since" tail.
	bankAccounts, _ := uc.bankAccountRepo.FindBySessionID(ctx, resolvedSessionID)
	if len(bankAccounts) == 0 && hasLegacyBank(session.BankName, session.BankAccountNumber, session.BankAccountHolder) {
		bankAccounts = []*entity.SessionBankAccount{{
			Ordinal:       0,
			BankName:      session.BankName,
			AccountNumber: session.BankAccountNumber,
			AccountHolder: session.BankAccountHolder,
		}}
	}
	bankAccountItems := make([]*BankAccountItem, 0, len(bankAccounts))
	for _, a := range bankAccounts {
		bankAccountItems = append(bankAccountItems, &BankAccountItem{
			AccountID:     a.AccountID.String(),
			Ordinal:       a.Ordinal,
			BankName:      a.BankName,
			AccountNumber: a.AccountNumber,
			AccountHolder: a.AccountHolder,
		})
	}

	// Build response
	var sessionDate *string
	if session.SessionDate != nil {
		sd := session.SessionDate.Format("2006-01-02T15:04:05Z07:00")
		sessionDate = &sd
	}

	participantItems := make([]*ParticipantItem, 0, len(participants))
	for _, p := range participants {
		var contactID *string
		if p.ContactID != nil {
			cid := p.ContactID.String()
			contactID = &cid
		}

		// Determine name and WhatsApp number
		name := p.CustomName
		whatsappNumber := p.CustomWhatsApp
		avatarURL := p.ContactAvatarURL

		var lastNotif *LastNotification
		if log := latestLogs[p.ParticipantID]; log != nil {
			age := time.Since(log.SentAt)
			lastNotif = &LastNotification{
				LogID:          log.LogID.String(),
				Type:           log.NotificationType.String(),
				Status:         string(log.Status),
				SentAt:         log.SentAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
				ErrorMessage:   log.ErrorMessage,
				FriendlyStatus: friendlyNotificationStatus(log.Status, age),
			}
		}

		participantItems = append(participantItems, &ParticipantItem{
			ParticipantID:      p.ParticipantID.String(),
			PublicSlug:         p.PublicSlug,
			ContactID:          contactID,
			Name:               name,
			WhatsAppNumber:     whatsappNumber,
			AvatarURL:          avatarURL,
			ShareAmount:        p.ShareAmount,
			PaymentStatus:      p.PaymentStatus.String(),
			PaymentProofURL:    p.PaymentProofURL,
			PaidManually:       p.PaidManually,
			NotificationCount:  p.NotificationCount,
			LastNotificationAt: p.LastNotificationAt,
			LastNotification:   lastNotif,
		})
	}

	billItems := make([]*BillItemItem, 0, len(bills))
	for _, b := range bills {
		pidStrs := make([]string, 0, len(b.ParticipantIDs))
		for _, id := range b.ParticipantIDs {
			pidStrs = append(pidStrs, id.String())
		}
		billItems = append(billItems, &BillItemItem{
			BillItemID:     b.BillItemID.String(),
			Description:    b.Description,
			Amount:         b.Amount,
			Category:       b.Category,
			ParticipantIDs: pidStrs,
		})
	}

	var lastNotifiedStr *string
	if session.LastNotifiedAt != nil {
		s := session.LastNotifiedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		lastNotifiedStr = &s
	}

	// Compute dirty for the FE so it can flip the Send/Resend button
	// label without re-deriving the predicate. Treat the participant
	// list we already have in memory as authoritative (cheaper than
	// re-counting via the repo).
	hasUnpaid := false
	for _, p := range participants {
		if !p.IsPaid() {
			hasUnpaid = true
			break
		}
	}

	return &GetSessionDetailResponse{
		SessionID:         session.SessionID.String(),
		PublicSlug:        session.PublicSlug,
		Title:             session.Title,
		Description:       session.Description,
		Status:            session.Status.String(),
		TotalAmount:       session.TotalAmount,
		Currency:          session.Currency,
		SessionDate:       sessionDate,
		BankName:          session.BankName,
		BankAccountNumber: session.BankAccountNumber,
		BankAccountHolder: session.BankAccountHolder,
		BankAccounts:      bankAccountItems,
		LastNotifiedAt:    lastNotifiedStr,
		IsDirty:           session.IsDirtyForNotify(hasUnpaid),
		CreatedAt:         session.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:         session.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		ParticipantCount:  len(participants),
		BillItemCount:     len(bills),
		PaidCount:         paidCount,
		Participants:      participantItems,
		Bills:             billItems,
	}, nil
}
