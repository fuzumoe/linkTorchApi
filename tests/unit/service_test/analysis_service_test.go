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

type MockAnalysisRepo struct {
	mock.Mock
}

func (m *MockAnalysisRepo) Create(ar *model.AnalysisResult, links []model.Link) error {
	args := m.Called(ar, links)
	return args.Error(0)
}

func (m *MockAnalysisRepo) ListByURL(urlID uint, p repository.Pagination) ([]model.AnalysisResult, error) {
	args := m.Called(urlID, p)
	return args.Get(0).([]model.AnalysisResult), args.Error(1)
}

func TestAnalysisService_Record(t *testing.T) {

	mockRepo := new(MockAnalysisRepo)
	svc := service.NewAnalysisService(mockRepo)

	testResult := &model.AnalysisResult{
		URLID:        42,
		HTMLVersion:  "HTML5",
		Title:        "Test Page",
		H1Count:      2,
		H2Count:      5,
		H3Count:      3,
		H4Count:      0,
		H5Count:      0,
		H6Count:      0,
		HasLoginForm: true,
	}

	t.Run("Success", func(t *testing.T) {

		emptyLinks := []model.Link{}
		mockRepo.On("Create", testResult, emptyLinks).Return(nil).Once()

		err := svc.Record(testResult, emptyLinks)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		emptyLinks := []model.Link{}
		expectedErr := errors.New("database error")
		mockRepo.On("Create", testResult, emptyLinks).Return(expectedErr).Once()

		err := svc.Record(testResult, emptyLinks)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestAnalysisService_List(t *testing.T) {

	mockRepo := new(MockAnalysisRepo)
	svc := service.NewAnalysisService(mockRepo)

	urlID := uint(42)
	pagination := repository.Pagination{Page: 1, PageSize: 10}

	analysisResults := []model.AnalysisResult{
		{
			ID:           1,
			URLID:        urlID,
			HTMLVersion:  "HTML5",
			Title:        "First Page",
			H1Count:      2,
			H2Count:      5,
			H3Count:      3,
			HasLoginForm: true,
		},
		{
			ID:           2,
			URLID:        urlID,
			HTMLVersion:  "HTML4",
			Title:        "Second Page",
			H1Count:      1,
			H2Count:      3,
			H3Count:      2,
			HasLoginForm: false,
		},
	}

	t.Run("Success", func(t *testing.T) {

		mockRepo.On("ListByURL", urlID, pagination).Return(analysisResults, nil).Once()

		dtos, err := svc.List(urlID, pagination)

		require.NoError(t, err)
		require.Len(t, dtos, 2, "Should return 2 DTOs")

		assert.Equal(t, uint(1), dtos[0].ID)
		assert.Equal(t, "HTML5", dtos[0].HTMLVersion)
		assert.Equal(t, "First Page", dtos[0].Title)
		assert.Equal(t, 2, dtos[0].H1Count)
		assert.Equal(t, 5, dtos[0].H2Count)
		assert.Equal(t, 3, dtos[0].H3Count)
		assert.True(t, dtos[0].HasLoginForm)
		assert.Equal(t, uint(2), dtos[1].ID)
		assert.Equal(t, "HTML4", dtos[1].HTMLVersion)
		assert.Equal(t, "Second Page", dtos[1].Title)
		assert.Equal(t, 1, dtos[1].H1Count)
		assert.Equal(t, 3, dtos[1].H2Count)
		assert.Equal(t, 2, dtos[1].H3Count)
		assert.False(t, dtos[1].HasLoginForm)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Empty Results", func(t *testing.T) {
		mockRepo.On("ListByURL", urlID, pagination).Return([]model.AnalysisResult{}, nil).Once()

		dtos, err := svc.List(urlID, pagination)

		require.NoError(t, err)
		assert.Empty(t, dtos, "Should return empty slice")
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		expectedErr := errors.New("database error")
		mockRepo.On("ListByURL", urlID, pagination).Return([]model.AnalysisResult{}, expectedErr).Once()

		dtos, err := svc.List(urlID, pagination)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, dtos, "Should return nil on error")
		mockRepo.AssertExpectations(t)
	})
}
