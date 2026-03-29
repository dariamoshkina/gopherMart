package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dariamoshkina/gopherMart/internal/auth"
)

const testSecret = "test-secret"

func nextHandler(t *testing.T, wantUserID int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := UserIDFromContext(r.Context())
		assert.Equal(t, wantUserID, got)
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuth_ValidTokenInHeader(t *testing.T) {
	token, err := auth.IssueToken(testSecret, 42)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	Auth(testSecret)(nextHandler(t, 42)).ServeHTTP(rec, r)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuth_ValidTokenInCookie(t *testing.T) {
	token, err := auth.IssueToken(testSecret, 7)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: auth.CookieName(), Value: token})
	rec := httptest.NewRecorder()

	Auth(testSecret)(nextHandler(t, 7)).ServeHTTP(rec, r)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuth_MissingToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	Auth(testSecret)(nextHandler(t, 0)).ServeHTTP(rec, r)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_InvalidToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer not-a-real-token")
	rec := httptest.NewRecorder()

	Auth(testSecret)(nextHandler(t, 0)).ServeHTTP(rec, r)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
