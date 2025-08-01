package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/internal/service"
	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

func TestUserService_Integration(t *testing.T) {

	db := utils.SetupTest(t)

	userRepo := repository.NewUserRepo(db)
	userService := service.NewUserService(userRepo)

	testUsername := "testuser"
	testEmail := "test@example.com"
	testPassword := "password123"

	secondTestUsername := "seconduser"
	secondTestEmail := "second@example.com"

	t.Run("Register", func(t *testing.T) {
		input := &model.CreateUserInput{
			Username: testUsername,
			Email:    testEmail,
			Password: testPassword,
		}

		user, err := userService.Register(input)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, testUsername, user.Username)
		assert.Equal(t, testEmail, user.Email)
		assert.NotZero(t, user.ID)

		dbUser, err := userRepo.FindByEmail(testEmail)
		require.NoError(t, err)
		assert.Equal(t, testUsername, dbUser.Username)

		err = bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(testPassword))
		assert.NoError(t, err)
	})

	t.Run("Register_DuplicateEmail", func(t *testing.T) {
		input := &model.CreateUserInput{
			Username: "anothername",
			Email:    testEmail,
			Password: "anotherpassword",
		}

		user, err := userService.Register(input)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "email already in use")
	})

	t.Run("Authenticate_Success", func(t *testing.T) {
		user, err := userService.Authenticate(testEmail, testPassword)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, testUsername, user.Username)
		assert.Equal(t, testEmail, user.Email)
	})

	t.Run("Authenticate_WrongPassword", func(t *testing.T) {
		user, err := userService.Authenticate(testEmail, "wrongpassword")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("Authenticate_NonExistentUser", func(t *testing.T) {
		user, err := userService.Authenticate("nonexistent@example.com", testPassword)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("Get", func(t *testing.T) {

		dbUser, err := userRepo.FindByEmail(testEmail)
		require.NoError(t, err)

		user, err := userService.Get(dbUser.ID)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, testUsername, user.Username)
		assert.Equal(t, testEmail, user.Email)
	})

	t.Run("Get_NonExistent", func(t *testing.T) {
		user, err := userService.Get(9999)
		assert.Error(t, err)
		assert.Nil(t, user)
	})

	t.Run("Search Empty", func(t *testing.T) {

		pagination := repository.Pagination{Page: 1, PageSize: 10}
		users, err := userService.Search("", "", "", pagination)
		require.NoError(t, err)
		assert.NotEmpty(t, users)
	})

	t.Run("Search Multiple", func(t *testing.T) {

		input := &model.CreateUserInput{
			Username: secondTestUsername,
			Email:    secondTestEmail,
			Password: "anotherpass",
		}
		_, err := userService.Register(input)
		require.NoError(t, err)

		pagination := repository.Pagination{Page: 1, PageSize: 10}

		t.Run("By Username", func(t *testing.T) {
			users, err := userService.Search("", "", "user", pagination)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(users), 2, "Should find at least 2 users with 'user' in username")

			foundFirst := false
			foundSecond := false
			for _, u := range users {
				if u.Email == testEmail {
					foundFirst = true
				}
				if u.Email == secondTestEmail {
					foundSecond = true
				}
			}
			assert.True(t, foundFirst, "First test user should be in the list")
			assert.True(t, foundSecond, "Second test user should be in the list")
		})

		t.Run("By Email", func(t *testing.T) {
			users, err := userService.Search(testEmail, "", "", pagination)
			require.NoError(t, err)
			assert.Equal(t, 1, len(users), "Should find exactly 1 user with this email")
			assert.Equal(t, testEmail, users[0].Email)
		})
		t.Run("By Partial Email", func(t *testing.T) {
			users, err := userService.Search("example.com", "", "", pagination)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(users), 2, "Should find at least 2 users with 'example.com' in email")
		})

		t.Run("All Users", func(t *testing.T) {
			users, err := userService.Search("", "", "", pagination)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(users), 2, "Should find at least 2 users total")
		})
	})

	t.Run("Delete", func(t *testing.T) {

		dbUser, err := userRepo.FindByEmail(secondTestEmail)
		require.NoError(t, err)

		err = userService.Delete(dbUser.ID)
		require.NoError(t, err)

		_, err = userService.Get(dbUser.ID)
		assert.Error(t, err)

		firstUser, err := userService.Get(1)
		assert.NoError(t, err)
		assert.NotNil(t, firstUser)
	})

	utils.CleanTestData(t)
}
