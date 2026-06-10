package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/dariamoshkina/gopherMart/internal/auth"
)

type contextKey int

const userIDKey contextKey = iota

func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := tokenFromHeader(r)
			if token == "" {
				token = tokenFromCookie(r)
			}
			if token == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			userID, err := auth.ParseToken(secret, token)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDKey).(int64)
	return id
}

// ContextWithUserID is used in tests to inject a user ID without going through the auth middleware.
func ContextWithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func tokenFromHeader(r *http.Request) string {
	headerVal := r.Header.Get("Authorization")
	if strings.HasPrefix(headerVal, "Bearer ") {
		return strings.TrimPrefix(headerVal, "Bearer ")
	}
	return ""
}

func tokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(auth.CookieName())
	if err != nil {
		return ""
	}
	return cookie.Value
}
