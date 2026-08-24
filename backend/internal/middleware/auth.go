package middleware

import (
	"context"
	"net/http"
	"strings"

	"linkmeqr/backend/internal/utils"
)

type contextKey string

const (
	ctxUserID contextKey = "userID"
	ctxRole   contextKey = "role"
)

func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				utils.Error(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid Authorization header.")
				return
			}

			tokenString := strings.TrimPrefix(header, "Bearer ")
			claims, err := utils.ParseAccessToken(jwtSecret, tokenString)
			if err != nil {
				utils.Error(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token.")
				return
			}

			ctx := context.WithValue(r.Context(), ctxUserID, claims.UserID)
			ctx = context.WithValue(ctx, ctxRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			currentRole, _ := r.Context().Value(ctxRole).(string)
			if currentRole != role {
				utils.Error(w, http.StatusForbidden, "forbidden", "You do not have access to this resource.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func UserIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxUserID).(string)
	return id
}

func RoleFromContext(ctx context.Context) string {
	role, _ := ctx.Value(ctxRole).(string)
	return role
}
