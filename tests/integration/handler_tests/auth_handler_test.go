package handler_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/urlinsight-backend/internal/handler"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/internal/service"
	"github.com/fuzumoe/urlinsight-backend/tests/integration"
)

func TestAuthHandlerIntegration(t *testing.T) {
	// Set up Gin in test mode.
	gin.SetMode(gin.TestMode)
	// Setup the integration test database.
	db := integration.SetupTest(t)
	defer integration.CleanTestData(t)

	// Create real repositories.
	userRepo := repository.NewUserRepo(db)
	tokenRepo := repository.NewTokenRepo(db)
	// Create real services.
	authSvc := service.NewAuthService(userRepo, tokenRepo, "test-secret", time.Hour)
	userSvc := service.NewUserService(userRepo)
	// Create the auth handler.
	authHandler := handler.NewAuthHandler(authSvc, userSvc)

	// Set up the Gin router with auth endpoints.
	router := gin.New()
	router.POST("/login/basic", authHandler.LoginBasic)
	router.POST("/login/jwt", authHandler.LoginJWT)
	router.POST("/register", authHandler.Register)
	router.POST("/logout", authHandler.Logout)

	// Test  registration endpoint.
	t.Run("Register", func(t *testing.T) {
		regPayload := map[string]string{
			"email":    "newuser@example.com",
			"password": "password123",
			"username": "newuser",
		}
		regBytes, _ := json.Marshal(regPayload)
		reqReg := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(regBytes))
		reqReg.Header.Set("Content-Type", "application/json")
		wReg := httptest.NewRecorder()

		router.ServeHTTP(wReg, reqReg)

		assert.Equal(t, http.StatusCreated, wReg.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(wReg.Body.Bytes(), &resp)
		assert.NoError(t, err)

		// Check for token
		token, ok := resp["token"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, token)

		// Check user data
		user, ok := resp["user"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "newuser@example.com", user["email"])
		assert.Equal(t, "newuser", user["username"])
	})

	// Register a user for subsequent login basic tests
	t.Run("Register and Login Basic", func(t *testing.T) {
		// 1. Register a new user
		regPayload := map[string]string{
			"email":    "basicuser@example.com",
			"password": "basicpass",
			"username": "basicuser",
		}
		regBytes, _ := json.Marshal(regPayload)
		reqReg := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(regBytes))
		reqReg.Header.Set("Content-Type", "application/json")
		wReg := httptest.NewRecorder()

		router.ServeHTTP(wReg, reqReg)
		assert.Equal(t, http.StatusCreated, wReg.Code)

		// 2. Login with the registered user using Basic auth
		creds := "basicuser@example.com:basicpass"
		encodedCreds := base64.StdEncoding.EncodeToString([]byte(creds))
		reqLogin := httptest.NewRequest(http.MethodPost, "/login/basic", nil)
		reqLogin.Header.Set("Authorization", "Basic "+encodedCreds)
		wLogin := httptest.NewRecorder()

		router.ServeHTTP(wLogin, reqLogin)
		assert.Equal(t, http.StatusOK, wLogin.Code)

		var loginResp map[string]interface{}
		err := json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
		assert.NoError(t, err)
		token, ok := loginResp["token"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, token)
	})

	// Test for Register and Login JWT flow
	t.Run("Register and Login JWT", func(t *testing.T) {
		// 1. Register a new user
		regPayload := map[string]string{
			"email":    "integration@example.com",
			"password": "integrate",
			"username": "integrationuser",
		}
		regBytes, _ := json.Marshal(regPayload)
		reqReg := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(regBytes))
		reqReg.Header.Set("Content-Type", "application/json")
		wReg := httptest.NewRecorder()

		router.ServeHTTP(wReg, reqReg)
		assert.Equal(t, http.StatusCreated, wReg.Code)

		// 2. Login with the registered user
		loginPayload := map[string]string{
			"email":    "integration@example.com",
			"password": "integrate",
		}
		loginBytes, _ := json.Marshal(loginPayload)
		reqLogin := httptest.NewRequest(http.MethodPost, "/login/jwt", bytes.NewBuffer(loginBytes))
		reqLogin.Header.Set("Content-Type", "application/json")
		wLogin := httptest.NewRecorder()

		router.ServeHTTP(wLogin, reqLogin)
		assert.Equal(t, http.StatusOK, wLogin.Code)

		var loginResp map[string]interface{}
		err := json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
		assert.NoError(t, err)
		token, ok := loginResp["token"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, token)
	})

	t.Run("Login Basic", func(t *testing.T) {
		// For LoginBasic, the client must send a Basic auth header.
		creds := "integration@example.com:integrate"
		encodedCreds := base64.StdEncoding.EncodeToString([]byte(creds))
		req := httptest.NewRequest(http.MethodPost, "/login/basic", nil)
		req.Header.Set("Authorization", "Basic "+encodedCreds)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		token, ok := resp["token"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, token)
	})

	t.Run("Login JWT", func(t *testing.T) {
		// Test the /login/jwt endpoint using the registered user.
		loginPayload := map[string]string{
			"email":    "integration@example.com",
			"password": "integrate",
		}
		loginBytes, _ := json.Marshal(loginPayload)
		req := httptest.NewRequest(http.MethodPost, "/login/jwt", bytes.NewBuffer(loginBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		token, ok := resp["token"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, token)
	})

	t.Run("Logout", func(t *testing.T) {
		// Log in the user to obtain a token using /login/jwt.
		loginPayload := map[string]string{
			"email":    "integration@example.com",
			"password": "integrate",
		}
		loginBytes, _ := json.Marshal(loginPayload)
		reqLogin := httptest.NewRequest(http.MethodPost, "/login/jwt", bytes.NewBuffer(loginBytes))
		reqLogin.Header.Set("Content-Type", "application/json")
		wLogin := httptest.NewRecorder()
		router.ServeHTTP(wLogin, reqLogin)
		assert.Equal(t, http.StatusOK, wLogin.Code)

		var loginResp map[string]interface{}
		err := json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
		assert.NoError(t, err)
		token, ok := loginResp["token"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, token)

		// Test the /logout endpoint.
		reqLogout := httptest.NewRequest(http.MethodPost, "/logout", nil)
		reqLogout.Header.Set("Authorization", "Bearer "+token)
		wLogout := httptest.NewRecorder()
		router.ServeHTTP(wLogout, reqLogout)
		assert.Equal(t, http.StatusOK, wLogout.Code)

		var logoutResp map[string]interface{}
		err = json.Unmarshal(wLogout.Body.Bytes(), &logoutResp)
		assert.NoError(t, err)
		assert.Equal(t, "logged out", logoutResp["message"])
	})
}
