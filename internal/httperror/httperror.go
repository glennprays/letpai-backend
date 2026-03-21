package httperror

import (
	"errors"

	"github.com/glennprays/letpai-backend/domain"
)

type APIError struct {
	Status  int
	Message string
}

// Response returns the error response in the format expected by the API
func (e APIError) Response() map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"message": e.Message,
		},
	}
}

func FromError(err error) APIError {
	var apiError APIError
	var domainError domain.Error

	if errors.As(err, &domainError) {
		apiError.Message = domainError.AppError().Error()
		svcErr := domainError.ServiceError()
		switch svcErr {
		case domain.ErrBadRequest:
			apiError.Status = 400
		case domain.ErrInternalFailure:
			apiError.Status = 500
		case domain.ErrNotFound:
			apiError.Status = 404
		case domain.ErrUnauthorized:
			apiError.Status = 401
		case domain.ErrForbidden:
			apiError.Status = 403
		case domain.ErrConflict:
			apiError.Status = 409
		default:
			apiError.Status = 500
			apiError.Message = "Internal server error"
		}
	} else {
		apiError.Status = 500
		apiError.Message = err.Error()
	}

	return apiError
}

// ErrBadRequest creates a bad request error
func ErrBadRequest(msg string) error {
	return domain.NewError(domain.ErrBadRequest, errors.New(msg))
}

// ErrNotFound creates a not found error
func ErrNotFound(msg string) error {
	return domain.NewError(domain.ErrNotFound, errors.New(msg))
}

// ErrUnauthorized creates an unauthorized error
func ErrUnauthorized(msg string) error {
	return domain.NewError(domain.ErrUnauthorized, errors.New(msg))
}

// ErrForbidden creates a forbidden error
func ErrForbidden(msg string) error {
	return domain.NewError(domain.ErrForbidden, errors.New(msg))
}

