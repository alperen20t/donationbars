package services

import (
	"testing"
	"time"

	"donationbars/internal/models"
)

func TestNewAIService_WithValidKey(t *testing.T) {
	timeout := 30 * time.Second
	service := NewAIService("valid-api-key", timeout)

	if service == nil {
		t.Error("Expected service to be created, got nil")
	}
}

func TestNewAIService_WithEmptyKey(t *testing.T) {
	timeout := 30 * time.Second
	service := NewAIService("", timeout)

	if service == nil {
		t.Error("Expected service to be created even with empty key, got nil")
	}
}

func TestNewAIService_WithPlaceholderKey(t *testing.T) {
	timeout := 30 * time.Second
	service := NewAIService("your_openai_api_key_here", timeout)

	if service == nil {
		t.Error("Expected service to be created even with placeholder key, got nil")
	}
}

func TestAIService_GenerateBar_WithNilClient(t *testing.T) {
	timeout := 30 * time.Second
	service := NewAIService("", timeout) // This will create service with nil client

	req := &models.GenerateBarRequest{
		Prompt:        "Create a modern donation bar",
		Language:      "tr",
		Theme:         "modern",
		InitialAmount: 100.0,
		GoalAmount:    1000.0,
	}

	result, err := service.GenerateBar(req)

	if err == nil {
		t.Error("Expected error with nil client, got nil")
	}

	if result != nil {
		t.Error("Expected nil result with nil client, got result")
	}
}

func TestAIService_ValidateInjections_ValidHTML(t *testing.T) {
	service := &AIService{client: nil}

	validHTML := `<div>
		<span>{goal}</span>
		<span>{total}</span>
		<span>{percentage}</span>
		<span>{remaining}</span>
		<span>{description}</span>
	</div>`

	if !service.validateInjections(validHTML) {
		t.Error("Expected HTML with all injections to be valid")
	}
}

func TestAIService_ValidateInjections_MissingInjections(t *testing.T) {
	service := &AIService{client: nil}

	invalidHTML := `<div>
		<span>{goal}</span>
		<span>{total}</span>
		<!-- Missing {percentage}, {remaining}, {description} -->
	</div>`

	if service.validateInjections(invalidHTML) {
		t.Error("Expected HTML with missing injections to be invalid")
	}
}

func TestAIService_ValidateCSSSizeConstraints(t *testing.T) {
	service := &AIService{client: nil}

	validCSS := `
	.donation-bar {
		max-width: 800px;
		max-height: 200px;
		width: 600px;
		height: 100px;
	}
	`

	if !service.validateCSSSizeConstraintsStrict(validCSS) {
		t.Error("Expected CSS with valid size constraints to be valid")
	}
}

func TestAIService_ValidateCSSSizeConstraints_InvalidSize(t *testing.T) {
	service := &AIService{client: nil}

	invalidCSS := `
	.donation-bar {
		width: 900px; /* Too wide */
		height: 250px; /* Too tall */
	}
	`

	if service.validateCSSSizeConstraintsStrict(invalidCSS) {
		t.Error("Expected CSS with invalid size constraints to be invalid")
	}
}

func TestAIService_CleanAndValidateHTML(t *testing.T) {
	service := &AIService{client: nil}

	dirtyHTML := `<div onclick="alert('xss')" style="background: url('http://evil.com')">
		<script>alert('xss')</script>
		Valid content {goal}
	</div>`

	cleaned := service.cleanAndValidateHTML(dirtyHTML)

	// Should remove dangerous attributes and scripts
	if cleaned == dirtyHTML {
		t.Error("Expected HTML to be cleaned, but it remained unchanged")
	}

	// Should preserve valid injection
	if !service.validateInjections(cleaned) {
		t.Error("Expected cleaned HTML to preserve injection fields")
	}
}

func TestAIService_CleanAndValidateCSS(t *testing.T) {
	service := &AIService{client: nil}

	dirtyCSS := `
	.bar {
		background: url('http://evil.com');
		behavior: url('evil.htc');
		-moz-binding: url('evil.xml');
		width: 800px;
	}
	`

	cleaned := service.cleanAndValidateCSS(dirtyCSS)

	// Should remove dangerous properties
	if cleaned == dirtyCSS {
		t.Error("Expected CSS to be cleaned, but it remained unchanged")
	}
}
