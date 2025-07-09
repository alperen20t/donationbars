package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"donationbars/internal/config"
	apperrors "donationbars/internal/errors"
	"donationbars/internal/interfaces"
	"donationbars/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BarService struct {
	repo        interfaces.BarRepositoryInterface
	redisClient *config.RedisClient
	config      *config.Config
}

// NewBarService creates a new bar service with repository dependency
func NewBarService(repo interfaces.BarRepositoryInterface, redisClient *config.RedisClient, cfg *config.Config) interfaces.BarServiceInterface {
	return &BarService{
		repo:        repo,
		redisClient: redisClient,
		config:      cfg,
	}
}

// CheckRateLimitRedis checks rate limit using Redis
func (s *BarService) checkRateLimitRedis(userID string) error {
	if !s.redisClient.IsEnabled() {
		// Fallback to database-based rate limiting
		return s.CheckDailyRateLimit(userID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeouts.RedisOperation)
	defer cancel()

	key := fmt.Sprintf("rate_limit:%s", userID)

	// Use Redis pipeline for atomicity
	pipe := s.redisClient.Client.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	expireCmd := pipe.Expire(ctx, key, 24*time.Hour)

	_, err := pipe.Exec(ctx)
	if err != nil {
		slog.Warn("Redis rate limit check failed, falling back to database",
			"error", err.Error(),
			"user_id", userID)
		return s.CheckDailyRateLimit(userID)
	}

	count := incrCmd.Val()

	// Set expiration only if this is the first request
	if count == 1 {
		expireCmd.Val()
	}

	if count > int64(s.config.RateLimitPerDay) {
		slog.Warn("Rate limit exceeded via Redis",
			"user_id", userID,
			"count", count,
			"limit", s.config.RateLimitPerDay)
		return apperrors.RateLimitError(userID, s.config.RateLimitPerDay)
	}

	slog.Debug("Rate limit check passed via Redis",
		"user_id", userID,
		"count", count,
		"limit", s.config.RateLimitPerDay)

	return nil
}

// CreateBar creates a new donation bar
func (s *BarService) CreateBar(userID string, req *models.CreateBarRequest) (*models.DonationBar, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeouts.DatabaseWrite)
	defer cancel()

	// Check daily rate limit (enhanced with Redis support)
	if err := s.checkRateLimitRedis(userID); err != nil {
		return nil, err
	}

	// Check user's total bar count
	count, err := s.GetUserBarCount(userID)
	if err != nil {
		return nil, apperrors.DatabaseError("count user bars", err)
	}

	if count >= int64(s.config.MaxBarsPerUser) {
		return nil, apperrors.MaxBarsReached(userID, count, int64(s.config.MaxBarsPerUser))
	}

	// Validate injections
	if !s.validateInjections(req.HTML) {
		return nil, apperrors.ValidationError("injection fields", "one or more required injection fields are missing")
	}

	// Create new bar
	bar := &models.DonationBar{
		ID:                 primitive.NewObjectID(),
		UserID:             userID,
		Name:               req.Name,
		Description:        req.Description,
		HTML:               req.HTML,
		CSS:                req.CSS,
		Language:           req.Language,
		Theme:              req.Theme,
		IsActive:           true,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		InitialAmount:      req.InitialAmount,
		GoalAmount:         req.GoalAmount,
		AIGenerated:        false,
		HasValidInjections: s.validateInjections(req.HTML),
	}

	err = s.repo.Insert(ctx, bar)
	if err != nil {
		return nil, apperrors.DatabaseError("insert bar", err)
	}

	slog.Info("Bar created successfully",
		"user_id", userID,
		"bar_id", bar.ID.Hex(),
		"name", bar.Name)

	return bar, nil
}

// CreateBarFromAI creates a bar from AI generation
func (s *BarService) CreateBarFromAI(userID, prompt string, aiResponse *models.AIGenerateResponse, initialAmount, goalAmount float64) (*models.DonationBar, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeouts.DatabaseWrite)
	defer cancel()

	// Check daily rate limit (enhanced with Redis support)
	if err := s.checkRateLimitRedis(userID); err != nil {
		return nil, err
	}

	// Check user's total bar count
	count, err := s.GetUserBarCount(userID)
	if err != nil {
		return nil, apperrors.DatabaseError("count user bars", err)
	}

	if count >= int64(s.config.MaxBarsPerUser) {
		return nil, apperrors.MaxBarsReached(userID, count, int64(s.config.MaxBarsPerUser))
	}

	// Generate name from prompt (first 50 chars)
	name := prompt
	if len(name) > 50 {
		name = name[:47] + "..."
	}

	bar := &models.DonationBar{
		ID:                 primitive.NewObjectID(),
		UserID:             userID,
		Name:               name,
		Description:        "AI tarafından oluşturulan donation bar",
		HTML:               aiResponse.HTML,
		CSS:                aiResponse.CSS,
		Language:           aiResponse.Metadata.Language,
		Theme:              aiResponse.Metadata.Theme,
		IsActive:           true,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		InitialAmount:      initialAmount,
		GoalAmount:         goalAmount,
		Prompt:             prompt,
		AIGenerated:        true,
		HasValidInjections: aiResponse.Metadata.HasInjections,
	}

	err = s.repo.Insert(ctx, bar)
	if err != nil {
		return nil, apperrors.DatabaseError("insert AI bar", err)
	}

	slog.Info("AI bar created successfully",
		"user_id", userID,
		"bar_id", bar.ID.Hex(),
		"prompt", prompt[:min(len(prompt), 50)])

	return bar, nil
}

