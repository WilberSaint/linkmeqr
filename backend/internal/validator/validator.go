package validator

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

var instance = validator.New()

// DecodeAndValidate reads a JSON body into dst and runs struct validation
// tags on it. Returns field-level errors keyed by JSON field name.
func DecodeAndValidate(r *http.Request, dst any) map[string]string {
	if err := Decode(r, dst); err != nil {
		return map[string]string{"_body": "Invalid JSON body."}
	}
	return Validate(dst)
}

// Decode reads a JSON body without validating it — for the handful of
// endpoints that accept two alternative payload shapes and only know which
// set of rules applies after looking at what arrived.
func Decode(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

// Validate runs struct validation tags on an already-decoded value,
// returning field-level errors keyed by JSON field name (nil when valid).
func Validate(dst any) map[string]string {
	if err := instance.Struct(dst); err != nil {
		if verrs, ok := err.(validator.ValidationErrors); ok {
			out := map[string]string{}
			for _, fe := range verrs {
				out[strings.ToLower(fe.Field())] = fe.Tag()
			}
			return out
		}
		return map[string]string{"_body": "Invalid payload."}
	}
	return nil
}
