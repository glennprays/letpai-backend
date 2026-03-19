package session

import (
	"context"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
)

// SessionItem represents a session item in the list
type SessionItem struct {
	SessionID        string    `json:"session_id"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Status           string    `json:"status"`
	TotalAmount      float64   `json:"total_amount"`
	Currency         string    `json:"currency"`
	SessionDate      *string   `json:"session_date,omitempty"`
	CreatedAt        string    `json:"created_at"`
}

// GetSessionsRequest represents the request to get sessions
type GetSessionsRequest struct {
	Status    *string
	Search    *string
	SortBy    string
	SortOrder string
	Page      int
	Limit     int
}

// GetSessionsResponse represents the response for listing sessions
type GetSessionsResponse struct {
	Sessions []*SessionItem `json:"sessions"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	Limit    int            `json:"limit"`
}

// GetSessionsUseCase handles retrieving sessions
type GetSessionsUseCase struct {
	sessionRepo ports.SessionRepository
}

// NewGetSessionsUseCase creates a new get sessions use case
func NewGetSessionsUseCase(
	sessionRepo ports.SessionRepository,
) *GetSessionsUseCase {
	return &GetSessionsUseCase{
		sessionRepo: sessionRepo,
	}
}

// Execute retrieves sessions with filters and pagination
func (uc *GetSessionsUseCase) Execute(ctx context.Context, userID string, req *GetSessionsRequest) (*GetSessionsResponse, error) {
	// Parse status filter
	var statusFilter *valueobject.SessionStatus
	if req.Status != nil && *req.Status != "" {
		status, err := valueobject.ParseSessionStatus(*req.Status)
		if err != nil {
			return nil, domain.NewError(domain.ErrBadRequest, err)
		}
		statusFilter = &status
	}

	opts := &ports.SessionFilterOptions{
		Status:    statusFilter,
		Search:    req.Search,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Page:      req.Page,
		Limit:     req.Limit,
	}

	result, err := uc.sessionRepo.FindAll(ctx, userID, opts)
	if err != nil {
		return nil, err
	}

	sessions := make([]*SessionItem, 0, len(result.Sessions))
	for _, s := range result.Sessions {
		var sessionDate *string
		if s.SessionDate != nil {
			sd := s.SessionDate.Format("2006-01-02T15:04:05Z07:00")
			sessionDate = &sd
		}

		sessions = append(sessions, &SessionItem{
			SessionID:   s.SessionID.String(),
			Title:       s.Title,
			Description: s.Description,
			Status:      s.Status.String(),
			TotalAmount: s.TotalAmount,
			Currency:    s.Currency,
			SessionDate: sessionDate,
			CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return &GetSessionsResponse{
		Sessions: sessions,
		Total:    result.Total,
		Page:     req.Page,
		Limit:    req.Limit,
	}, nil
}
