package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DonationBar represents a donation bar template
type DonationBar struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID      string             `bson:"user_id" json:"user_id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	HTML        string             `bson:"html" json:"html"`
	CSS         string             `bson:"css" json:"css"`
	Language    string             `bson:"language" json:"language"` // "tr" or "en"
	Theme       string             `bson:"theme" json:"theme"`
	IsActive    bool               `bson:"is_active" json:"is_active"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`

	// Donation amounts
	InitialAmount float64 `bson:"initial_amount" json:"initial_amount"` // Starting amount (current total)
	GoalAmount    float64 `bson:"goal_amount" json:"goal_amount"`       // Target amount

	// AI generation metadata
	Prompt      string `bson:"prompt" json:"prompt"`
	AIGenerated bool   `bson:"ai_generated" json:"ai_generated"`

	// Injection validation
	HasValidInjections bool `bson:"has_valid_injections" json:"has_valid_injections"`
}

// CreateBarRequest represents the request to create a new bar
type CreateBarRequest struct {
	Name          string  `json:"name" binding:"required,min=1,max=100"`
	Description   string  `json:"description" binding:"max=500"`
	HTML          string  `json:"html" binding:"required"`
	CSS           string  `json:"css" binding:"required"`
	Language      string  `json:"language" binding:"required,oneof=tr en"`
	Theme         string  `json:"theme" binding:"max=50"`
	InitialAmount float64 `json:"initial_amount" binding:"gte=0"`
	GoalAmount    float64 `json:"goal_amount" binding:"gt=0"`
}

// GenerateBarRequest represents the request for AI bar generation
type GenerateBarRequest struct {
	Prompt        string  `json:"prompt" form:"prompt" binding:"required,min=10,max=1000"`
	Language      string  `json:"language" form:"language" binding:"required,oneof=tr en"`
	Theme         string  `json:"theme" form:"theme" binding:"max=50"`
	InitialAmount float64 `json:"initial_amount" form:"initial_amount" binding:"gte=0"`
	GoalAmount    float64 `json:"goal_amount" form:"goal_amount" binding:"gt=0"`
}

// UpdateBarRequest represents the request to update a bar
type UpdateBarRequest struct {
	Name          *string  `json:"name,omitempty"`
	Description   *string  `json:"description,omitempty"`
	IsActive      *bool    `json:"is_active,omitempty"`
	InitialAmount *float64 `json:"initial_amount,omitempty"`
	GoalAmount    *float64 `json:"goal_amount,omitempty"`
}

// AIGenerateResponse represents the AI service response
type AIGenerateResponse struct {
	HTML     string             `json:"html"`
	CSS      string             `json:"css"`
	Metadata AIGenerateMetadata `json:"metadata"`
}

type AIGenerateMetadata struct {
	Language      string `json:"language"`
	Theme         string `json:"theme"`
	HasInjections bool   `json:"injection"`
}

// Required injection fields that must be present
var RequiredInjections = []string{
	"{goal}",
	"{total}",
	"{percentage}",
	"{remaining}",
	"{description}",
}
