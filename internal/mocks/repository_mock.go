package mocks

import (
	"context"

	"donationbars/internal/models"

	"github.com/stretchr/testify/mock"
)

// MockBarRepository is a mock implementation of BarRepositoryInterface
type MockBarRepository struct {
	mock.Mock
}

func (m *MockBarRepository) Insert(ctx context.Context, bar *models.DonationBar) error {
	args := m.Called(ctx, bar)
	return args.Error(0)
}

func (m *MockBarRepository) FindByUserID(ctx context.Context, userID string) ([]*models.DonationBar, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.DonationBar), args.Error(1)
}

func (m *MockBarRepository) FindByID(ctx context.Context, userID, barID string) (*models.DonationBar, error) {
	args := m.Called(ctx, userID, barID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DonationBar), args.Error(1)
}

func (m *MockBarRepository) Update(ctx context.Context, userID, barID string, req *models.UpdateBarRequest) (*models.DonationBar, error) {
	args := m.Called(ctx, userID, barID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DonationBar), args.Error(1)
}

func (m *MockBarRepository) UpdateComplete(ctx context.Context, userID, barID string, req *models.CreateBarRequest, isActive bool) error {
	args := m.Called(ctx, userID, barID, req, isActive)
	return args.Error(0)
}

func (m *MockBarRepository) Delete(ctx context.Context, userID, barID string) error {
	args := m.Called(ctx, userID, barID)
	return args.Error(0)
}

func (m *MockBarRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockBarRepository) CountByUserIDToday(ctx context.Context, userID string) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}
