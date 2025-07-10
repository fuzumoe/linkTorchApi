package service_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/internal/service"
)

// MockURLRepo is a mock implementation of URLRepository
type MockURLRepo struct {
	mock.Mock
}

func (m *MockURLRepo) Create(url *model.URL) error {
	args := m.Called(url)
	return args.Error(0)
}

func (m *MockURLRepo) FindByID(id uint) (*model.URL, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.URL), args.Error(1)
}

func (m *MockURLRepo) ListByUser(userID uint, p repository.Pagination) ([]model.URL, error) {
	args := m.Called(userID, p)
	return args.Get(0).([]model.URL), args.Error(1)
}

func (m *MockURLRepo) Update(url *model.URL) error {
	args := m.Called(url)
	return args.Error(0)
}

func (m *MockURLRepo) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func TestURLService_Create(t *testing.T) {
	// Setup
	mockRepo := new(MockURLRepo)
	svc := service.NewURLService(mockRepo)

	// Test data
	input := &model.CreateURLInput{
		UserID:      1,
		OriginalURL: "https://example.com",
	}

	t.Run("Success", func(t *testing.T) {
		// Setup expectations - when repo.Create is called with a URL created from input,
		// it should set the ID to 42 and return nil error
		mockRepo.On("Create", mock.MatchedBy(func(u *model.URL) bool {
			return u.UserID == input.UserID && u.OriginalURL == input.OriginalURL
		})).Run(func(args mock.Arguments) {
			url := args.Get(0).(*model.URL)
			url.ID = 42
		}).Return(nil).Once()

		// Execute
		id, err := svc.Create(input)

		// Verify
		assert.NoError(t, err)
		assert.Equal(t, uint(42), id)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		// Setup expectations - simulate repository error
		expectedErr := errors.New("database error")
		mockRepo.On("Create", mock.MatchedBy(func(u *model.URL) bool {
			return u.UserID == input.UserID && u.OriginalURL == input.OriginalURL
		})).Return(expectedErr).Once()

		// Execute
		id, err := svc.Create(input)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, uint(0), id)
		mockRepo.AssertExpectations(t)
	})
}

func TestURLService_Get(t *testing.T) {
	// Setup
	mockRepo := new(MockURLRepo)
	svc := service.NewURLService(mockRepo)

	// Test data
	urlID := uint(42)
	testURL := &model.URL{
		ID:          urlID,
		UserID:      1,
		OriginalURL: "https://example.com",
		Status:      "queued",
	}

	t.Run("Success", func(t *testing.T) {
		// Setup expectations
		mockRepo.On("FindByID", urlID).Return(testURL, nil).Once()

		// Execute
		dto, err := svc.Get(urlID)

		// Verify
		require.NoError(t, err)
		assert.NotNil(t, dto)
		assert.Equal(t, urlID, dto.ID)
		assert.Equal(t, testURL.UserID, dto.UserID)
		assert.Equal(t, testURL.OriginalURL, dto.OriginalURL)
		assert.Equal(t, testURL.Status, dto.Status)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		// Setup expectations
		expectedErr := errors.New("record not found")
		mockRepo.On("FindByID", urlID).Return(nil, expectedErr).Once()

		// Execute
		dto, err := svc.Get(urlID)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, dto)
		mockRepo.AssertExpectations(t)
	})
}

func TestURLService_List(t *testing.T) {
	// Setup
	mockRepo := new(MockURLRepo)
	svc := service.NewURLService(mockRepo)

	// Test data
	userID := uint(1)
	pagination := repository.Pagination{Page: 1, PageSize: 10}
	urls := []model.URL{
		{
			ID:          1,
			UserID:      userID,
			OriginalURL: "https://example1.com",
			Status:      "done",
		},
		{
			ID:          2,
			UserID:      userID,
			OriginalURL: "https://example2.com",
			Status:      "queued",
		},
	}

	t.Run("Success", func(t *testing.T) {
		// Setup expectations
		mockRepo.On("ListByUser", userID, pagination).Return(urls, nil).Once()

		// Execute
		dtos, err := svc.List(userID, pagination)

		// Verify
		require.NoError(t, err)
		require.Len(t, dtos, 2)

		// Verify first DTO
		assert.Equal(t, uint(1), dtos[0].ID)
		assert.Equal(t, userID, dtos[0].UserID)
		assert.Equal(t, "https://example1.com", dtos[0].OriginalURL)
		assert.Equal(t, "done", dtos[0].Status)

		// Verify second DTO
		assert.Equal(t, uint(2), dtos[1].ID)
		assert.Equal(t, userID, dtos[1].UserID)
		assert.Equal(t, "https://example2.com", dtos[1].OriginalURL)
		assert.Equal(t, "queued", dtos[1].Status)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Empty Results", func(t *testing.T) {
		// Setup expectations
		mockRepo.On("ListByUser", userID, pagination).Return([]model.URL{}, nil).Once()

		// Execute
		dtos, err := svc.List(userID, pagination)

		// Verify
		require.NoError(t, err)
		assert.Empty(t, dtos)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		// Setup expectations
		expectedErr := errors.New("database error")
		mockRepo.On("ListByUser", userID, pagination).Return([]model.URL{}, expectedErr).Once()

		// Execute
		dtos, err := svc.List(userID, pagination)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, dtos)
		mockRepo.AssertExpectations(t)
	})
}

