package httperror

import (
	"errors"

	"github.com/glennprays/letpai-backend/domain"
)

type APIError struct {
	Status  int
	Code    string
	Message string
}

// CodeForStatus maps an HTTP status to the stable machine-readable error code
// the frontend branches on. Keeping this in one place is what lets every error
// path — handler-level and the global ErrorHandler — emit the same envelope.
func CodeForStatus(status int) string {
	switch status {
	case 400:
		return "BAD_REQUEST"
	case 401:
		return "UNAUTHORIZED"
	case 403:
		return "FORBIDDEN"
	case 404:
		return "NOT_FOUND"
	case 409:
		return "CONFLICT"
	case 429:
		return "RATE_LIMITED"
	default:
		return "INTERNAL"
	}
}

// Response returns the unified error envelope used across the whole API:
// { "success": false, "error": { "code": ..., "message": ... } }.
func (e APIError) Response() map[string]interface{} {
	code := e.Code
	if code == "" {
		code = CodeForStatus(e.Status)
	}
	return map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    code,
			"message": e.Message,
		},
	}
}

func FromError(err error) APIError {
	var apiError APIError
	var domainError domain.Error

	if errors.As(err, &domainError) {
		// Several repo paths return NewError(svc, nil) — notably NotFound
		// rows — so AppError() can be nil. Dereferencing it used to crash
		// the handler. Fall back to the domain's own Error() message,
		// which already handles the appErr-is-nil branch.
		if appErr := domainError.AppError(); appErr != nil {
			apiError.Message = appErr.Error()
		} else {
			apiError.Message = domainError.Error()
		}
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
		// Non-domain errors are unexpected. Return a generic message to
		// avoid leaking internals (SQL errors, file paths, wrapping context
		// from fmt.Errorf chains). The original err should still be logged
		// server-side with a trace ID for debugging — see middleware.ErrorHandler.
		apiError.Status = 500
		apiError.Message = "Internal server error"
	}

	apiError.Code = CodeForStatus(apiError.Status)
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
