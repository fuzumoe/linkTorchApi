package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fmt"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

func TestLinkRepo_Integration(t *testing.T) {

	db := utils.SetupTest(t)

	linkRepo := repository.NewLinkRepo(db)
	urlRepo := repository.NewURLRepo(db)
	userRepo := repository.NewUserRepo(db)

	defaultPage := repository.Pagination{Page: 1, PageSize: 10}

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

		secondLink := &model.Link{
			URLID:      testURL.ID,
			Href:       "https://second-link.com",
			IsExternal: false,
			StatusCode: 301,
		}
		err := linkRepo.Create(secondLink)
		require.NoError(t, err, "Should create second Link")

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

		links, err := linkRepo.ListByURL(testURL.ID, defaultPage)
		require.NoError(t, err, "Should list Links by URL")
		assert.Len(t, links, 2, "Should have 2 Links for test URL")

		for _, l := range links {
			assert.Equal(t, testURL.ID, l.URLID, "Link should belong to test URL")
		}
		otherURLLinks, err := linkRepo.ListByURL(anotherURL.ID, defaultPage)
		require.NoError(t, err, "Should list Links for other URL")
		assert.Len(t, otherURLLinks, 1, "Should have 1 Link for other URL")
		assert.Equal(t, anotherURL.ID, otherURLLinks[0].URLID, "Link should belong to other URL")
	})

	t.Run("Update", func(t *testing.T) {

		testLink.Href = "https://updated-link.com"
		testLink.IsExternal = false
		testLink.StatusCode = 301

		err := linkRepo.Update(testLink)
		require.NoError(t, err, "Should update Link without error")

		updatedLinks, err := linkRepo.ListByURL(testURL.ID, defaultPage)
		require.NoError(t, err, "Should list updated Links")

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

		remainingLinks, err := linkRepo.ListByURL(testURL.ID, defaultPage)
		require.NoError(t, err, "Should list remaining links")
		for _, link := range remainingLinks {
			assert.NotEqual(t, testLink.ID, link.ID, "Deleted link should not be in the list")
		}
	})

	t.Run("Pagination", func(t *testing.T) {

		for i := 0; i < 5; i++ {
			newLink := &model.Link{
				URLID:      testURL.ID,
				Href:       "https://paginated-link.com/" + fmt.Sprint(rune('A'+i)),
				IsExternal: i%2 == 0,
				StatusCode: 200 + i,
			}
			err := linkRepo.Create(newLink)
			require.NoError(t, err, "Should create paginated link")
		}

		p2 := repository.Pagination{Page: 2, PageSize: 3}
		pagedLinks, err := linkRepo.ListByURL(testURL.ID, p2)
		require.NoError(t, err, "Should list paginated links")

		assert.LessOrEqual(t, len(pagedLinks), 3, "Paginated result should have at most 3 links")
	})

	utils.CleanTestData(t)
}
