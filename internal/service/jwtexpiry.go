package service

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

// ParseJWTExpiry extracts the `exp` claim from a JWT without verifying
// the signature. Verification happens on the gateway side on every
// request; we only need the timestamp to display "token expires <date>"
// to the admin. Returns ok=false for anything that isn't a shaped JWT
// with a numeric exp.
func ParseJWTExpiry(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Some minters pad; try the padded alphabet too.
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return time.Time{}, false
		}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp <= 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0), true
}
