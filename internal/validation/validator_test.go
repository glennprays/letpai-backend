package validation

import "testing"

func TestPhoneValidation(t *testing.T) {
	type payload struct {
		Phone string `validate:"required,phone"`
	}

	valid := []string{
		"628123456789",  // country-code format the FE sends
		"+628123456789", // with leading +
		"08123456789",   // local format with leading 0 (must still pass)
		"12345678",      // minimum length
	}
	for _, v := range valid {
		if err := Struct(&payload{Phone: v}); err != nil {
			t.Errorf("expected %q to be valid, got: %v", v, err)
		}
	}

	invalid := []string{
		"",                       // empty
		"abc",                    // letters
		"123",                    // too short
		"+",                      // no digits
		"62-812-345-678",         // punctuation
		"6281234567890123456789", // too long
	}
	for _, v := range invalid {
		if err := Struct(&payload{Phone: v}); err == nil {
			t.Errorf("expected %q to be rejected, got no error", v)
		}
	}
}
