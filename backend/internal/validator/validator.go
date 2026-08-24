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
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return map[string]string{"_body": "Invalid JSON body."}
	}

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
