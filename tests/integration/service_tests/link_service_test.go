package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/internal/service"
	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

func TestLinkService_Integration(t *testing.T) {

	db := utils.SetupTest(t)
	defer utils.CleanTestData(t)

	userRepo := repository.NewUserRepo(db)
	urlRepo := repository.NewURLRepo(db)
	linkRepo := repository.NewLinkRepo(db)
	linkService := service.NewLinkService(linkRepo)

	testUser := &model.User{
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := userRepo.Create(testUser)
	require.NoError(t, err, "Should create test user without error.")
	require.NotZero(t, testUser.ID, "User ID should be set after creation.")

	testURL := &model.URL{
		UserID:      testUser.ID,
		OriginalURL: "https://test-domain.com",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = urlRepo.Create(testURL)
	require.NoError(t, err, "Should create test URL without error.")
	require.NotZero(t, testURL.ID, "URL ID should be set after creation.")

	urlID := testURL.ID

	createTestLink := func(t *testing.T, hrefSuffix string) *model.Link {
		link := &model.Link{
			URLID:      urlID,
			Href:       "https://example.com/" + hrefSuffix,
			IsExternal: false,
			StatusCode: 200,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		err := linkService.Add(link)
		require.NoError(t, err, "Should add link without error.")
		require.NotZero(t, link.ID, "Link ID should be set after creation.")
		return link
	}

	t.Run("Add", func(t *testing.T) {

		link := &model.Link{
			URLID:      urlID,
			Href:       "https://example.com/test",
			IsExternal: true,
			StatusCode: 200,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := linkService.Add(link)
		assert.NoError(t, err, "Should add link without error.")
		assert.NotZero(t, link.ID, "Link ID should be set after creation.")
	})

	t.Run("List", func(t *testing.T) {

		createTestLink(t, "Link1")
		createTestLink(t, "Link2")
		createTestLink(t, "Link3")

		pagination := repository.Pagination{
			Page:     1,
			PageSize: 10,
		}

		links, err := linkService.List(urlID, pagination)
		assert.NoError(t, err, "Should list links without error.")
		assert.GreaterOrEqual(t, len(links), 3, "Should return at least 3 links.")

		for _, link := range links {
			assert.Equal(t, urlID, link.URLID, "Link should have the correct URL ID.")
			assert.NotEmpty(t, link.Href, "Link should have an href.")
		}

		paginationSmall := repository.Pagination{
			Page:     1,
			PageSize: 2,
		}
		linksPage1, err := linkService.List(urlID, paginationSmall)
		assert.NoError(t, err, "Should list links without error.")
		assert.Len(t, linksPage1, 2, "Should return exactly 2 links for page 1.")

		paginationPage2 := repository.Pagination{
			Page:     2,
			PageSize: 2,
		}
		linksPage2, err := linkService.List(urlID, paginationPage2)
		assert.NoError(t, err, "Should list links without error.")
		assert.GreaterOrEqual(t, len(linksPage2), 1, "Should have at least 1 link on page 2.")

		if len(linksPage2) > 0 {
			assert.NotEqual(t, linksPage1[0].ID, linksPage2[0].ID, "Links on different pages should be different.")
		}
	})

	t.Run("ListByURL", func(t *testing.T) {

		createTestLink(t, "PaginatedLink1")
		createTestLink(t, "PaginatedLink2")
		createTestLink(t, "PaginatedLink3")

		pagination := repository.Pagination{
			Page:     1,
			PageSize: 10,
		}

		paginatedResult, err := linkService.ListByURL(urlID, pagination)
		require.NoError(t, err, "Should list links without error.")

		assert.Equal(t, 1, paginatedResult.Pagination.Page, "Page should be 1")
		assert.Equal(t, 10, paginatedResult.Pagination.PageSize, "PageSize should be 10")
		assert.GreaterOrEqual(t, paginatedResult.Pagination.TotalItems, 6, "TotalItems should include all created links")
		assert.GreaterOrEqual(t, paginatedResult.Pagination.TotalPages, 1, "Should have at least 1 page")

		assert.NotEmpty(t, paginatedResult.Data, "Data should not be empty")
		assert.GreaterOrEqual(t, len(paginatedResult.Data), 6, "Should return at least 6 links")

		for _, link := range paginatedResult.Data {
			assert.Equal(t, urlID, link.URLID, "Link should have the correct URL ID.")
			assert.NotEmpty(t, link.Href, "Link should have an href.")
		}

		smallPagination := repository.Pagination{
			Page:     1,
			PageSize: 3,
		}

		smallPageResult, err := linkService.ListByURL(urlID, smallPagination)
		require.NoError(t, err, "Should list links without error.")

		assert.Equal(t, 1, smallPageResult.Pagination.Page, "Page should be 1")
		assert.Equal(t, 3, smallPageResult.Pagination.PageSize, "PageSize should be 3")
		assert.Len(t, smallPageResult.Data, 3, "Should return exactly 3 links")
		assert.GreaterOrEqual(t, smallPageResult.Pagination.TotalPages, 2, "Should have at least 2 pages")

		page2Pagination := repository.Pagination{
			Page:     2,
			PageSize: 3,
		}

		page2Result, err := linkService.ListByURL(urlID, page2Pagination)
		require.NoError(t, err, "Should list links without error.")

		assert.Equal(t, 2, page2Result.Pagination.Page, "Page should be 2")
		assert.Equal(t, 3, page2Result.Pagination.PageSize, "PageSize should be 3")
		assert.NotEmpty(t, page2Result.Data, "Page 2 should have data")

		if len(page2Result.Data) > 0 && len(smallPageResult.Data) > 0 {
			assert.NotEqual(t, smallPageResult.Data[0].ID, page2Result.Data[0].ID,
				"Links on different pages should be different")
		}
	})

	t.Run("Update", func(t *testing.T) {

		link := createTestLink(t, "UpdateTest")

		link.Href = "https://example.com/updated"
		link.IsExternal = true
		link.StatusCode = 301
		link.UpdatedAt = time.Now()

		err := linkService.Update(link)
		assert.NoError(t, err, "Should update link without error.")

		paginatedResult, err := linkService.ListByURL(urlID, repository.Pagination{Page: 1, PageSize: 100})
		assert.NoError(t, err, "Should list links without error.")

		var updatedLink model.LinkDTO
		for _, l := range paginatedResult.Data {
			if l.ID == link.ID {
				updatedLink = l
				break
			}
		}

		assert.Equal(t, link.ID, updatedLink.ID, "Updated link should be found in the list.")
		assert.Equal(t, "https://example.com/updated", updatedLink.Href, "Link href should be updated.")
		assert.Equal(t, true, updatedLink.IsExternal, "Link IsExternal should be updated.")
		assert.Equal(t, 301, updatedLink.StatusCode, "Link StatusCode should be updated.")
	})

	t.Run("Delete", func(t *testing.T) {

		link := createTestLink(t, "DeleteTest")

		initialResult, err := linkService.ListByURL(urlID, repository.Pagination{Page: 1, PageSize: 100})
		assert.NoError(t, err, "Should list links without error.")
		initialCount := len(initialResult.Data)

		err = linkService.Delete(link)
		assert.NoError(t, err, "Should delete link without error.")

		afterResult, err := linkService.ListByURL(urlID, repository.Pagination{Page: 1, PageSize: 100})
		assert.NoError(t, err, "Should list links without error.")
		afterCount := len(afterResult.Data)

		assert.Equal(t, initialCount-1, afterCount, "One link should be deleted.")

		found := false
		for _, l := range afterResult.Data {
			if l.ID == link.ID {
				found = true
				break
			}
		}
		assert.False(t, found, "Deleted link should not be found in the list.")
	})

	t.Run("ErrorCases", func(t *testing.T) {

		invalidLink := &model.Link{
			URLID:      urlID,
			Href:       "",
			StatusCode: 200,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		err := linkService.Add(invalidLink)

		assert.NoError(t, err, "Adding link with empty Href should not return error by default.")

		nonExistentLink := &model.Link{
			ID:         999999,
			URLID:      urlID,
			Href:       "https://example.com/non-existent",
			IsExternal: false,
			StatusCode: 200,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		err = linkService.Update(nonExistentLink)

		assert.NoError(t, err, "Updating non-existent link should not return error by default.")

		err = linkService.Delete(nonExistentLink)

		assert.NoError(t, err, "Deleting non-existent link should not return error by default.")
	})
}
