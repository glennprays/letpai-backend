package payment

import (
	"context"
	"time"

	"github.com/glennprays/letpai-backend/domain/ports"
)

// PaymentPageResponse represents the public payment page data
type PaymentPageResponse struct {
	SessionName     string         `json:"session_name"`
	ParticipantName string         `json:"participant_name"`
	ShareAmount     float64        `json:"share_amount"`
	Currency        string         `json:"currency"`
	BillItems       []BillItemInfo `json:"bill_items"`
	PaymentStatus   string         `json:"payment_status"`
	LinkExpiresAt   string         `json:"link_expires_at"`
	IsExpired       bool           `json:"is_expired"`
}

// BillItemInfo represents bill item information
type BillItemInfo struct {
	BillItemID  string  `json:"bill_item_id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Category    *string `json:"category,omitempty"`
}

// GetPaymentPageUseCase handles getting public payment page data
type GetPaymentPageUseCase struct {
	participantRepo ports.ParticipantRepository
	sessionRepo     ports.SessionRepository
	billItemRepo    ports.BillItemRepository
	contactRepo     ports.ContactRepository
}

// NewGetPaymentPageUseCase creates a new get payment page use case
func NewGetPaymentPageUseCase(
	participantRepo ports.ParticipantRepository,
	sessionRepo ports.SessionRepository,
	billItemRepo ports.BillItemRepository,
	contactRepo ports.ContactRepository,
) *GetPaymentPageUseCase {
	return &GetPaymentPageUseCase{
		participantRepo: participantRepo,
		sessionRepo:     sessionRepo,
		billItemRepo:    billItemRepo,
		contactRepo:     contactRepo,
	}
}

// Execute returns payment page data for a participant (public endpoint, no auth required)
func (uc *GetPaymentPageUseCase) Execute(ctx context.Context, participantID string) (*PaymentPageResponse, error) {
	// Get participant
	participant, err := uc.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return nil, err
	}

	// Get session
	session, err := uc.sessionRepo.FindByID(ctx, participant.SessionID.String(), "")
	if err != nil {
		return nil, err
	}

	// Get participant name
	participantName := ""
	if participant.ContactID != nil {
		contact, err := uc.contactRepo.FindByID(ctx, participant.ContactID.String(), session.UserID.String())
		if err == nil {
			participantName = contact.Name
		}
	} else {
		participantName = participant.CustomName
	}

	// Get bill items for the session
	billItems, err := uc.billItemRepo.FindBySessionID(ctx, participant.SessionID.String())
	if err != nil {
		return nil, err
	}

	billItemInfos := make([]BillItemInfo, len(billItems))
	for i, bill := range billItems {
		billItemInfos[i] = BillItemInfo{
			BillItemID:  bill.BillItemID.String(),
			Description: bill.Description,
			Amount:      bill.Amount,
			Category:    bill.Category,
		}
	}

	// Calculate link expiry (7 days from session creation or completion)
	linkExpiresAt := session.CreatedAt.Add(7 * 24 * time.Hour)
	isExpired := time.Now().After(linkExpiresAt)

	return &PaymentPageResponse{
		SessionName:     session.Title,
		ParticipantName: participantName,
		ShareAmount:     participant.ShareAmount,
		Currency:        session.Currency,
		BillItems:       billItemInfos,
		PaymentStatus:   participant.PaymentStatus.String(),
		LinkExpiresAt:   linkExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		IsExpired:       isExpired,
	}, nil
}
