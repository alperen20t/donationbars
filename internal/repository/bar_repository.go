package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"donationbars/internal/config"
	"donationbars/internal/interfaces"
	"donationbars/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type BarRepository struct {
	db         *config.Database
	collection *mongo.Collection
	timeouts   config.TimeoutConfig
}

// NewBarRepository creates a new bar repository
func NewBarRepository(db *config.Database, timeouts config.TimeoutConfig) interfaces.BarRepositoryInterface {
	repo := &BarRepository{
		db:       db,
		timeouts: timeouts,
	}
	if db != nil && db.DB != nil {
		repo.collection = db.DB.Collection("donation_bars")
	}
	return repo
}

// Insert adds a new bar to the database
func (r *BarRepository) Insert(ctx context.Context, bar *models.DonationBar) error {
	if r.collection == nil {
		return errors.New("database connection not available")
	}

	// Use configured timeout for write operations
	writeCtx, cancel := context.WithTimeout(ctx, r.timeouts.DatabaseWrite)
	defer cancel()

	_, err := r.collection.InsertOne(writeCtx, bar)
	return err
}

// FindByUserID returns all bars for a user
func (r *BarRepository) FindByUserID(ctx context.Context, userID string) ([]*models.DonationBar, error) {
	if r.collection == nil {
		return []*models.DonationBar{}, nil
	}

	// Use configured timeout for read operations
	readCtx, cancel := context.WithTimeout(ctx, r.timeouts.DatabaseRead)
	defer cancel()

	filter := bson.M{"user_id": userID}
	cursor, err := r.collection.Find(readCtx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var bars []*models.DonationBar
	if err = cursor.All(readCtx, &bars); err != nil {
		return nil, err
	}

	return bars, nil
}

// FindByID returns a specific bar by ID for a user
func (r *BarRepository) FindByID(ctx context.Context, userID, barID string) (*models.DonationBar, error) {
	if r.collection == nil {
		return nil, errors.New("database connection not available")
	}

	objectID, err := primitive.ObjectIDFromHex(barID)
	if err != nil {
		return nil, errors.New("invalid bar ID format")
	}

	// Use configured timeout for read operations
	readCtx, cancel := context.WithTimeout(ctx, r.timeouts.DatabaseRead)
	defer cancel()

	filter := bson.M{
		"_id":     objectID,
		"user_id": userID,
	}

	var bar models.DonationBar
	err = r.collection.FindOne(readCtx, filter).Decode(&bar)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("bar not found")
		}
		return nil, err
	}

	return &bar, nil
}

// Update updates basic fields of a bar
func (r *BarRepository) Update(ctx context.Context, userID, barID string, req *models.UpdateBarRequest) (*models.DonationBar, error) {
	if r.collection == nil {
		return nil, errors.New("database connection not available")
	}

	objectID, err := primitive.ObjectIDFromHex(barID)
	if err != nil {
		return nil, errors.New("invalid bar ID format")
	}

	// Build update document
	update := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	if req.Name != nil {
		update["$set"].(bson.M)["name"] = *req.Name
	}
	if req.Description != nil {
		update["$set"].(bson.M)["description"] = *req.Description
	}
	if req.IsActive != nil {
		update["$set"].(bson.M)["is_active"] = *req.IsActive
	}
	if req.InitialAmount != nil {
		update["$set"].(bson.M)["initial_amount"] = *req.InitialAmount
	}
	if req.GoalAmount != nil {
		update["$set"].(bson.M)["goal_amount"] = *req.GoalAmount
	}

	filter := bson.M{
		"_id":     objectID,
		"user_id": userID,
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, errors.New("bar not found")
	}

	// Return updated bar
	return r.FindByID(ctx, userID, barID)
}

// UpdateComplete updates all fields of a bar including HTML/CSS
func (r *BarRepository) UpdateComplete(ctx context.Context, userID, barID string, req *models.CreateBarRequest, isActive bool) error {
	if r.collection == nil {
		return errors.New("database connection not available")
	}

	objectID, err := primitive.ObjectIDFromHex(barID)
	if err != nil {
		return errors.New("invalid bar ID format")
	}

	// Validate injections
	hasValidInjections := r.validateInjections(req.HTML)

	// Build update document
	update := bson.M{
		"$set": bson.M{
			"name":                 req.Name,
			"description":          req.Description,
			"html":                 req.HTML,
			"css":                  req.CSS,
			"language":             req.Language,
			"theme":                req.Theme,
			"is_active":            isActive,
			"initial_amount":       req.InitialAmount,
			"goal_amount":          req.GoalAmount,
			"updated_at":           time.Now(),
			"has_valid_injections": hasValidInjections,
		},
	}

	filter := bson.M{
		"_id":     objectID,
		"user_id": userID,
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("bar not found")
	}

	return nil
}

// Delete removes a bar from the database
func (r *BarRepository) Delete(ctx context.Context, userID, barID string) error {
	if r.collection == nil {
		return errors.New("database connection not available")
	}

	objectID, err := primitive.ObjectIDFromHex(barID)
	if err != nil {
		return errors.New("invalid bar ID format")
	}

	filter := bson.M{
		"_id":     objectID,
		"user_id": userID,
	}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("bar not found")
	}

	return nil
}

// CountByUserID returns the total number of bars for a user
func (r *BarRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	if r.collection == nil {
		return 0, errors.New("database connection not available")
	}

	filter := bson.M{"user_id": userID}
	return r.collection.CountDocuments(ctx, filter)
}

// CountByUserIDToday returns the number of bars created by user today
func (r *BarRepository) CountByUserIDToday(ctx context.Context, userID string) (int64, error) {
	if r.collection == nil {
		return 0, errors.New("database connection not available")
	}

	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	filter := bson.M{
		"user_id": userID,
		"created_at": bson.M{
			"$gte": today,
			"$lt":  tomorrow,
		},
	}

	return r.collection.CountDocuments(ctx, filter)
}

// validateInjections checks if all required injection fields are present
func (r *BarRepository) validateInjections(html string) bool {
	for _, injection := range models.RequiredInjections {
		if !strings.Contains(html, injection) {
			return false
		}
	}
	return true
}
