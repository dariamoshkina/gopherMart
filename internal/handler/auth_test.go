package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"golang.org/x/crypto/bcrypt"

	"github.com/dariamoshkina/gopherMart/internal/auth"
	"github.com/dariamoshkina/gopherMart/internal/model"
	"github.com/dariamoshkina/gopherMart/internal/service"
	"github.com/dariamoshkina/gopherMart/internal/service/mocks"
)

const testAuthSecret = "test-secret-for-jwt"

func TestAuthHandler_Register_Success(t *testing.T) {
	repo := mocks.NewMockUserRepository(t)
	authSvc := service.NewAuthService(repo)
	h := NewAuthHandler(authSvc, testAuthSecret)

	created := &model.User{ID: 42, Login: "user1", PasswordHash: "hash", CreatedAt: time.Now()}
	repo.On("Create", mock.Anything, "user1", mock.AnythingOfType("string")).Return(created, nil).Once()

	body := `{"login":"user1","password":"pass123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Authorization"), "Bearer ")
	assert.NotEmpty(t, rec.Result().Cookies())
	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName() {
			cookie = c
			break
		}
	}
	require.NotNil(t, cookie)
	assert.NotEmpty(t, cookie.Value)
	repo.AssertExpectations(t)
}

func TestAuthHandler_Register_BadRequest_InvalidJSON(t *testing.T) {
	repo := mocks.NewMockUserRepository(t)
	h := NewAuthHandler(service.NewAuthService(repo), testAuthSecret)

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	repo.AssertNotCalled(t, "Create")
}

func TestAuthHandler_Register_BadRequest_EmptyLogin(t *testing.T) {
	repo := mocks.NewMockUserRepository(t)
	h := NewAuthHandler(service.NewAuthService(repo), testAuthSecret)

	body := `{"login":"","password":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	repo.AssertNotCalled(t, "Create")
}

func TestAuthHandler_Register_BadRequest_NoContentType(t *testing.T) {
	repo := mocks.NewMockUserRepository(t)
	h := NewAuthHandler(service.NewAuthService(repo), testAuthSecret)

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(`{"login":"u","password":"p"}`))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Register_Conflict_LoginTaken(t *testing.T) {
	repo := mocks.NewMockUserRepository(t)
	authSvc := service.NewAuthService(repo)
	h := NewAuthHandler(authSvc, testAuthSecret)

	repo.On("Create", mock.Anything, "taken", mock.AnythingOfType("string")).Return(nil, service.ErrLoginTaken).Once()

	body := `{"login":"taken","password":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	repo.AssertExpectations(t)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	repo := mocks.NewMockUserRepository(t)
	authSvc := service.NewAuthService(repo)
	h := NewAuthHandler(authSvc, testAuthSecret)

	hashed, err := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.DefaultCost)
	require.NoError(t, err)
	stored := &model.User{ID: 1, Login: "user1", PasswordHash: string(hashed), CreatedAt: time.Now()}
	repo.On("GetByLogin", mock.Anything, "user1").Return(stored, nil).Once()

	body := `{"login":"user1","password":"pass123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Authorization"), "Bearer ")
	assert.NotEmpty(t, rec.Result().Cookies())
	repo.AssertExpectations(t)
}

func TestAuthHandler_Login_Unauthorized_WrongPassword(t *testing.T) {
	repo := mocks.NewMockUserRepository(t)
	authSvc := service.NewAuthService(repo)
	h := NewAuthHandler(authSvc, testAuthSecret)

	stored := &model.User{ID: 1, Login: "user1", PasswordHash: "wrong-hash", CreatedAt: time.Now()}
	repo.On("GetByLogin", mock.Anything, "user1").Return(stored, nil).Once()

	body := `{"login":"user1","password":"pass123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	repo.AssertExpectations(t)
}

func TestAuthHandler_Login_BadRequest_EmptyPassword(t *testing.T) {
	repo := mocks.NewMockUserRepository(t)
	h := NewAuthHandler(service.NewAuthService(repo), testAuthSecret)

	body := `{"login":"user1","password":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	repo.AssertNotCalled(t, "GetByLogin")
}

func TestAuthHandler_Login_MethodNotAllowed(t *testing.T) {
	h := NewAuthHandler(service.NewAuthService(mocks.NewMockUserRepository(t)), testAuthSecret)

	req := httptest.NewRequest(http.MethodGet, "/api/user/login", nil)
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}
