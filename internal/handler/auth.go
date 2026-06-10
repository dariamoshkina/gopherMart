package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/dariamoshkina/gopherMart/internal/auth"
	"github.com/dariamoshkina/gopherMart/internal/model"
	"github.com/dariamoshkina/gopherMart/internal/service"
)

type AuthService interface {
	Register(ctx context.Context, login, password string) (*model.User, error)
	Login(ctx context.Context, login, password string) (*model.User, error)
}

type AuthHandler struct {
	authSvc    AuthService
	authSecret string
}

func NewAuthHandler(authSvc AuthService, authSecret string) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, authSecret: authSecret}
}

type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	login, password, ok := h.parseLoginPasswordRequest(w, r)
	if !ok {
		return
	}

	user, err := h.authSvc.Register(r.Context(), login, password)
	if err != nil {
		if errors.Is(err, service.ErrLoginTaken) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err = h.setAuthToken(w, user.ID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	login, password, ok := h.parseLoginPasswordRequest(w, r)
	if !ok {
		return
	}

	user, err := h.authSvc.Login(r.Context(), login, password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err = h.setAuthToken(w, user.ID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) parseLoginPasswordRequest(w http.ResponseWriter, r *http.Request) (login, password string, ok bool) {
	if !strings.HasPrefix(strings.TrimSpace(r.Header.Get("Content-Type")), "application/json") {
		w.WriteHeader(http.StatusBadRequest)
		return "", "", false
	}
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return "", "", false
	}
	login = strings.TrimSpace(req.Login)
	password = strings.TrimSpace(req.Password)
	if login == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return "", "", false
	}
	return login, password, true
}

func (h *AuthHandler) setAuthToken(w http.ResponseWriter, userID int64) error {
	token, err := auth.IssueToken(h.authSecret, userID)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName(),
		Value:    token,
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Authorization", "Bearer "+token)
	return nil
}
