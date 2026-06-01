package session

import (
	"context"
	"math"
	"time"

	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
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
	LastNotification   *LastNotification        `json:"last_notification,omitempty"`
	FeeBreakdown       *ParticipantFeeBreakdown `json:"fee_breakdown,omitempty"`
}

// BillItemItem represents a bill item in session detail.
//
// ParticipantIDs is the per-bill participant assignment; an empty slice
// means "applies to everyone in the session" (legacy default).
type BillItemItem struct {
	BillItemID            string   `json:"bill_item_id"`
	Description           string   `json:"description"`
	Amount                float64  `json:"amount"`
	Category              *string  `json:"category,omitempty"`
	ParticipantIDs        []string `json:"participant_ids"`
	IncludesServiceCharge bool     `json:"includes_service_charge"`
	IncludesTax           bool     `json:"includes_tax"`
}

// BillImageDetailItem represents a bill image in session detail.
type BillImageDetailItem struct {
	BillImageID  string  `json:"bill_image_id"`
	SessionID    string  `json:"session_id"`
	ImageURL     string  `json:"image_url"`
	ThumbnailURL *string `json:"thumbnail_url,omitempty"`
	FileName     string  `json:"file_name"`
	FileFormat   string  `json:"file_format"`
	FileSize     int64   `json:"file_size"`
	UploadedAt   string  `json:"uploaded_at"`
	Ordinal      int     `json:"ordinal"`
}

// ParticipantFeeBreakdown shows the fee breakdown for a participant.
type ParticipantFeeBreakdown struct {
	ItemsTotal         float64 `json:"items_total"`
	ServiceChargeShare float64 `json:"service_charge_share"`
	TaxShare           float64 `json:"tax_share"`
	Total              float64 `json:"total"`
}

// FeeConfigItem represents the fee configuration for a session.
type FeeConfigItem struct {
	ServiceChargePercentage float64 `json:"service_charge_percentage"`
	TaxPercentage           float64 `json:"tax_percentage"`
}

// GetSessionDetailResponse represents the response for getting session details
type GetSessionDetailResponse struct {
	SessionID         string                 `json:"session_id"`
	PublicSlug        string                 `json:"public_slug"`
	Title             string                 `json:"title"`
	Description       string                 `json:"description"`
	Status            string                 `json:"status"`
	TotalAmount       float64                `json:"total_amount"`
	Currency          string                 `json:"currency"`
	SessionDate       *string                `json:"session_date,omitempty"`
	BankName          *string                `json:"bank_name,omitempty"`
	BankAccountNumber *string                `json:"bank_account_number,omitempty"`
	BankAccountHolder *string                `json:"bank_account_holder,omitempty"`
	BankAccounts      []*BankAccountItem     `json:"bank_accounts"`
	LastNotifiedAt    *string                `json:"last_notified_at,omitempty"`
	IsDirty           bool                   `json:"is_dirty"`
	CreatedAt         string                 `json:"created_at"`
	UpdatedAt         string                 `json:"updated_at"`
	ParticipantCount  int                    `json:"participant_count"`
	BillItemCount     int                    `json:"bill_item_count"`
	PaidCount         int                    `json:"paid_count"`
	Participants      []*ParticipantItem     `json:"participants"`
	Bills             []*BillItemItem        `json:"bills"`
	BillImages        []*BillImageDetailItem `json:"bill_images"`
	FeeConfig         *FeeConfigItem         `json:"fee_config,omitempty"`
}

// GetSessionDetailUseCase handles retrieving session details
type GetSessionDetailUseCase struct {
	sessionRepo         ports.SessionRepository
	participantRepo     ports.ParticipantRepository
	billItemRepo        ports.BillItemRepository
	notificationLogRepo ports.NotificationLogRepository
	bankAccountRepo     ports.SessionBankAccountRepository
	billImageRepo       ports.BillImageRepository
	imageService        *service.ImageService
}

