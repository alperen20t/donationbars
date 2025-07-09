package errors

import (
	"errors"
	"fmt"
)

// Error types for better error handling
var (
	ErrNotFound             = errors.New("resource not found")
	ErrInvalidInput         = errors.New("invalid input")
	ErrDatabaseUnavailable  = errors.New("database connection not available")
	ErrRateLimitExceeded    = errors.New("rate limit exceeded")
	ErrMaxBarsReached       = errors.New("maximum bars limit reached")
	ErrInvalidBarID         = errors.New("invalid bar ID format")
	ErrAIServiceUnavailable = errors.New("AI service unavailable")
	ErrValidationFailed     = errors.New("validation failed")
)

// AppError represents an application error with context
type AppError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Error constructors
func NotFound(resource string, id string) *AppError {
	return &AppError{
		Type:    "NOT_FOUND",
		Message: fmt.Sprintf("%s not found", resource),
		Details: fmt.Sprintf("ID: %s", id),
		Err:     ErrNotFound,
	}
}

func InvalidInput(field string, value string) *AppError {
	return &AppError{
		Type:    "INVALID_INPUT",
		Message: fmt.Sprintf("invalid %s", field),
		Details: fmt.Sprintf("value: %s", value),
		Err:     ErrInvalidInput,
	}
}

func DatabaseError(operation string, err error) *AppError {
	return &AppError{
		Type:    "DATABASE_ERROR",
		Message: fmt.Sprintf("database %s failed", operation),
		Err:     fmt.Errorf("database operation failed: %w", err),
	}
}

func MaxBarsReached(userID string, current, max int64) *AppError {
	return &AppError{
		Type:    "MAX_BARS_REACHED",
		Message: fmt.Sprintf("maksimum bar sayısına ulaşıldı (%d/%d)", current, max),
		Details: fmt.Sprintf("user_id: %s", userID),
		Err:     ErrMaxBarsReached,
	}
}

func ValidationError(field string, message string) *AppError {
	return &AppError{
		Type:    "VALIDATION_ERROR",
		Message: fmt.Sprintf("validation failed for %s: %s", field, message),
		Err:     ErrValidationFailed,
	}
}

func RateLimitError(userID string, limit int) *AppError {
	return &AppError{
		Type:    "RATE_LIMIT_EXCEEDED",
		Message: fmt.Sprintf("günlük maksimum bar oluşturma sınırına ulaşıldı (%d bar/gün)", limit),
		Details: fmt.Sprintf("user_id: %s", userID),
		Err:     ErrRateLimitExceeded,
	}
}

func AIServiceError(operation string, err error) *AppError {
	return &AppError{
		Type:    "AI_SERVICE_ERROR",
		Message: fmt.Sprintf("AI %s failed", operation),
		Err:     fmt.Errorf("AI service error: %w", err),
	}
}
