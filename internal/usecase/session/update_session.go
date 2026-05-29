package session

import (
	"context"
	"errors"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// UpdateSessionRequest represents the request to update a session
type UpdateSessionRequest struct {
	Title             *string    `json:"title" validate:"omitempty,min=1,max=200"`
	Description       *string    `json:"description" validate:"omitempty,max=1000"`
	Currency          *string    `json:"currency" validate:"omitempty,len=3"`
	SessionDate       *time.Time `json:"session_date,omitempty"`
	BankName          *string    `json:"bank_name,omitempty"`
	BankAccountNumber *string    `json:"bank_account_number,omitempty"`
	BankAccountHolder *string    `json:"bank_account_holder,omitempty"`
}

// UpdateSessionResponse represents the response after updating a session
type UpdateSessionResponse struct {
	SessionID         string  `json:"session_id"`
	Title             string  `json:"title"`
	Description       string  `json:"description"`
	Status            string  `json:"status"`
	TotalAmount       float64 `json:"total_amount"`
	Currency          string  `json:"currency"`
	SessionDate       *string `json:"session_date,omitempty"`
	BankName          *string `json:"bank_name,omitempty"`
	BankAccountNumber *string `json:"bank_account_number,omitempty"`
	BankAccountHolder *string `json:"bank_account_holder,omitempty"`
	UpdatedAt         string  `json:"updated_at"`
}

// UpdateSessionUseCase handles updating a session
type UpdateSessionUseCase struct {
	sessionRepo ports.SessionRepository
}

// NewUpdateSessionUseCase creates a new update session use case
func NewUpdateSessionUseCase(
	sessionRepo ports.SessionRepository,
) *UpdateSessionUseCase {
	return &UpdateSessionUseCase{
		sessionRepo: sessionRepo,
	}
}

// Execute updates a session
func (uc *UpdateSessionUseCase) Execute(ctx context.Context, userID, sessionID string, req *UpdateSessionRequest) (*UpdateSessionResponse, error) {
	// Fetch existing session
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Only allow updates for active sessions
	if !session.IsActive() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("cannot update a completed or cancelled session"))
	}

	// Validate currency if provided
	if req.Currency != nil && len(*req.Currency) != 3 {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("invalid currency code"))
	}

	// Update session
	session.Update(
		coalesceString(req.Title, session.Title),
		coalesceString(req.Description, session.Description),
		coalesceString(req.Currency, session.Currency),
		req.SessionDate,
	)
	session.SetBankInfo(req.BankName, req.BankAccountNumber, req.BankAccountHolder)

	if err := uc.sessionRepo.Update(ctx, session); err != nil {
		return nil, err
	}

	var sessionDate *string
	if session.SessionDate != nil {
		sd := session.SessionDate.Format("2006-01-02T15:04:05Z07:00")
		sessionDate = &sd
	}

	return &UpdateSessionResponse{
		SessionID:         session.SessionID.String(),
		Title:             session.Title,
		Description:       session.Description,
		Status:            session.Status.String(),
		TotalAmount:       session.TotalAmount,
		Currency:          session.Currency,
		SessionDate:       sessionDate,
		BankName:          session.BankName,
		BankAccountNumber: session.BankAccountNumber,
		BankAccountHolder: session.BankAccountHolder,
		UpdatedAt:         session.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func coalesceString(s *string, defaultVal string) string {
	if s != nil {
		return *s
	}
	return defaultVal
}