// NewGetSessionDetailUseCase creates a new get session detail use case
func NewGetSessionDetailUseCase(
	sessionRepo ports.SessionRepository,
	participantRepo ports.ParticipantRepository,
	billItemRepo ports.BillItemRepository,
	notificationLogRepo ports.NotificationLogRepository,
	bankAccountRepo ports.SessionBankAccountRepository,
	billImageRepo ports.BillImageRepository,
	imageService *service.ImageService,
) *GetSessionDetailUseCase {
	return &GetSessionDetailUseCase{
		sessionRepo:         sessionRepo,
		participantRepo:     participantRepo,
		billItemRepo:        billItemRepo,
		notificationLogRepo: notificationLogRepo,
		bankAccountRepo:     bankAccountRepo,
		billImageRepo:       billImageRepo,
		imageService:        imageService,
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

	// Bill images (best-effort — don't fail the whole request if this errors)
	var billImageItems []*BillImageDetailItem
	billImages, imgErr := uc.billImageRepo.FindBySessionID(ctx, resolvedSessionID)
	if imgErr == nil && len(billImages) > 0 {
		billImageItems = make([]*BillImageDetailItem, 0, len(billImages))
		for _, img := range billImages {
			// Resolve a presigned GET URL so the frontend can display the image
			s3Key := "bill-images/" + img.ImageURL
			signedURL, _ := uc.imageService.GetPresignedGetURL(ctx, s3Key, 15*time.Minute)
			item := &BillImageDetailItem{
				BillImageID:  img.BillImageID.String(),
				SessionID:    img.SessionID.String(),
				ImageURL:     signedURL, // presigned URL (or empty if signing failed)
				ThumbnailURL: img.ThumbnailURL,
				FileName:     img.FileName,
				FileFormat:   img.FileFormat,
				FileSize:     img.FileSize,
				UploadedAt:   img.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				Ordinal:      img.Ordinal,
			}
			if signedURL == "" {
				item.ImageURL = img.ImageURL // fallback to raw key
			}
			billImageItems = append(billImageItems, item)
		}
	}

	// Fee config — only include when at least one percentage is non-zero
	var feeConfig *FeeConfigItem
	if session.ServiceChargePercentage > 0 || session.TaxPercentage > 0 {
		feeConfig = &FeeConfigItem{
			ServiceChargePercentage: session.ServiceChargePercentage,
			TaxPercentage:           session.TaxPercentage,
		}
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

		// Compute fee breakdown for this participant when session has fees
		var feeBreakdown *ParticipantFeeBreakdown
		if session.ServiceChargePercentage > 0 || session.TaxPercentage > 0 {
			pidUUID := p.ParticipantID.String()
			itemsTotal := 0.0
			scBase := 0.0
			taxBase := 0.0
			for _, bill := range bills {
				isMine := false
				sharedWith := 0
				if len(bill.ParticipantIDs) == 0 {
					isMine = true
					sharedWith = len(participants)
				} else {
					for _, pid := range bill.ParticipantIDs {
						if pid.String() == pidUUID {
							isMine = true
						}
					}
					sharedWith = len(bill.ParticipantIDs)
				}
				if !isMine || sharedWith == 0 {
					continue
				}
				share := math.Floor(bill.Amount / float64(sharedWith))
				itemsTotal += share
				if bill.IncludesServiceCharge {
					scBase += share
				}
				if bill.IncludesTax {
					taxBase += share
				}
			}
			scShare := math.Round(scBase * session.ServiceChargePercentage / 100)
			taxShare := math.Round(taxBase * session.TaxPercentage / 100)
			feeBreakdown = &ParticipantFeeBreakdown{
				ItemsTotal:         math.Round(itemsTotal),
				ServiceChargeShare: scShare,
				TaxShare:           taxShare,
				Total:              math.Round(itemsTotal) + scShare + taxShare,
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
			FeeBreakdown:       feeBreakdown,
		})
	}

	billItems := make([]*BillItemItem, 0, len(bills))
	for _, b := range bills {
		pidStrs := make([]string, 0, len(b.ParticipantIDs))
		for _, id := range b.ParticipantIDs {
			pidStrs = append(pidStrs, id.String())
		}
		billItems = append(billItems, &BillItemItem{
			BillItemID:            b.BillItemID.String(),
			Description:           b.Description,
			Amount:                b.Amount,
			Category:              b.Category,
			ParticipantIDs:        pidStrs,
			IncludesServiceCharge: b.IncludesServiceCharge,
			IncludesTax:           b.IncludesTax,
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
		BillImages:        billImageItems,
		FeeConfig:         feeConfig,
	}, nil
}
