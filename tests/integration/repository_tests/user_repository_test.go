package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

func TestUserRepo_CRUD_Integration(t *testing.T) {

	// Get a clean database state.
	db := utils.SetupTest(t)

	// Create the user repository.
	userRepo := repository.NewUserRepo(db)

	// Define a default pagination (Page 1, PageSize 10)
	defaultPage := repository.Pagination{Page: 1, PageSize: 10}

	// Test data.
	testUser := &model.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "securepassword",
	}

	t.Run("Create", func(t *testing.T) {
		err := userRepo.Create(testUser)
		require.NoError(t, err, "Should create user without error")
		assert.NotZero(t, testUser.ID, "User ID should be set after creation")
		assert.False(t, testUser.CreatedAt.IsZero(), "CreatedAt should be set")
		assert.False(t, testUser.UpdatedAt.IsZero(), "UpdatedAt should be set")
	})

	t.Run("FindByID", func(t *testing.T) {
		foundUser, err := userRepo.FindByID(testUser.ID)
		require.NoError(t, err, "Should find user by ID")
		assert.Equal(t, testUser.ID, foundUser.ID)
		assert.Equal(t, testUser.Username, foundUser.Username)
		assert.Equal(t, testUser.Email, foundUser.Email)
		assert.Equal(t, testUser.Password, foundUser.Password)

		_, err = userRepo.FindByID(9999)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Should return record not found for non-existent ID")
	})

	t.Run("FindByEmail", func(t *testing.T) {
		foundUser, err := userRepo.FindByEmail(testUser.Email)
		require.NoError(t, err, "Should find user by email")
		assert.Equal(t, testUser.ID, foundUser.ID)
		assert.Equal(t, testUser.Username, foundUser.Username)
		assert.Equal(t, testUser.Email, foundUser.Email)

		_, err = userRepo.FindByEmail("nonexistent@example.com")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Should return record not found for non-existent email")
	})

	t.Run("ListAll", func(t *testing.T) {
		secondUser := &model.User{
			Username: "seconduser",
			Email:    "second@example.com",
			Password: "anotherpassword",
		}
		err := userRepo.Create(secondUser)
		require.NoError(t, err, "Should create second user")

		users, err := userRepo.ListAll(defaultPage)
		require.NoError(t, err, "Should list all users")
		assert.Len(t, users, 2, "Should have 2 users")

		foundFirst := false
		foundSecond := false
		for _, u := range users {
			if u.ID == testUser.ID {
				foundFirst = true
			}
			if u.ID == secondUser.ID {
				foundSecond = true
			}
		}
		assert.True(t, foundFirst, "First user should be in the list")
		assert.True(t, foundSecond, "Second user should be in the list")
	})

	t.Run("Delete", func(t *testing.T) {
		err := userRepo.Delete(testUser.ID)
		require.NoError(t, err, "Should delete user without error")

		_, err = userRepo.FindByID(testUser.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Deleted user should not be found")

		users, err := userRepo.ListAll(defaultPage)
		require.NoError(t, err, "Should list all users")
		assert.Len(t, users, 1, "Should have 1 user after deletion")
		assert.NotEqual(t, testUser.ID, users[0].ID, "Deleted user should not be in the list")

		err = userRepo.Delete(9999)
		assert.EqualError(t, err, "user not found", "Should return error when deleting non-existent user")
	})

	utils.CleanTestData(t)
}
