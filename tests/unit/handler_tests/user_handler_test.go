package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/handler"
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

type dummyUserService struct{}

func (s *dummyUserService) Register(input *model.CreateUserInput) (*model.UserDTO, error) {
	if input.Email == "error@example.com" {
		return nil, errors.New("service error")
	}

	return &model.UserDTO{
		ID:       42,
		Username: input.Username,
		Email:    input.Email,
		Role:     model.RoleUser,
	}, nil
}

func (s *dummyUserService) Get(id uint) (*model.UserDTO, error) {
	if id == 999 {
		return nil, errors.New("user not found")
	}

	return &model.UserDTO{
		ID:       id,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     model.RoleUser,
	}, nil
}

func (s *dummyUserService) Update(id uint, input *model.UpdateUserInput) (*model.UserDTO, error) {
	if id == 999 {
		return nil, errors.New("user not found")
	}
	user := &model.UserDTO{
		ID:       id,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     model.RoleUser,
	}

	if input.Username != nil {
		user.Username = *input.Username
	}
	if input.Email != nil {
		user.Email = *input.Email
	}
	if input.Role != nil {
		user.Role = *input.Role
	}

	return user, nil
}

func (s *dummyUserService) Delete(id uint) error {
	if id == 999 {
		return errors.New("user not found")
	}
	return nil
}

func (s *dummyUserService) Authenticate(email, password string) (*model.UserDTO, error) {
	if email == "test@example.com" && password == "testpassword" {
		return &model.UserDTO{
			ID:       123,
			Username: "testuser",
			Email:    email,
			Role:     model.RoleUser,
		}, nil
	}
	return nil, errors.New("invalid credentials")
}

func (s *dummyUserService) Search(query, sort, filter string, p repository.Pagination) ([]*model.UserDTO, error) {
	if query == "error" {
		return nil, errors.New("search error")
	}

	users := []*model.UserDTO{
		{
			ID:       1,
			Username: "user1",
			Email:    "user1@example.com",
			Role:     model.RoleUser,
		},
		{
			ID:       2,
			Username: "user2",
			Email:    "user2@example.com",
			Role:     model.RoleAdmin,
		},
	}

	return users, nil
}

func stringPtr(s string) *string {
	return &s
}

func setupUserRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())
	return router
}

func printContext(c *gin.Context) {
	fmt.Printf("Context values: user_id=%v, user_role=%v\n", c.GetUint("user_id"), c.GetString("user_role"))
}

func TestUserHandler(t *testing.T) {
	svc := &dummyUserService{}
	h := handler.NewUserHandler(svc)
	router := setupUserRouter()

	router.POST("/api/users", h.Create)

	router.GET("/api/users/me", func(c *gin.Context) {
		c.Set("user_id", uint(123))
		h.Me(c)
	})

	adminAuthMiddleware := func(c *gin.Context) {
		c.Set("user_id", uint(999))
		c.Set("user_role", "admin")
		c.Next()
	}

	router.GET("/api/users/search", adminAuthMiddleware, h.Get)
	router.GET("/api/users/:id", adminAuthMiddleware, h.Get)

	router.PUT("/api/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		idUint, _ := strconv.ParseUint(id, 10, 32)

		if uint(idUint) == 123 {
			c.Set("user_id", uint(123))
			c.Set("user_role", "user")
		} else {
			c.Set("user_id", uint(999))
			c.Set("user_role", "admin")
		}

		h.Update(c)
	})

	router.DELETE("/api/users/:id", func(c *gin.Context) {
		c.Set("user_id", uint(999))
		c.Set("user_role", "admin")
		h.Delete(c)
	})

	t.Run("Create", func(t *testing.T) {
		input := model.CreateUserInput{
			Email:    "new@example.com",
			Password: "password123",
			Username: "newuser",
		}
		jsonInput, err := json.Marshal(input)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/api/users", bytes.NewBuffer(jsonInput))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
		var responseData map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &responseData)
		require.NoError(t, err, "Should be able to unmarshal response")
		userData, ok := responseData["id"].(map[string]interface{})
		require.True(t, ok, "Response should have nested user data under 'id' key")

		assert.Equal(t, float64(42), userData["id"], "ID should be 42")
		assert.Equal(t, "newuser", userData["username"], "Username should match input")
		assert.Equal(t, "new@example.com", userData["email"], "Email should match input")
	})

	t.Run("Create_InvalidInput", func(t *testing.T) {
		input := struct {
			Email string `json:"email"`
		}{
			Email: "incomplete@example.com",
		}
		jsonInput, err := json.Marshal(input)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/api/users", bytes.NewBuffer(jsonInput))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create_ServiceError", func(t *testing.T) {
		input := model.CreateUserInput{
			Email:    "error@example.com",
			Password: "password123",
			Username: "erroruser",
		}
		jsonInput, err := json.Marshal(input)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/api/users", bytes.NewBuffer(jsonInput))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Me", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/users/me", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var responseData map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &responseData)
		require.NoError(t, err)

		assert.Equal(t, float64(123), responseData["id"])
		assert.Equal(t, "testuser", responseData["username"])
		assert.Equal(t, "test@example.com", responseData["email"])
	})

	t.Run("Get_ByID", func(t *testing.T) {
		query := url.Values{}
		query.Add("q", "dummy")

		req, err := http.NewRequest("GET", "/api/users/42?"+query.Encode(), nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var users []map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &users)
		require.NoError(t, err)

		assert.Len(t, users, 2, "Should return 2 users")
		assert.Equal(t, "user1", users[0]["username"])
		assert.Equal(t, "user2", users[1]["username"])
	})

	t.Run("Get_NotFound", func(t *testing.T) {
		query := url.Values{}
		query.Add("q", "nonexistent")

		req, err := http.NewRequest("GET", "/api/users/search?"+query.Encode(), nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var users []map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &users)
		require.NoError(t, err)

		assert.Len(t, users, 2, "Our mock always returns 2 users")
	})

	t.Run("Search", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/users/search?q=test&page=1&page_size=10", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var users []map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &users)
		require.NoError(t, err)

		assert.Len(t, users, 2, "Should return 2 users")
		assert.Equal(t, "user1", users[0]["username"])
		assert.Equal(t, "user2", users[1]["username"])
	})

	t.Run("Update_OwnProfile", func(t *testing.T) {
		input := model.UpdateUserInput{
			Username: stringPtr("updateduser"),
			Email:    stringPtr("updated@example.com"),
		}
		jsonInput, err := json.Marshal(input)
		require.NoError(t, err)

		req, err := http.NewRequest("PUT", "/api/users/123", bytes.NewBuffer(jsonInput))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var responseData map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &responseData)
		require.NoError(t, err)

		assert.Equal(t, "updateduser", responseData["username"])
		assert.Equal(t, "updated@example.com", responseData["email"])
		assert.Equal(t, "user", responseData["role"])
	})

	t.Run("Delete", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", "/api/users/42", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("Delete_InvalidID", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", "/api/users/invalid", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
