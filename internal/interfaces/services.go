package interfaces

import (
	"context"
	"donationbars/internal/models"
)

// BarServiceInterface defines the contract for bar operations
type BarServiceInterface interface {
	CreateBar(userID string, req *models.CreateBarRequest) (*models.DonationBar, error)
	CreateBarFromAI(userID, prompt string, aiResponse *models.AIGenerateResponse, initialAmount, goalAmount float64) (*models.DonationBar, error)
	GetUserBars(userID string) ([]*models.DonationBar, error)
	GetBar(userID, barID string) (*models.DonationBar, error)
	UpdateBar(userID, barID string, req *models.UpdateBarRequest) (*models.DonationBar, error)
	UpdateBarComplete(userID, barID string, req *models.CreateBarRequest, isActive bool) error
	DeleteBar(userID, barID string) error
	GetUserBarCount(userID string) (int64, error)
	GetUserDailyBarCount(userID string) (int64, error)
	CheckDailyRateLimit(userID string) error
}

// AIServiceInterface defines the contract for AI operations
type AIServiceInterface interface {
	GenerateBar(req *models.GenerateBarRequest) (*models.AIGenerateResponse, error)
}

// BarRepositoryInterface defines the contract for bar data operations
type BarRepositoryInterface interface {
	Insert(ctx context.Context, bar *models.DonationBar) error
	FindByUserID(ctx context.Context, userID string) ([]*models.DonationBar, error)
	FindByID(ctx context.Context, userID, barID string) (*models.DonationBar, error)
	Update(ctx context.Context, userID, barID string, req *models.UpdateBarRequest) (*models.DonationBar, error)
	UpdateComplete(ctx context.Context, userID, barID string, req *models.CreateBarRequest, isActive bool) error
	Delete(ctx context.Context, userID, barID string) error
	CountByUserID(ctx context.Context, userID string) (int64, error)
	CountByUserIDToday(ctx context.Context, userID string) (int64, error)
}
