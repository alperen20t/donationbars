package services

import (
	"testing"
	"time"

	"donationbars/internal/config"
	apperrors "donationbars/internal/errors"
	"donationbars/internal/mocks"
	"donationbars/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func createTestConfig() *config.Config {
	return &config.Config{
		MaxBarsPerUser:  5,
		RateLimitPerDay: 5,
		Timeouts: config.TimeoutConfig{
			DatabaseRead:   5 * time.Second,
			DatabaseWrite:  10 * time.Second,
			RedisOperation: 2 * time.Second,
		},
	}
}

func createTestRedisClient() *config.RedisClient {
	return &config.RedisClient{
		Client:  nil,
		Enabled: false, // Disabled for tests
	}
}

func TestBarService_CreateBar_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockBarRepository)
	redisClient := createTestRedisClient()
	cfg := createTestConfig()
	service := NewBarService(mockRepo, redisClient, cfg)

	userID := "test-user"
	req := &models.CreateBarRequest{
		Name:          "Test Bar",
		Description:   "Test Description",
		HTML:          "<div>{goal} {total} {percentage} {remaining} {description}</div>",
		CSS:           ".bar { width: 800px; height: 200px; }",
		Language:      "tr",
		Theme:         "modern",
		InitialAmount: 100.0,
		GoalAmount:    1000.0,
	}

	// Mock expectations
	mockRepo.On("CountByUserIDToday", mock.Anything, userID).Return(int64(2), nil)
	mockRepo.On("CountByUserID", mock.Anything, userID).Return(int64(3), nil)
	mockRepo.On("Insert", mock.Anything, mock.AnythingOfType("*models.DonationBar")).Return(nil)

	// Act
	result, err := service.CreateBar(userID, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, req.Name, result.Name)
	assert.Equal(t, req.Description, result.Description)
	assert.Equal(t, userID, result.UserID)
	assert.False(t, result.AIGenerated)
	assert.True(t, result.HasValidInjections)
	mockRepo.AssertExpectations(t)
}

func TestBarService_CreateBar_RateLimitExceeded(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockBarRepository)
	redisClient := createTestRedisClient()
	cfg := createTestConfig()
	service := NewBarService(mockRepo, redisClient, cfg)

	userID := "test-user"
	req := &models.CreateBarRequest{
		Name:          "Test Bar",
		HTML:          "<div>{goal} {total} {percentage} {remaining} {description}</div>",
		CSS:           ".bar { width: 800px; }",
		Language:      "tr",
		InitialAmount: 100.0,
		GoalAmount:    1000.0,
	}

	// Mock expectations - user has reached daily limit
	mockRepo.On("CountByUserIDToday", mock.Anything, userID).Return(int64(5), nil)

	// Act
	result, err := service.CreateBar(userID, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)

	var appErr *apperrors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, "RATE_LIMIT_EXCEEDED", appErr.Type)
	mockRepo.AssertExpectations(t)
}

func TestBarService_CreateBar_MaxBarsReached(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockBarRepository)
	redisClient := createTestRedisClient()
	cfg := createTestConfig()
	service := NewBarService(mockRepo, redisClient, cfg)

	userID := "test-user"
	req := &models.CreateBarRequest{
		Name:          "Test Bar",
		HTML:          "<div>{goal} {total} {percentage} {remaining} {description}</div>",
		CSS:           ".bar { width: 800px; }",
		Language:      "tr",
		InitialAmount: 100.0,
		GoalAmount:    1000.0,
	}

	// Mock expectations
	mockRepo.On("CountByUserIDToday", mock.Anything, userID).Return(int64(2), nil)
	mockRepo.On("CountByUserID", mock.Anything, userID).Return(int64(5), nil) // Max reached

	// Act
	result, err := service.CreateBar(userID, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)

	var appErr *apperrors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, "MAX_BARS_REACHED", appErr.Type)
	mockRepo.AssertExpectations(t)
}

func TestBarService_CreateBar_InvalidInjections(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockBarRepository)
	redisClient := createTestRedisClient()
	cfg := createTestConfig()
	service := NewBarService(mockRepo, redisClient, cfg)

	userID := "test-user"
	req := &models.CreateBarRequest{
		Name:          "Test Bar",
		HTML:          "<div>{goal} {total}</div>", // Missing required injections
		CSS:           ".bar { width: 800px; }",
		Language:      "tr",
		InitialAmount: 100.0,
		GoalAmount:    1000.0,
	}

	// Mock expectations
	mockRepo.On("CountByUserIDToday", mock.Anything, userID).Return(int64(2), nil)
	mockRepo.On("CountByUserID", mock.Anything, userID).Return(int64(3), nil)

	// Act
	result, err := service.CreateBar(userID, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)

	var appErr *apperrors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, "VALIDATION_ERROR", appErr.Type)
	mockRepo.AssertExpectations(t)
}

func TestBarService_GetBar_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockBarRepository)
	redisClient := createTestRedisClient()
	cfg := createTestConfig()
	service := NewBarService(mockRepo, redisClient, cfg)

	userID := "test-user"
	barID := "507f1f77bcf86cd799439011"

	expectedBar := &models.DonationBar{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		Name:        "Test Bar",
		Description: "Test Description",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, userID, barID).Return(expectedBar, nil)

	// Act
	result, err := service.GetBar(userID, barID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedBar.Name, result.Name)
	assert.Equal(t, expectedBar.UserID, result.UserID)
	mockRepo.AssertExpectations(t)
}

func TestBarService_GetBar_NotFound(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockBarRepository)
	redisClient := createTestRedisClient()
	cfg := createTestConfig()
	service := NewBarService(mockRepo, redisClient, cfg)

	userID := "test-user"
	barID := "507f1f77bcf86cd799439011"

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, userID, barID).Return(nil, assert.AnError)

	// Act
	result, err := service.GetBar(userID, barID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestBarService_ValidateInjections(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockBarRepository)
	redisClient := createTestRedisClient()
	cfg := createTestConfig()
	service := NewBarService(mockRepo, redisClient, cfg)

	tests := []struct {
		name     string
		html     string
		expected bool
	}{
		{
			name:     "Valid HTML with all injections",
			html:     "<div>{goal} {total} {percentage} {remaining} {description}</div>",
			expected: true,
		},
		{
			name:     "Invalid HTML missing injections",
			html:     "<div>{goal} {total}</div>",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.(*BarService).validateInjections(tt.html)
			assert.Equal(t, tt.expected, result)
		})
	}
}