func TestURLService_Update(t *testing.T) {
	// Setup
	mockRepo := new(MockURLRepo)
	svc := service.NewURLService(mockRepo)

	// Test data
	urlID := uint(42)

	t.Run("Update Original URL", func(t *testing.T) {
		// For each test case, create a fresh instance of the existing URL
		existingURL := &model.URL{
			ID:          urlID,
			UserID:      1,
			OriginalURL: "https://old-example.com",
			Status:      "queued",
		}

		// Input to update only the original URL
		input := &model.UpdateURLInput{
			OriginalURL: "https://new-example.com",
		}

		// Setup expectations
		mockRepo.On("FindByID", urlID).Return(existingURL, nil).Once()
		mockRepo.On("Update", mock.AnythingOfType("*model.URL")).Return(nil).Once()

		// Execute
		err := svc.Update(urlID, input)

		// Verify
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Update Status", func(t *testing.T) {
		// For each test case, create a fresh instance of the existing URL
		existingURL := &model.URL{
			ID:          urlID,
			UserID:      1,
			OriginalURL: "https://old-example.com",
			Status:      "queued",
		}

		// Input to update only the status
		input := &model.UpdateURLInput{
			Status: "done",
		}

		// Setup expectations
		mockRepo.On("FindByID", urlID).Return(existingURL, nil).Once()
		mockRepo.On("Update", mock.AnythingOfType("*model.URL")).Return(nil).Once()

		// Execute
		err := svc.Update(urlID, input)

		// Verify
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Invalid Status", func(t *testing.T) {
		// For each test case, create a fresh instance of the existing URL
		existingURL := &model.URL{
			ID:          urlID,
			UserID:      1,
			OriginalURL: "https://old-example.com",
			Status:      "queued",
		}

		// Input with invalid status
		input := &model.UpdateURLInput{
			Status: "invalid_status",
		}

		// Setup expectations
		mockRepo.On("FindByID", urlID).Return(existingURL, nil).Once()

		// Execute
		err := svc.Update(urlID, input)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, "invalid status value", err.Error())
		mockRepo.AssertExpectations(t)
	})

	t.Run("URL Not Found", func(t *testing.T) {
		// Input doesn't matter for this test
		input := &model.UpdateURLInput{
			OriginalURL: "https://new-example.com",
		}

		// Setup expectations
		expectedErr := errors.New("record not found")
		mockRepo.On("FindByID", urlID).Return(nil, expectedErr).Once()

		// Execute
		err := svc.Update(urlID, input)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Update Error", func(t *testing.T) {
		// For each test case, create a fresh instance of the existing URL
		existingURL := &model.URL{
			ID:          urlID,
			UserID:      1,
			OriginalURL: "https://old-example.com",
			Status:      "queued",
		}

		// Input to update
		input := &model.UpdateURLInput{
			OriginalURL: "https://new-example.com",
		}

		// Setup expectations
		mockRepo.On("FindByID", urlID).Return(existingURL, nil).Once()
		expectedErr := errors.New("update error")
		mockRepo.On("Update", mock.AnythingOfType("*model.URL")).Return(expectedErr).Once()

		// Execute
		err := svc.Update(urlID, input)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestURLService_Delete(t *testing.T) {
	// Setup
	mockRepo := new(MockURLRepo)
	svc := service.NewURLService(mockRepo)

	// Test data
	urlID := uint(42)

	t.Run("Success", func(t *testing.T) {
		// Setup expectations
		mockRepo.On("Delete", urlID).Return(nil).Once()

		// Execute
		err := svc.Delete(urlID)

		// Verify
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		// Setup expectations
		expectedErr := errors.New("url not found")
		mockRepo.On("Delete", urlID).Return(expectedErr).Once()

		// Execute
		err := svc.Delete(urlID)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}
