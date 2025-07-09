package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"donationbars/internal/config"
	"donationbars/internal/handlers"
	"donationbars/internal/interfaces"
	"donationbars/internal/repository"
	"donationbars/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Initialize structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting Donation Bars application", "version", "1.0.0")

	// Load .env file
	if err := godotenv.Load(); err != nil {
		slog.Warn("Environment file not found", "error", err.Error())
	}

	// Load configuration
	cfg := config.Load()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		slog.Error("Configuration validation failed", "error", err.Error())
		os.Exit(1)
	}

	slog.Info("Configuration loaded successfully",
		"port", cfg.Port,
		"db_name", cfg.DBName,
		"max_bars_per_user", cfg.MaxBarsPerUser,
		"rate_limit_per_day", cfg.RateLimitPerDay,
		"redis_enabled", cfg.Redis.Enabled)

	// Initialize database with improved connection handling
	db, err := config.InitDB(cfg.MongoURI, cfg.Timeouts)
	if err != nil {
		slog.Error("Critical: Failed to connect to database",
			"error", err.Error(),
			"mongo_uri", cfg.MongoURI)
		slog.Error("Application cannot start without database connection")
		os.Exit(1)
	}
	slog.Info("Database initialized successfully")

	// Ensure database cleanup on exit
	defer func() {
		if db != nil {
			if err := db.Disconnect(); err != nil {
				slog.Error("Failed to disconnect from database", "error", err.Error())
			} else {
				slog.Info("Database connection closed")
			}
		}
	}()

	// Initialize Redis connection
	redisClient, err := config.InitRedis(cfg.Redis, cfg.Timeouts.RedisOperation)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err.Error())
		slog.Warn("Continuing with database-based rate limiting")
		redisClient = &config.RedisClient{Enabled: false}
	}

	// Ensure Redis cleanup on exit
	defer func() {
		if redisClient != nil && redisClient.IsEnabled() {
			if err := redisClient.Close(); err != nil {
				slog.Error("Failed to close Redis connection", "error", err.Error())
			} else {
				slog.Info("Redis connection closed")
			}
		}
	}()

	// Initialize dependencies with interface-based dependency injection
	var barRepo interfaces.BarRepositoryInterface
	var barService interfaces.BarServiceInterface
	var aiService interfaces.AIServiceInterface

	// Initialize repository
	barRepo = repository.NewBarRepository(db, cfg.Timeouts)
	slog.Info("Repository initialized")

	// Initialize services with dependency injection
	barService = services.NewBarService(barRepo, redisClient, cfg)
	aiService = services.NewAIService(cfg.OpenAIKey, cfg.Timeouts.AI)
	slog.Info("Services initialized",
		"redis_rate_limiting", redisClient.IsEnabled(),
		"ai_service_ready", cfg.OpenAIKey != "")

	// Initialize handlers with service interfaces
	h := handlers.New(barService, aiService)
	slog.Info("Handlers initialized")

	// Setup router
	r := gin.Default()

	// Load HTML templates
	r.LoadHTMLGlob("templates/*.html")

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	})

	// Request logging middleware
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		slog.Info("HTTP Request",
			"method", param.Method,
			"path", param.Path,
			"status", param.StatusCode,
			"latency", param.Latency,
			"ip", param.ClientIP,
			"user_agent", param.Request.UserAgent())
		return ""
	}))

	// API routes
	api := r.Group("/api/v1")
	{
		api.POST("/bars", h.CreateBar)
		api.GET("/bars", h.GetUserBars)
		api.GET("/bars/:id", h.GetBar)
		api.PUT("/bars/:id", h.UpdateBar)
		api.DELETE("/bars/:id", h.DeleteBar)
		api.POST("/bars/generate", h.GenerateBarWithAI)
	}

	// Web routes (Server-Side Rendering)
	r.GET("/", h.HomePage)
	r.GET("/create", h.CreatePage)
	r.POST("/create", h.CreateBarForm)
	r.POST("/create/ai", h.CreateBarAIForm)
	r.POST("/create/ai/save", h.SaveAIBarForm)
	r.GET("/edit/:id", h.EditPage)
	r.POST("/edit/:id", h.EditBarForm)
	r.GET("/manage", h.ManagePage)
	r.POST("/manage/:id/toggle", h.ToggleBarStatus)
	r.POST("/manage/:id/delete", h.DeleteBarForm)
	r.GET("/preview/:id", h.PreviewBar)

	// Static files (CSS only, no JS)
	r.Static("/static", "./static")

	// Enhanced health check
	r.GET("/health", func(c *gin.Context) {
		status := "healthy"
		checks := map[string]interface{}{
			"database": map[string]interface{}{
				"connected": db != nil,
				"status":    "ok",
			},
			"redis": map[string]interface{}{
				"enabled":   redisClient.IsEnabled(),
				"connected": redisClient.IsEnabled(),
				"status": func() string {
					if redisClient.IsEnabled() {
						return "ok"
					}
					return "disabled"
				}(),
			},
			"openai": map[string]interface{}{
				"configured": cfg.OpenAIKey != "" && cfg.OpenAIKey != "your_openai_api_key_here",
				"status":     "ok",
			},
		}

		// Check if any critical service is down
		if db == nil {
			status = "unhealthy"
			checks["database"].(map[string]interface{})["status"] = "error"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    status,
			"checks":    checks,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   "1.0.0",
		})
	})

	slog.Info("Server starting", "port", cfg.Port)

	// Create HTTP server with configured timeouts
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err.Error())
			os.Exit(1)
		}
	}()

	slog.Info("Server started successfully",
		"port", cfg.Port,
		"pid", os.Getpid(),
		"endpoints", []string{"/", "/api/v1", "/health"})

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	// Give outstanding requests configured timeout to complete
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeouts.ServerShutdown)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err.Error())
		os.Exit(1)
	}

	slog.Info("Server shutdown complete")
}
