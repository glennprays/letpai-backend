package session

import (
	"context"
	"errors"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
)

// CreateSessionRequest represents the request to create a session
type CreateSessionRequest struct {
	Title             string     `json:"title" validate:"required,min=1,max=200"`
	Description       string     `json:"description" validate:"max=1000"`
	Currency          string     `json:"currency" validate:"required,len=3"`
	SessionDate       *time.Time `json:"session_date,omitempty"`
	BankName          *string    `json:"bank_name,omitempty"`
	BankAccountNumber *string    `json:"bank_account_number,omitempty"`
	BankAccountHolder *string    `json:"bank_account_holder,omitempty"`
}

// CreateSessionResponse represents the response after creating a session
type CreateSessionResponse struct {
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
	CreatedAt         string  `json:"created_at"`
}

// CreateSessionUseCase handles creating a new session
type CreateSessionUseCase struct {
	sessionRepo ports.SessionRepository
}

// NewCreateSessionUseCase creates a new create session use case
func NewCreateSessionUseCase(
	sessionRepo ports.SessionRepository,
) *CreateSessionUseCase {
	return &CreateSessionUseCase{
		sessionRepo: sessionRepo,
	}
}

// Execute creates a new session
func (uc *CreateSessionUseCase) Execute(ctx context.Context, userID string, req *CreateSessionRequest) (*CreateSessionResponse, error) {
	// Parse userID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("invalid user ID"))
	}

	// Validate currency
	if len(req.Currency) != 3 {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("invalid currency code"))
	}

	// Create session
	session := entity.NewSession(userUUID, req.Title, req.Description, req.Currency, req.SessionDate)
	session.SetBankInfo(req.BankName, req.BankAccountNumber, req.BankAccountHolder)

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	var sessionDate *string
	if session.SessionDate != nil {
		sd := session.SessionDate.Format("2006-01-02T15:04:05Z07:00")
		sessionDate = &sd
	}

	return &CreateSessionResponse{
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
		CreatedAt:         session.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
