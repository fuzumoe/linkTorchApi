package model_test

import (
	"testing"
	"time"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

// TestUserToDTO tests the conversion of User model to UserDTO.
func TestUserToDTO(t *testing.T) {
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

	if dto.ID != user.ID {
		t.Errorf("ToDTO ID = %d; want %d", dto.ID, user.ID)
	}
	if dto.Username != user.Username {
		t.Errorf("ToDTO Username = %s; want %s", dto.Username, user.Username)
	}
	if dto.Email != user.Email {
		t.Errorf("ToDTO Email = %s; want %s", dto.Email, user.Email)
	}
	if !dto.CreatedAt.Equal(user.CreatedAt) {
		t.Errorf("ToDTO CreatedAt = %v; want %v", dto.CreatedAt, user.CreatedAt)
	}
	if !dto.UpdatedAt.Equal(user.UpdatedAt) {
		t.Errorf("ToDTO UpdatedAt = %v; want %v", dto.UpdatedAt, user.UpdatedAt)
	}
}

// TestFromCreateInput tests the conversion from CreateUserInput to User model.
func TestUserFromCreateInput(t *testing.T) {
	input := &model.CreateUserInput{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "password123",
	}

	// Update this line to use the renamed function
	user := model.UserFromCreateInput(input)

	if user.Username != input.Username {
		t.Errorf("UserFromCreateInput Username = %s; want %s", user.Username, input.Username)
	}
	if user.Email != input.Email {
		t.Errorf("UserFromCreateInput Email = %s; want %s", user.Email, input.Email)
	}
	if user.Password != input.Password {
		t.Errorf("UserFromCreateInput Password = %s; want %s", user.Password, input.Password)
	}
	if user.ID != 0 {
		t.Errorf("UserFromCreateInput ID = %d; want 0", user.ID)
	}
	// CreatedAt and UpdatedAt will be zero values
	if !user.CreatedAt.IsZero() {
		t.Errorf("UserFromCreateInput CreatedAt = %v; want zero value", user.CreatedAt)
	}
	if !user.UpdatedAt.IsZero() {
		t.Errorf("UserFromCreateInput UpdatedAt = %v; want zero value", user.UpdatedAt)
	}
}

// TestUserTableName checks the table name for User model.
func TestUserTableName(t *testing.T) {
	expected := "users"
	user := model.User{}                        // Create the User struct first
	if tn := user.TableName(); tn != expected { // Then call the method on it
		t.Errorf("TableName = %s; want %s", tn, expected)
	}
}
