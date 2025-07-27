package repository_test

import (
	"fmt"
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

	// admin user for testing
	adminUser := &model.User{
		Username: "adminuser",
		Role:     model.RoleAdmin,
		Email:    "admin@example.com",
		Password: "securepassword",
	}

	// crawler user for testing
	crawlerUser := &model.User{
		Username: "crawleruser",
		Role:     model.RoleCrawler,
		Email:    "crawler@example.com",
		Password: "securepassword",
	}

	//worker user for testing
	workerUser := &model.User{
		Username: "workeruser",
		Role:     model.RoleWorker,
		Email:    "worker@example.com",
		Password: "securepassword",
	}

	t.Run("Create Normal User", func(t *testing.T) {
		err := userRepo.Create(testUser)
		require.NoError(t, err, "Should create user without error")
		assert.NotZero(t, testUser.ID, "User ID should be set after creation")
		assert.False(t, testUser.CreatedAt.IsZero(), "CreatedAt should be set")
		assert.False(t, testUser.UpdatedAt.IsZero(), "UpdatedAt should be set")
	})

	t.Run("Create Admin User", func(t *testing.T) {
		err := userRepo.Create(adminUser)
		require.NoError(t, err, "Should create admin user without error")
		assert.NotZero(t, adminUser.ID, "Admin User ID should be set after creation")
		assert.Equal(t, model.RoleAdmin, adminUser.Role, "Admin user role should be set to 'admin'")
	})

	t.Run("Create Crawler User", func(t *testing.T) {
		err := userRepo.Create(crawlerUser)
		require.NoError(t, err, "Should create crawler user without error")
		assert.NotZero(t, crawlerUser.ID, "Crawler User ID should be set after creation")
		assert.Equal(t, model.RoleCrawler, crawlerUser.Role, "Crawler user role should be set to 'crawler'")
	})

	t.Run("Create Worker User", func(t *testing.T) {
		err := userRepo.Create(workerUser)
		require.NoError(t, err, "Should create worker user without error")
		assert.NotZero(t, workerUser.ID, "Worker User ID should be set after creation")
		assert.Equal(t, model.RoleWorker, workerUser.Role, "Worker user role should be set to 'worker'")
	})

	t.Run("Find By ID", func(t *testing.T) {
		foundUser, err := userRepo.FindByID(testUser.ID)
		require.NoError(t, err, "Should find user by ID")
		assert.Equal(t, testUser.ID, foundUser.ID)
		assert.Equal(t, testUser.Username, foundUser.Username)
		assert.Equal(t, testUser.Email, foundUser.Email)
		assert.Equal(t, testUser.Password, foundUser.Password)

		_, err = userRepo.FindByID(9999)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Should return record not found for non-existent ID")
	})

	t.Run("Find By Email", func(t *testing.T) {
		foundUser, err := userRepo.FindByEmail(testUser.Email)
		require.NoError(t, err, "Should find user by email")
		assert.Equal(t, testUser.ID, foundUser.ID)
		assert.Equal(t, testUser.Username, foundUser.Username)
		assert.Equal(t, testUser.Email, foundUser.Email)

		_, err = userRepo.FindByEmail("nonexistent@example.com")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Should return record not found for non-existent email")
	})

	t.Run("Search Users", func(t *testing.T) {
		// Search by email
		t.Run("Search by Email", func(t *testing.T) {
			users, err := userRepo.Search("example.com", "", "", defaultPage)
			require.NoError(t, err, "Should search users by email")
			assert.GreaterOrEqual(t, len(users), 4, "Should find all users with 'example.com' in email")

			users, err = userRepo.Search("admin", "", "", defaultPage)
			require.NoError(t, err, "Should search users by email")
			assert.Len(t, users, 1, "Should find only admin user")
			assert.Equal(t, adminUser.Email, users[0].Email)
		})

		// Search by role
		t.Run("Search by Role", func(t *testing.T) {
			users, err := userRepo.Search("", string(model.RoleAdmin), "", defaultPage)
			require.NoError(t, err, "Should search users by role")
			assert.Len(t, users, 1, "Should find only admin user")
			assert.Equal(t, model.RoleAdmin, users[0].Role)

			users, err = userRepo.Search("", string(model.RoleCrawler), "", defaultPage)
			require.NoError(t, err, "Should search users by role")
			assert.Len(t, users, 1, "Should find only crawler user")
			assert.Equal(t, model.RoleCrawler, users[0].Role)
		})

		// Search by username
		t.Run("Search by Username", func(t *testing.T) {
			users, err := userRepo.Search("", "", "user", defaultPage)
			require.NoError(t, err, "Should search users by username")
			assert.GreaterOrEqual(t, len(users), 4, "Should find all users with 'user' in username")

			users, err = userRepo.Search("", "", "admin", defaultPage)
			require.NoError(t, err, "Should search users by username")
			assert.Len(t, users, 1, "Should find only admin user")
			assert.Equal(t, adminUser.Username, users[0].Username)
		})

		// Combined search
		t.Run("Combined Search", func(t *testing.T) {
			users, err := userRepo.Search("admin", string(model.RoleAdmin), "", defaultPage)
			require.NoError(t, err, "Should perform combined search")
			assert.Len(t, users, 1, "Should find only admin user")
			assert.Equal(t, adminUser.Email, users[0].Email)
			assert.Equal(t, model.RoleAdmin, users[0].Role)

			users, err = userRepo.Search("example.com", "", "worker", defaultPage)
			require.NoError(t, err, "Should perform combined search")
			assert.Len(t, users, 1, "Should find only worker user")
			assert.Equal(t, workerUser.Username, users[0].Username)
		})

		// No results
		t.Run("No Results", func(t *testing.T) {
			users, err := userRepo.Search("nonexistent", "", "", defaultPage)
			require.NoError(t, err, "Should handle search with no results")
			assert.Len(t, users, 0, "Should return empty slice for no matches")

			users, err = userRepo.Search("", "invalid_role", "", defaultPage)
			require.NoError(t, err, "Should handle search with no results")
			assert.Len(t, users, 0, "Should return empty slice for no matches")
		})

		// Pagination
		t.Run("Pagination", func(t *testing.T) {
			// Add more users to test pagination
			for i := 0; i < 10; i++ {
				extraUser := &model.User{
					Username: fmt.Sprintf("extra_user_%d", i),
					Email:    fmt.Sprintf("extra%d@example.com", i),
					Password: "password",
				}
				err := userRepo.Create(extraUser)
				require.NoError(t, err)
			}

			// Get first page with 5 results
			page1 := repository.Pagination{Page: 1, PageSize: 5}
			users1, err := userRepo.Search("example.com", "", "", page1)
			require.NoError(t, err)
			assert.Len(t, users1, 5, "Should return 5 users for first page")

			// Get second page with 5 results
			page2 := repository.Pagination{Page: 2, PageSize: 5}
			users2, err := userRepo.Search("example.com", "", "", page2)
			require.NoError(t, err)
			assert.Len(t, users2, 5, "Should return 5 users for second page")

			// Ensure pages have different users
			for _, user1 := range users1 {
				for _, user2 := range users2 {
					assert.NotEqual(t, user1.ID, user2.ID, "Users on different pages should be different")
				}
			}
		})
	})

	t.Run("Delete", func(t *testing.T) {
		err := userRepo.Delete(testUser.ID)
		require.NoError(t, err, "Should delete user without error")

		_, err = userRepo.FindByID(testUser.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Deleted user should not be found")

		users, err := userRepo.Search("", "", "", defaultPage)
		require.NoError(t, err, "Should list all users")
		assert.GreaterOrEqual(t, len(users), 10, "Should have at least 10 users after deletion")
		for _, user := range users {
			assert.NotEqual(t, testUser.ID, user.ID, "Deleted user should not be in the list")
		}

		err = userRepo.Delete(9999)
		assert.EqualError(t, err, "user not found", "Should return error when deleting non-existent user")
	})

	utils.CleanTestData(t)
}
