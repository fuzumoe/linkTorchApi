package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/tests/integration"
)

func TestLinkRepo_Integration(t *testing.T) {
	// Get a clean database state
	db := integration.SetupTest(t)

	// Create repositories
	linkRepo := repository.NewLinkRepo(db)
	urlRepo := repository.NewURLRepo(db)
	userRepo := repository.NewUserRepo(db)

	// First create a test user and URL (needed for Link foreign key)
	testUser := &model.User{
		Username: "linkowner",
		Email:    "linkowner@example.com",
		Password: "password123",
	}
	err := userRepo.Create(testUser)
	require.NoError(t, err, "Should create user without error")
	require.NotZero(t, testUser.ID, "User ID should be set")

	testURL := &model.URL{
		UserID:      testUser.ID,
		OriginalURL: "https://example.com",
		Status:      "queued",
	}
	err = urlRepo.Create(testURL)
	require.NoError(t, err, "Should create URL without error")
	require.NotZero(t, testURL.ID, "URL ID should be set")

	// Test Link data
	testLink := &model.Link{
		URLID:      testURL.ID,
		Href:       "https://linked-site.com",
		IsExternal: true,
		StatusCode: 200,
	}

	t.Run("Create", func(t *testing.T) {
		err := linkRepo.Create(testLink)
		require.NoError(t, err, "Should create Link without error")
		assert.NotZero(t, testLink.ID, "Link ID should be set after creation")
		assert.False(t, testLink.CreatedAt.IsZero(), "CreatedAt should be set")
		assert.False(t, testLink.UpdatedAt.IsZero(), "UpdatedAt should be set")
	})

	t.Run("ListByURL", func(t *testing.T) {
		// Create another link for the same URL
		secondLink := &model.Link{
			URLID:      testURL.ID,
			Href:       "https://second-link.com",
			IsExternal: false,
			StatusCode: 301,
		}
		err := linkRepo.Create(secondLink)
		require.NoError(t, err, "Should create second Link")

		// Create another URL and link for it
		anotherURL := &model.URL{
			UserID:      testUser.ID,
			OriginalURL: "https://another-example.com",
			Status:      "queued",
		}
		err = urlRepo.Create(anotherURL)
		require.NoError(t, err, "Should create another URL")

		otherURLLink := &model.Link{
			URLID:      anotherURL.ID,
			Href:       "https://other-url-link.com",
			IsExternal: true,
			StatusCode: 200,
		}
		err = linkRepo.Create(otherURLLink)
		require.NoError(t, err, "Should create Link for other URL")

		// Test listing links for our test URL
		links, err := linkRepo.ListByURL(testURL.ID)
		require.NoError(t, err, "Should list Links by URL")
		assert.Len(t, links, 2, "Should have 2 Links for test URL")

		// Verify the returned Links belong to our test URL
		for _, l := range links {
			assert.Equal(t, testURL.ID, l.URLID, "Link should belong to test URL")
		}

		// Test listing for the other URL
		otherURLLinks, err := linkRepo.ListByURL(anotherURL.ID)
		require.NoError(t, err, "Should list Links for other URL")
		assert.Len(t, otherURLLinks, 1, "Should have 1 Link for other URL")
		assert.Equal(t, anotherURL.ID, otherURLLinks[0].URLID, "Link should belong to other URL")
	})

	t.Run("Update", func(t *testing.T) {
		// Change Link properties
		testLink.Href = "https://updated-link.com"
		testLink.IsExternal = false
		testLink.StatusCode = 301

		err := linkRepo.Update(testLink)
		require.NoError(t, err, "Should update Link without error")

		// Verify the changes were saved by fetching the updated links
		updatedLinks, err := linkRepo.ListByURL(testURL.ID)
		require.NoError(t, err, "Should find updated Links")

		// Find our updated link in the list
		var found bool
		for _, link := range updatedLinks {
			if link.ID == testLink.ID {
				assert.Equal(t, "https://updated-link.com", link.Href, "Href should be updated")
				assert.False(t, link.IsExternal, "IsExternal should be updated to false")
				assert.Equal(t, 301, link.StatusCode, "StatusCode should be updated")
				found = true
				break
			}
		}
		assert.True(t, found, "Updated link should be found in the list")
	})

	t.Run("Delete", func(t *testing.T) {
		err := linkRepo.Delete(testLink)
		require.NoError(t, err, "Should delete Link without error")

		// Verify Link was deleted by checking it's not in the list
		remainingLinks, err := linkRepo.ListByURL(testURL.ID)
		require.NoError(t, err, "Should list remaining links")

		// Make sure our deleted link is not in the list
		for _, link := range remainingLinks {
			assert.NotEqual(t, testLink.ID, link.ID, "Deleted link should not be in the list")
		}
	})

	integration.CleanTestData(t)
}
