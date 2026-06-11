// Package validation wraps go-playground/validator with our domain error
// envelope. Handlers should call `validation.Struct(&req)` after BodyParser
// to enforce the `validate:"..."` tags on every request DTO — otherwise the
// tags are decorative.
package validation

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/go-playground/validator/v10"
)

var (
	once     sync.Once
	instance *validator.Validate
)

// phoneRegex accepts an MSISDN as an optional leading '+' followed by 8-15
// digits. Deliberately lenient (it allows a leading 0 in local-format numbers)
// so it doesn't reject what the current frontend submits; the goal here is to
// catch the garbage that otherwise only fails later, silently, at the WhatsApp
// gateway (letters, empty strings, obviously-too-short/long input). Full E.164
// normalisation is a follow-up coordinated with the frontend.
var phoneRegex = regexp.MustCompile(`^\+?\d{8,15}$`)

func validatePhone(fl validator.FieldLevel) bool {
	return phoneRegex.MatchString(fl.Field().String())
}

// get lazily initialises the singleton validator. Initialised once on first
// use to avoid the cost (~few µs) at startup; trivially safe under concurrent
// reads because sync.Once handles the race.
func get() *validator.Validate {
	once.Do(func() {
		instance = validator.New(validator.WithRequiredStructEnabled())
		_ = instance.RegisterValidation("phone", validatePhone)
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
