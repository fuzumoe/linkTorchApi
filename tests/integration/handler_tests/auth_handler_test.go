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
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/handler"
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/internal/service"
	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

func TestAuthHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := utils.SetupTest(t)
	defer utils.CleanTestData(t)

	userRepo := repository.NewUserRepo(db)
	tokenRepo := repository.NewTokenRepo(db)
	authSvc := service.NewAuthService(userRepo, tokenRepo, "test-secret", time.Hour)
	userSvc := service.NewUserService(userRepo)

	authHandler := handler.NewAuthHandler(authSvc, userSvc)

	router := gin.New()
	router.POST("/login/basic", authHandler.LoginBasic)
	router.POST("/login/jwt", authHandler.LoginJWT)
	router.POST("/logout", authHandler.Logout)

	testUser := &model.CreateUserInput{
		Email:    "testuser@example.com",
		Password: "testpassword",
		Username: "testuser",
	}
	userDTO, err := userSvc.Register(testUser)
	require.NoError(t, err)
	require.NotNil(t, userDTO)

	t.Run("LoginBasic", func(t *testing.T) {

		creds := "testuser@example.com:testpassword"
		encodedCreds := base64.StdEncoding.EncodeToString([]byte(creds))
		req := httptest.NewRequest(http.MethodPost, "/login/basic", nil)
		req.Header.Set("Authorization", "Basic "+encodedCreds)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		token, ok := resp["token"].(string)
		assert.True(t, ok, "Response should contain token")
		assert.NotEmpty(t, token, "Token should not be empty")
	})

	t.Run("LoginJWT", func(t *testing.T) {

		loginPayload := handler.LoginRequest{
			Email:    "testuser@example.com",
			Password: "testpassword",
		}
		loginBytes, err := json.Marshal(loginPayload)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/login/jwt", bytes.NewBuffer(loginBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		token, ok := resp["token"].(string)
		assert.True(t, ok, "Response should contain token")
		assert.NotEmpty(t, token, "Token should not be empty")
	})

	t.Run("Logout", func(t *testing.T) {

		loginPayload := handler.LoginRequest{
			Email:    "testuser@example.com",
			Password: "testpassword",
		}
		loginBytes, err := json.Marshal(loginPayload)
		require.NoError(t, err)

		reqLogin := httptest.NewRequest(http.MethodPost, "/login/jwt", bytes.NewBuffer(loginBytes))
		reqLogin.Header.Set("Content-Type", "application/json")
		wLogin := httptest.NewRecorder()

		router.ServeHTTP(wLogin, reqLogin)
		assert.Equal(t, http.StatusOK, wLogin.Code)

		var loginResp map[string]interface{}
		err = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
		require.NoError(t, err)

		token, ok := loginResp["token"].(string)
		assert.True(t, ok, "Response should contain token")
		assert.NotEmpty(t, token, "Token should not be empty")

		reqLogout := httptest.NewRequest(http.MethodPost, "/logout", nil)
		reqLogout.Header.Set("Authorization", "Bearer "+token)
		wLogout := httptest.NewRecorder()

		router.ServeHTTP(wLogout, reqLogout)
		assert.Equal(t, http.StatusOK, wLogout.Code)

		var logoutResp map[string]interface{}
		err = json.Unmarshal(wLogout.Body.Bytes(), &logoutResp)
		require.NoError(t, err)

		assert.Equal(t, "logged out", logoutResp["message"], "Logout response should contain success message")
	})

	t.Run("LoginBasic_Invalid", func(t *testing.T) {

		creds := "testuser@example.com:wrongpassword"
		encodedCreds := base64.StdEncoding.EncodeToString([]byte(creds))
		req := httptest.NewRequest(http.MethodPost, "/login/basic", nil)
		req.Header.Set("Authorization", "Basic "+encodedCreds)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("LoginJWT_Invalid", func(t *testing.T) {

		loginPayload := handler.LoginRequest{
			Email:    "testuser@example.com",
			Password: "wrongpassword",
		}
		loginBytes, err := json.Marshal(loginPayload)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/login/jwt", bytes.NewBuffer(loginBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Logout_NoToken", func(t *testing.T) {

		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
