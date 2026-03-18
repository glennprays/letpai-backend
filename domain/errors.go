package domain

import "errors"

// Service error types
var (
	ErrBadRequest      = errors.New("bad request")
	ErrNotFound        = errors.New("not found")
	ErrInternalFailure = errors.New("internal failure")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrConflict        = errors.New("conflict")
)

// Error wraps service and application errors
type Error struct {
	serviceErr error
	appErr     error
}

// NewError creates a new domain error
func NewError(serviceErr, appErr error) Error {
	return Error{
		serviceErr: serviceErr,
		appErr:     appErr,
	}
}

func (e Error) Error() string {
	if e.appErr != nil {
		return e.appErr.Error()
	}
	if e.serviceErr != nil {
		return e.serviceErr.Error()
	}
	return "unknown error"
}

func (e Error) ServiceError() error {
	return e.serviceErr
}

func (e Error) AppError() error {
	return e.appErr
}
