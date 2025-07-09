package config

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TimeoutConfig holds all timeout configurations
type TimeoutConfig struct {
	DatabaseRead   time.Duration
	DatabaseWrite  time.Duration
	AI             time.Duration
	ServerShutdown time.Duration
	RedisOperation time.Duration
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	Enabled  bool
}

type Config struct {
	// Database
	MongoURI string
	DBName   string

	// External services
	OpenAIKey string
	Redis     RedisConfig

	// Server
	Port string

	// Business rules
	MaxBarsPerUser  int
	RateLimitPerDay int

	// Timeouts
	Timeouts TimeoutConfig
}

type Database struct {
	Client *mongo.Client
	DB     *mongo.Database
}

func Load() *Config {
	return &Config{
		MongoURI:        getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:          getEnv("DB_NAME", "donationbars"),
		OpenAIKey:       getEnv("OPENAI_API_KEY", ""),
		Port:            getEnv("PORT", "8080"),
		MaxBarsPerUser:  getEnvInt("MAX_BARS_PER_USER", 5),
		RateLimitPerDay: getEnvInt("RATE_LIMIT_PER_DAY", 5),

		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			Enabled:  getEnvBool("REDIS_ENABLED", false),
		},

		Timeouts: TimeoutConfig{
			DatabaseRead:   getEnvDuration("DB_READ_TIMEOUT", 5*time.Second),
			DatabaseWrite:  getEnvDuration("DB_WRITE_TIMEOUT", 10*time.Second),
			AI:             getEnvDuration("AI_TIMEOUT", 30*time.Second),
			ServerShutdown: getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 5*time.Second),
			RedisOperation: getEnvDuration("REDIS_TIMEOUT", 2*time.Second),
		},
	}
}

// Validate checks if all required configuration values are present
func (c *Config) Validate() error {
	if c.OpenAIKey == "" || c.OpenAIKey == "your_openai_api_key_here" {
		return errors.New("OpenAI API key is required and cannot be placeholder value")
	}

	if c.MongoURI == "" {
		return errors.New("MongoDB URI is required")
	}

	if c.DBName == "" {
		return errors.New("Database name is required")
	}

	if c.Port == "" {
		return errors.New("Port is required")
	}

	if c.MaxBarsPerUser <= 0 {
		return errors.New("MaxBarsPerUser must be positive")
	}

	if c.RateLimitPerDay <= 0 {
		return errors.New("RateLimitPerDay must be positive")
	}

	// Timeout validations
	if c.Timeouts.DatabaseRead <= 0 {
		return errors.New("DatabaseRead timeout must be positive")
	}

	if c.Timeouts.DatabaseWrite <= 0 {
		return errors.New("DatabaseWrite timeout must be positive")
	}

	if c.Timeouts.AI <= 0 {
		return errors.New("AI timeout must be positive")
	}

	return nil
}

func InitDB(mongoURI string, timeoutConfig TimeoutConfig) (*Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutConfig.DatabaseWrite)
	defer cancel()

	// Configure connection options
	clientOptions := options.Client().
		ApplyURI(mongoURI).
		SetMaxPoolSize(10).
		SetMinPoolSize(2).
		SetMaxConnIdleTime(30 * time.Second).
		SetServerSelectionTimeout(5 * time.Second).
		SetConnectTimeout(10 * time.Second)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Test connection with ping
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()

	err = client.Ping(pingCtx, nil)
	if err != nil {
		client.Disconnect(ctx)
		return nil, err
	}

	cfg := Load()
	db := client.Database(cfg.DBName)

	slog.Info("Successfully connected to MongoDB",
		"database", cfg.DBName,
		"max_pool_size", 10,
		"min_pool_size", 2)

	return &Database{
		Client: client,
		DB:     db,
	}, nil
}

func (d *Database) Disconnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return d.Client.Disconnect(ctx)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
