package session

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/ports"
)

// ParticipantItem represents a participant in session detail
type ParticipantItem struct {
	ParticipantID   string  `json:"participant_id"`
	ContactID       *string `json:"contact_id,omitempty"`
	Name            string  `json:"name"`
	WhatsAppNumber  string  `json:"whatsapp_number"`
	AvatarURL       *string `json:"avatar_url,omitempty"`
	ShareAmount     float64 `json:"share_amount"`
	PaymentStatus   string  `json:"payment_status"`
	PaymentProofURL *string `json:"payment_proof_url,omitempty"`
}

// BillItemItem represents a bill item in session detail
type BillItemItem struct {
	BillItemID  string  `json:"bill_item_id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Category    *string `json:"category,omitempty"`
}

// GetSessionDetailResponse represents the response for getting session details
type GetSessionDetailResponse struct {
	SessionID        string             `json:"session_id"`
	Title            string             `json:"title"`
	Description      string             `json:"description"`
	Status           string             `json:"status"`
	TotalAmount      float64            `json:"total_amount"`
	Currency         string             `json:"currency"`
	SessionDate      *string            `json:"session_date,omitempty"`
	CreatedAt        string             `json:"created_at"`
	UpdatedAt        string             `json:"updated_at"`
	ParticipantCount int                `json:"participant_count"`
	BillItemCount    int                `json:"bill_item_count"`
	PaidCount        int                `json:"paid_count"`
	Participants     []*ParticipantItem `json:"participants"`
	Bills            []*BillItemItem    `json:"bills"`
}

// GetSessionDetailUseCase handles retrieving session details
type GetSessionDetailUseCase struct {
	sessionRepo     ports.SessionRepository
	participantRepo ports.ParticipantRepository
	billItemRepo    ports.BillItemRepository
}

// NewGetSessionDetailUseCase creates a new get session detail use case
func NewGetSessionDetailUseCase(
	sessionRepo ports.SessionRepository,
	participantRepo ports.ParticipantRepository,
	billItemRepo ports.BillItemRepository,
) *GetSessionDetailUseCase {
	return &GetSessionDetailUseCase{
		sessionRepo:     sessionRepo,
		participantRepo: participantRepo,
		billItemRepo:    billItemRepo,
	}
}

// Execute retrieves session details with participants and bills
func (uc *GetSessionDetailUseCase) Execute(ctx context.Context, userID, sessionID string) (*GetSessionDetailResponse, error) {
	// Fetch session
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Fetch participants with contact info
	participants, err := uc.participantRepo.FindBySessionIDWithContactInfo(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Fetch bills
	bills, err := uc.billItemRepo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Count paid participants
	paidCount, err := uc.participantRepo.CountBySessionIDAndStatus(ctx, sessionID, "paid")
	if err != nil {
		paidCount = 0
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

		participantItems = append(participantItems, &ParticipantItem{
			ParticipantID:   p.ParticipantID.String(),
			ContactID:       contactID,
			Name:            name,
			WhatsAppNumber:  whatsappNumber,
			AvatarURL:       avatarURL,
			ShareAmount:     p.ShareAmount,
			PaymentStatus:   p.PaymentStatus.String(),
			PaymentProofURL: p.PaymentProofURL,
		})
	}

	billItems := make([]*BillItemItem, 0, len(bills))
	for _, b := range bills {
		billItems = append(billItems, &BillItemItem{
			BillItemID:  b.BillItemID.String(),
			Description: b.Description,
			Amount:      b.Amount,
			Category:    b.Category,
		})
	}

	return &GetSessionDetailResponse{
		SessionID:        session.SessionID.String(),
		Title:            session.Title,
		Description:      session.Description,
		Status:           session.Status.String(),
		TotalAmount:      session.TotalAmount,
		Currency:         session.Currency,
		SessionDate:      sessionDate,
		CreatedAt:        session.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        session.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		ParticipantCount: len(participants),
		BillItemCount:    len(bills),
		PaidCount:        paidCount,
		Participants:     participantItems,
		Bills:            billItems,
	}, nil
}
