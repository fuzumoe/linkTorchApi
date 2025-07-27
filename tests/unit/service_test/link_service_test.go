package service_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/internal/service"
)

type MockLinkRepo struct {
	mock.Mock
}

func (m *MockLinkRepo) Create(link *model.Link) error {
	args := m.Called(link)
	return args.Error(0)
}

func (m *MockLinkRepo) ListByURL(urlID uint, p repository.Pagination) ([]model.Link, error) {
	args := m.Called(urlID, p)
	return args.Get(0).([]model.Link), args.Error(1)
}

func (m *MockLinkRepo) CountByURL(urlID uint) (int, error) {
	args := m.Called(urlID)
	return args.Int(0), args.Error(1)
}

func (m *MockLinkRepo) Update(link *model.Link) error {
	args := m.Called(link)
	return args.Error(0)
}

func (m *MockLinkRepo) Delete(link *model.Link) error {
	args := m.Called(link)
	return args.Error(0)
}

func testSimpleRepoOperation(t *testing.T, testName string, operation func(repo *MockLinkRepo) error) {
	mockRepo := new(MockLinkRepo)

	t.Run("Success", func(t *testing.T) {
		mockRepo.On(testName, mock.Anything).Return(nil).Once()
		assert.NoError(t, operation(mockRepo))
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		expectedErr := errors.New("database error")
		mockRepo.On(testName, mock.Anything).Return(expectedErr).Once()
		err := operation(mockRepo)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestLinkService_Add(t *testing.T) {
	testLink := &model.Link{
		URLID:      42,
		Href:       "https://example.com",
		IsExternal: true,
		StatusCode: 200,
	}

	testSimpleRepoOperation(t, "Create", func(mockRepo *MockLinkRepo) error {
		svc := service.NewLinkService(mockRepo)
		return svc.Add(testLink)
	})
}

func TestLinkService_List(t *testing.T) {

	mockRepo := new(MockLinkRepo)
	svc := service.NewLinkService(mockRepo)

	urlID := uint(42)
	pagination := repository.Pagination{Page: 1, PageSize: 10}

	links := []model.Link{
		{
			ID:         1,
			URLID:      urlID,
			Href:       "https://example.com/page1",
			IsExternal: true,
			StatusCode: 200,
		},
		{
			ID:         2,
			URLID:      urlID,
			Href:       "https://example.com/page2",
			IsExternal: false,
			StatusCode: 301,
		},
	}

	t.Run("Success", func(t *testing.T) {

		mockRepo.On("ListByURL", urlID, pagination).Return(links, nil).Once()

		dtos, err := svc.List(urlID, pagination)

		require.NoError(t, err)
		require.Len(t, dtos, 2, "Should return 2 DTOs")

		assert.Equal(t, uint(1), dtos[0].ID)
		assert.Equal(t, "https://example.com/page1", dtos[0].Href)
		assert.True(t, dtos[0].IsExternal)
		assert.Equal(t, 200, dtos[0].StatusCode)

		assert.Equal(t, uint(2), dtos[1].ID)
		assert.Equal(t, "https://example.com/page2", dtos[1].Href)
		assert.False(t, dtos[1].IsExternal)
		assert.Equal(t, 301, dtos[1].StatusCode)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Empty Results", func(t *testing.T) {
		mockRepo.On("ListByURL", urlID, pagination).Return([]model.Link{}, nil).Once()
		dtos, err := svc.List(urlID, pagination)

		require.NoError(t, err)
		assert.Empty(t, dtos, "Should return empty slice")
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		expectedErr := errors.New("database error")
		mockRepo.On("ListByURL", urlID, pagination).Return([]model.Link{}, expectedErr).Once()

		dtos, err := svc.List(urlID, pagination)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, dtos, "Should return nil on error")
		mockRepo.AssertExpectations(t)
	})
}

func TestLinkService_ListByURL(t *testing.T) {

	mockRepo := new(MockLinkRepo)
	svc := service.NewLinkService(mockRepo)

	urlID := uint(42)
	pagination := repository.Pagination{Page: 1, PageSize: 10}

	links := []model.Link{
		{
			ID:         1,
			URLID:      urlID,
			Href:       "https://example.com/page1",
			IsExternal: true,
			StatusCode: 200,
		},
		{
			ID:         2,
			URLID:      urlID,
			Href:       "https://example.com/page2",
			IsExternal: false,
			StatusCode: 301,
		},
	}

	t.Run("Success", func(t *testing.T) {

		mockRepo.On("ListByURL", urlID, pagination).Return(links, nil).Once()
		mockRepo.On("CountByURL", urlID).Return(2, nil).Once()

		result, err := svc.ListByURL(urlID, pagination)

		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, 1, result.Pagination.Page)
		assert.Equal(t, 10, result.Pagination.PageSize)
		assert.Equal(t, 2, result.Pagination.TotalItems)
		assert.Equal(t, 1, result.Pagination.TotalPages)

		require.Len(t, result.Data, 2, "Should return 2 DTOs")

		assert.Equal(t, uint(1), result.Data[0].ID)
		assert.Equal(t, "https://example.com/page1", result.Data[0].Href)
		assert.True(t, result.Data[0].IsExternal)
		assert.Equal(t, 200, result.Data[0].StatusCode)

		assert.Equal(t, uint(2), result.Data[1].ID)
		assert.Equal(t, "https://example.com/page2", result.Data[1].Href)
		assert.False(t, result.Data[1].IsExternal)
		assert.Equal(t, 301, result.Data[1].StatusCode)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Empty Results", func(t *testing.T) {
		mockRepo.On("ListByURL", urlID, pagination).Return([]model.Link{}, nil).Once()
		mockRepo.On("CountByURL", urlID).Return(0, nil).Once()

		result, err := svc.ListByURL(urlID, pagination)

		require.NoError(t, err)
		assert.Empty(t, result.Data, "Should return empty data array")
		assert.Equal(t, 0, result.Pagination.TotalItems)
		assert.Equal(t, 0, result.Pagination.TotalPages)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error on ListByURL", func(t *testing.T) {
		expectedErr := errors.New("database error")
		mockRepo.On("ListByURL", urlID, pagination).Return([]model.Link{}, expectedErr).Once()

		result, err := svc.ListByURL(urlID, pagination)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result, "Should return nil on error")
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error on CountByURL", func(t *testing.T) {
		mockRepo.On("ListByURL", urlID, pagination).Return(links, nil).Once()
		expectedErr := errors.New("count error")
		mockRepo.On("CountByURL", urlID).Return(0, expectedErr).Once()

		result, err := svc.ListByURL(urlID, pagination)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result, "Should return nil on error")
		mockRepo.AssertExpectations(t)
	})

	t.Run("Multiple Pages", func(t *testing.T) {
		mockRepo.On("ListByURL", urlID, pagination).Return(links, nil).Once()
		mockRepo.On("CountByURL", urlID).Return(21, nil).Once()

		result, err := svc.ListByURL(urlID, pagination)

		require.NoError(t, err)
		assert.Equal(t, 21, result.Pagination.TotalItems)
		assert.Equal(t, 3, result.Pagination.TotalPages)
		mockRepo.AssertExpectations(t)
	})
}

func TestLinkService_Update(t *testing.T) {
	testLink := &model.Link{
		ID:         1,
		URLID:      42,
		Href:       "https://updated-example.com",
		IsExternal: false,
		StatusCode: 301,
	}

	testSimpleRepoOperation(t, "Update", func(mockRepo *MockLinkRepo) error {
		svc := service.NewLinkService(mockRepo)
		return svc.Update(testLink)
	})
}

func TestLinkService_Delete(t *testing.T) {
	testLink := &model.Link{
		ID:         1,
		URLID:      42,
		Href:       "https://example.com",
		IsExternal: true,
		StatusCode: 200,
	}

	testSimpleRepoOperation(t, "Delete", func(mockRepo *MockLinkRepo) error {
		svc := service.NewLinkService(mockRepo)
		return svc.Delete(testLink)
	})
}
