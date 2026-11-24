package httperror

import (
	"errors"
	"testing"

	"github.com/glennprays/letpai-backend/domain"
)

func TestFromError_StatusTransitionIs400(t *testing.T) {
	apiErr := FromError(domain.NewError(domain.ErrInvalidStatusTransition, errors.New("invalid status transition")))
	if apiErr.Status != 400 {
		t.Errorf("expected 400 for ErrInvalidStatusTransition, got %d", apiErr.Status)
	}
}

func TestFromError_PaymentStatusTransitionIs400(t *testing.T) {
	apiErr := FromError(domain.NewError(domain.ErrInvalidPaymentStatusTransition, errors.New("invalid payment status transition")))
	if apiErr.Status != 400 {
		t.Errorf("expected 400 for ErrInvalidPaymentStatusTransition, got %d", apiErr.Status)
	}
}

func TestFromError_PlainErrorIs500WithoutLeak(t *testing.T) {
	apiErr := FromError(errors.New("boom: pq: relation does not exist"))
	if apiErr.Status != 500 {
		t.Errorf("expected 500 for non-domain error, got %d", apiErr.Status)
	}
	if apiErr.Message != "Internal server error" {
		t.Errorf("expected fixed message without leak, got %q", apiErr.Message)
	}
}
