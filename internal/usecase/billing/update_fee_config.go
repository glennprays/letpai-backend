package billing

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// UpdateFeeConfigRequest is the request body for updating fee configuration.
type UpdateFeeConfigRequest struct {
	ServiceChargePercentage *float64 `json:"service_charge_percentage,omitempty" validate:"omitempty,min=0,max=100"`
	TaxPercentage           *float64 `json:"tax_percentage,omitempty" validate:"omitempty,min=0,max=100"`
}

// FeeConfigResponse is the response for fee configuration.
type FeeConfigResponse struct {
	ServiceChargePercentage float64 `json:"service_charge_percentage"`
	TaxPercentage           float64 `json:"tax_percentage"`
}

// UpdateFeeConfigUseCase handles updating the fee configuration for a session.
type UpdateFeeConfigUseCase struct {
	sessionRepo ports.SessionRepository
}

// NewUpdateFeeConfigUseCase creates a new update fee config use case.
func NewUpdateFeeConfigUseCase(
	sessionRepo ports.SessionRepository,
) *UpdateFeeConfigUseCase {
	return &UpdateFeeConfigUseCase{
		sessionRepo: sessionRepo,
	}
}

// Execute updates the fee configuration for a session.
func (uc *UpdateFeeConfigUseCase) Execute(ctx context.Context, userID, sessionID string, req *UpdateFeeConfigRequest) (*FeeConfigResponse, error) {
	// Verify session exists and belongs to user
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	if !session.IsActive() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("cannot update fee config for a completed or cancelled session"))
	}

	// Build new fee values: use provided values or keep existing
	serviceCharge := session.ServiceChargePercentage
	tax := session.TaxPercentage
	if req.ServiceChargePercentage != nil {
		serviceCharge = *req.ServiceChargePercentage
	}
	if req.TaxPercentage != nil {
		tax = *req.TaxPercentage
	}

	// Update the session entity
	session.SetFeeConfig(serviceCharge, tax)

	// Persist
	if err := uc.sessionRepo.Update(ctx, session); err != nil {
		return nil, err
	}

	return &FeeConfigResponse{
		ServiceChargePercentage: session.ServiceChargePercentage,
		TaxPercentage:           session.TaxPercentage,
	}, nil
}