// GetUserBars returns all bars for a user
func (s *BarService) GetUserBars(userID string) ([]*models.DonationBar, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeouts.DatabaseRead)
	defer cancel()

	bars, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.DatabaseError("find user bars", err)
	}

	return bars, nil
}

// GetBar returns a specific bar by ID
func (s *BarService) GetBar(userID, barID string) (*models.DonationBar, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeouts.DatabaseRead)
	defer cancel()

	bar, err := s.repo.FindByID(ctx, userID, barID)
	if err != nil {
		if err.Error() == "bar not found" {
			return nil, apperrors.NotFound("bar", barID)
		}
		if err.Error() == "invalid bar ID format" {
			return nil, apperrors.InvalidInput("bar ID", barID)
		}
		return nil, apperrors.DatabaseError("find bar", err)
	}

	return bar, nil
}

// UpdateBar updates a bar
func (s *BarService) UpdateBar(userID, barID string, req *models.UpdateBarRequest) (*models.DonationBar, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeouts.DatabaseWrite)
	defer cancel()

	bar, err := s.repo.Update(ctx, userID, barID, req)
	if err != nil {
		if err.Error() == "bar not found" {
			return nil, apperrors.NotFound("bar", barID)
		}
		if err.Error() == "invalid bar ID format" {
			return nil, apperrors.InvalidInput("bar ID", barID)
		}
		return nil, apperrors.DatabaseError("update bar", err)
	}

	return bar, nil
}

// UpdateBarComplete updates all fields of a bar including HTML/CSS
func (s *BarService) UpdateBarComplete(userID, barID string, req *models.CreateBarRequest, isActive bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeouts.DatabaseWrite)
	defer cancel()

	// Validate injections before update
	if !s.validateInjections(req.HTML) {
		return apperrors.ValidationError("injection fields", "one or more required injection fields are missing")
	}

	err := s.repo.UpdateComplete(ctx, userID, barID, req, isActive)
	if err != nil {
		if err.Error() == "bar not found" {
			return apperrors.NotFound("bar", barID)
		}
		if err.Error() == "invalid bar ID format" {
			return apperrors.InvalidInput("bar ID", barID)
		}
		return apperrors.DatabaseError("update complete bar", err)
	}

	return nil
}

// DeleteBar deletes a bar
func (s *BarService) DeleteBar(userID, barID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeouts.DatabaseWrite)
	defer cancel()

	err := s.repo.Delete(ctx, userID, barID)
	if err != nil {
		if err.Error() == "bar not found" {
			return apperrors.NotFound("bar", barID)
		}
		if err.Error() == "invalid bar ID format" {
			return apperrors.InvalidInput("bar ID", barID)
		}
		return apperrors.DatabaseError("delete bar", err)
	}

	return nil
}

// GetUserBarCount returns the total number of bars for a user
func (s *BarService) GetUserBarCount(userID string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeouts.DatabaseRead)
	defer cancel()

	count, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		return 0, apperrors.DatabaseError("count user bars", err)
	}

	return count, nil
}

// GetUserDailyBarCount returns the number of bars created by user today
func (s *BarService) GetUserDailyBarCount(userID string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeouts.DatabaseRead)
	defer cancel()

	count, err := s.repo.CountByUserIDToday(ctx, userID)
	if err != nil {
		return 0, apperrors.DatabaseError("count daily bars", err)
	}

	return count, nil
}

// CheckDailyRateLimit checks if user has exceeded daily rate limit
func (s *BarService) CheckDailyRateLimit(userID string) error {
	dailyCount, err := s.GetUserDailyBarCount(userID)
	if err != nil {
		return err
	}

	// cursorrules.rules: günlük maksimum 5 bar
	if dailyCount >= 5 {
		return apperrors.RateLimitError(userID, 5)
	}

	return nil
}

// validateInjections checks if all required injection fields are present
func (s *BarService) validateInjections(html string) bool {
	for _, injection := range models.RequiredInjections {
		if !strings.Contains(html, injection) {
			return false
		}
	}
	return true
}
