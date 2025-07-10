package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

func TestUser(t *testing.T) {
	t.Run("To DTO", func(t *testing.T) {
		createdAt := time.Date(2025, 7, 9, 12, 0, 0, 0, time.UTC)
		updatedAt := createdAt.Add(time.Hour)
		user := &model.User{
			ID:        1,
			Username:  "testuser",
			Email:     "test@example.com",
			Password:  "secret",
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}

		dto := user.ToDTO()

		assert.Equal(t, user.ID, dto.ID, "ID should match")
		assert.Equal(t, user.Username, dto.Username, "Username should match")
		assert.Equal(t, user.Email, dto.Email, "Email should match")
		assert.WithinDuration(t, user.CreatedAt, dto.CreatedAt, time.Second, "CreatedAt should match")
		assert.WithinDuration(t, user.UpdatedAt, dto.UpdatedAt, time.Second, "UpdatedAt should match")
	})

	t.Run("From Create Input", func(t *testing.T) {
		input := &model.CreateUserInput{
			Username: "newuser",
			Email:    "new@example.com",
			Password: "password123",
		}

		user := model.UserFromCreateInput(input)

		assert.NotNil(t, user, "User should not be nil")
		assert.Equal(t, input.Username, user.Username, "Username should match")
		assert.Equal(t, input.Email, user.Email, "Email should match")
		assert.Equal(t, input.Password, user.Password, "Password should match")
		assert.NotZero(t, user.CreatedAt, "CreatedAt should be set")
		assert.NotZero(t, user.UpdatedAt, "UpdatedAt should be set")
	})

	t.Run("Table Name", func(t *testing.T) {
		expected := "users"
		user := model.User{}

		assert.Equal(t, expected, user.TableName(), "TableName should return 'users'")
	})
}
