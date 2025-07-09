package repository

import (
	"context"
	"testing"
	"time"

	"donationbars/internal/config"
	"donationbars/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func createTestTimeoutConfig() config.TimeoutConfig {
	return config.TimeoutConfig{
		DatabaseRead:   5 * time.Second,
		DatabaseWrite:  10 * time.Second,
		AI:             30 * time.Second,
		ServerShutdown: 5 * time.Second,
		RedisOperation: 2 * time.Second,
	}
}

func TestBarRepository_NewBarRepository_WithNilDB(t *testing.T) {
	timeouts := createTestTimeoutConfig()
	repo := NewBarRepository(nil, timeouts)

	if repo == nil {
		t.Error("Expected repository to be created, got nil")
	}
}

func TestBarRepository_NewBarRepository_WithValidDB(t *testing.T) {
	// Mock database - in real scenario you'd use test database
	db := &config.Database{
		Client: nil,
		DB:     nil,
	}
	timeouts := createTestTimeoutConfig()

	repo := NewBarRepository(db, timeouts)

	if repo == nil {
		t.Error("Expected repository to be created, got nil")
	}
}

func TestBarRepository_Insert_WithNilDB(t *testing.T) {
	timeouts := createTestTimeoutConfig()
	repo := NewBarRepository(nil, timeouts)

	bar := &models.DonationBar{
		ID:          primitive.NewObjectID(),
		UserID:      "test-user",
		Name:        "Test Bar",
		Description: "Test description",
		HTML:        "<div>{goal} {total} {percentage} {remaining} {description}</div>",
		CSS:         ".bar { width: 800px; height: 200px; }",
		Language:    "tr",
		Theme:       "modern",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Insert(context.Background(), bar)

	if err == nil {
		t.Error("Expected error when database is nil, got nil")
	}

	expectedError := "database connection not available"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestBarRepository_FindByUserID_WithNilDB(t *testing.T) {
	timeouts := createTestTimeoutConfig()
	repo := NewBarRepository(nil, timeouts)

	bars, err := repo.FindByUserID(context.Background(), "test-user")

	if err != nil {
		t.Errorf("Expected no error with nil database, got %v", err)
	}

	if bars == nil {
		t.Error("Expected empty slice, got nil")
	}

	if len(bars) != 0 {
		t.Errorf("Expected empty slice, got %d bars", len(bars))
	}
}

func TestBarRepository_FindByID_WithNilDB(t *testing.T) {
	timeouts := createTestTimeoutConfig()
	repo := NewBarRepository(nil, timeouts)

	bar, err := repo.FindByID(context.Background(), "test-user", "507f1f77bcf86cd799439011")

	if err == nil {
		t.Error("Expected error when database is nil, got nil")
	}

	if bar != nil {
		t.Error("Expected nil bar when database is nil, got bar")
	}

	expectedError := "database connection not available"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestBarRepository_FindByID_WithInvalidID(t *testing.T) {
	timeouts := createTestTimeoutConfig()
	repo := NewBarRepository(nil, timeouts)

	bar, err := repo.FindByID(context.Background(), "test-user", "invalid-id")

	if err == nil {
		t.Error("Expected error with invalid ID, got nil")
	}

	if bar != nil {
		t.Error("Expected nil bar with invalid ID, got bar")
	}

	expectedError := "invalid bar ID format"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestBarRepository_ValidateInjections_Valid(t *testing.T) {
	timeouts := createTestTimeoutConfig()
	repo := NewBarRepository(nil, timeouts)

	// Cast to concrete type to access private method
	concreteRepo := repo.(*BarRepository)

	validHTML := `<div>
		<span>{goal}</span>
		<span>{total}</span>
		<span>{percentage}</span>
		<span>{remaining}</span>
		<span>{description}</span>
	</div>`

	if !concreteRepo.validateInjections(validHTML) {
		t.Error("Expected HTML with all injections to be valid")
	}
}

func TestBarRepository_ValidateInjections_MissingFields(t *testing.T) {
	timeouts := createTestTimeoutConfig()
	repo := NewBarRepository(nil, timeouts)

	// Cast to concrete type to access private method
	concreteRepo := repo.(*BarRepository)

	invalidHTML := `<div>
		<span>{goal}</span>
		<span>{total}</span>
		<!-- Missing {percentage}, {remaining}, {description} -->
	</div>`

	if concreteRepo.validateInjections(invalidHTML) {
		t.Error("Expected HTML with missing injections to be invalid")
	}
}

func TestBarRepository_Update_WithNilDB(t *testing.T) {
	timeouts := createTestTimeoutConfig()
	repo := NewBarRepository(nil, timeouts)

	req := &models.UpdateBarRequest{
		Name:        stringPtr("Updated Name"),
		Description: stringPtr("Updated Description"),
		IsActive:    boolPtr(false),
	}

	bar, err := repo.Update(context.Background(), "test-user", "507f1f77bcf86cd799439011", req)

	if err == nil {
		t.Error("Expected error when database is nil, got nil")
	}

	if bar != nil {
		t.Error("Expected nil bar when database is nil, got bar")
	}

	expectedError := "database connection not available"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestBarRepository_Update_WithInvalidID(t *testing.T) {
	timeouts := createTestTimeoutConfig()
	repo := NewBarRepository(nil, timeouts)

	req := &models.UpdateBarRequest{
		Name: stringPtr("Updated Name"),
	}

	bar, err := repo.Update(context.Background(), "test-user", "invalid-id", req)

	if err == nil {
		t.Error("Expected error with invalid ID, got nil")
	}

	if bar != nil {
		t.Error("Expected nil bar with invalid ID, got bar")
	}

	expectedError := "invalid bar ID format"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestBarRepository_Delete_WithNilDB(t *testing.T) {
	timeouts := createTestTimeoutConfig()
	repo := NewBarRepository(nil, timeouts)

	err := repo.Delete(context.Background(), "test-user", "507f1f77bcf86cd799439011")

	if err == nil {
		t.Error("Expected error when database is nil, got nil")
	}

	expectedError := "database connection not available"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestBarRepository_CountByUserID_WithNilDB(t *testing.T) {
	timeouts := createTestTimeoutConfig()
	repo := NewBarRepository(nil, timeouts)

	count, err := repo.CountByUserID(context.Background(), "test-user")

	if err != nil {
		t.Errorf("Expected no error with nil database, got %v", err)
	}

	if count != 0 {
		t.Errorf("Expected count 0 with nil database, got %d", count)
	}
}

func TestBarRepository_CountByUserIDToday_WithNilDB(t *testing.T) {
	timeouts := createTestTimeoutConfig()
	repo := NewBarRepository(nil, timeouts)

	count, err := repo.CountByUserIDToday(context.Background(), "test-user")

	if err != nil {
		t.Errorf("Expected no error with nil database, got %v", err)
	}

	if count != 0 {
		t.Errorf("Expected count 0 with nil database, got %d", count)
	}
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
