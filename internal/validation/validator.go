// Package validation wraps go-playground/validator with our domain error
// envelope. Handlers should call `validation.Struct(&req)` after BodyParser
// to enforce the `validate:"..."` tags on every request DTO — otherwise the
// tags are decorative.
package validation

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/go-playground/validator/v10"
)

var (
	once     sync.Once
	instance *validator.Validate
)

// get lazily initialises the singleton validator. Initialised once on first
// use to avoid the cost (~few µs) at startup; trivially safe under concurrent
// reads because sync.Once handles the race.
func get() *validator.Validate {
	once.Do(func() {
		instance = validator.New(validator.WithRequiredStructEnabled())
	})
	return instance
}

// Struct validates `req` against its `validate:"..."` tags. On failure
// returns a domain.ErrBadRequest wrapping a flat human-readable summary
// of every failing field. Pointer receivers are required so that the
// validator's reflection finds the tags on the struct value, not on a copy.
func Struct(req interface{}) error {
	if err := get().Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			msgs := make([]string, 0, len(ve))
			for _, fe := range ve {
				msgs = append(msgs, fmt.Sprintf("%s: failed %q", fe.Field(), fe.Tag()))
			}
			return domain.NewError(domain.ErrBadRequest, errors.New(strings.Join(msgs, "; ")))
		}
		return domain.NewError(domain.ErrBadRequest, err)
	}
	return nil
}
