package config

import (
	"fmt"
	"os"
	"strconv"
)

type AppConfig struct {
	ServiceName string
	Port        int
	Env         string
	Version     string
	BaseURL     string
}

type DatabaseConfig struct {
	URL            string
	MaxConnections int
}

type LoggerConfig struct {
	Level string
}

type OtelConfig struct {
	Endpoint string
	Protocol string
}

type RedisConfig struct {
	URL string
}

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromAddress  string
	FromName     string
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessExpiry  int // minutes
	RefreshExpiry int // days
	Issuer        string
}

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Logger   LoggerConfig
	Otel     OtelConfig
	Redis    RedisConfig
	Email    EmailConfig
	JWT      JWTConfig
}

func NewConfig() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	port, err := strconv.Atoi(getEnvOrDefault("PORT", "3000"))
	if err != nil {
		return nil, fmt.Errorf("PORT must be a valid number: %w", err)
	}

	maxConn, err := strconv.Atoi(getEnvOrDefault("DATABASE_MAX_CONNECTIONS", "10"))
	if err != nil {
		return nil, fmt.Errorf("DATABASE_MAX_CONNECTIONS must be a valid number: %w", err)
	}

	smtpPort, err := strconv.Atoi(getEnvOrDefault("SMTP_PORT", "1025"))
	if err != nil {
		return nil, fmt.Errorf("SMTP_PORT must be a valid number: %w", err)
	}

	accessExpiry, err := strconv.Atoi(getEnvOrDefault("JWT_ACCESS_EXPIRY_MIN", "15"))
	if err != nil {
		return nil, fmt.Errorf("JWT_ACCESS_EXPIRY_MIN must be a valid number: %w", err)
	}

	refreshExpiry, err := strconv.Atoi(getEnvOrDefault("JWT_REFRESH_EXPIRY_DAYS", "7"))
	if err != nil {
		return nil, fmt.Errorf("JWT_REFRESH_EXPIRY_DAYS must be a valid number: %w", err)
	}

	return &Config{
		App: AppConfig{
			ServiceName: getEnvOrDefault("SERVICE_NAME", "boilerplate-api"),
			Port:        port,
			Env:         getEnvOrDefault("APP_ENV", "production"),
			Version:     "0.1.0",
			BaseURL:     getEnvOrDefault("APP_BASE_URL", "http://localhost:3000"),
		},
		Database: DatabaseConfig{
			URL:            dbURL,
			MaxConnections: maxConn,
		},
		Logger: LoggerConfig{
			Level: getEnvOrDefault("LOG_LEVEL", "error"),
		},
		Otel: OtelConfig{
			Endpoint: getEnvOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
			Protocol: getEnvOrDefault("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf"),
		},
		Redis: RedisConfig{
			URL: getEnvOrDefault("REDIS_URL", "redis://localhost:6379/0"),
		},
		Email: EmailConfig{
			SMTPHost:     getEnvOrDefault("SMTP_HOST", "localhost"),
			SMTPPort:     smtpPort,
			SMTPUser:     getEnvOrDefault("SMTP_USER", ""),
			SMTPPassword: getEnvOrDefault("SMTP_PASSWORD", ""),
			FromAddress:  getEnvOrDefault("EMAIL_FROM", "noreply@example.com"),
			FromName:     getEnvOrDefault("EMAIL_FROM_NAME", "SaaS Boilerplate"),
		},
		JWT: JWTConfig{
			AccessSecret:  getEnvOrDefault("JWT_ACCESS_SECRET", "change-me-in-production-32chars!"),
			RefreshSecret: getEnvOrDefault("JWT_REFRESH_SECRET", "change-me-refresh-production-32!"),
			AccessExpiry:  accessExpiry,
			RefreshExpiry: refreshExpiry,
			Issuer:        getEnvOrDefault("JWT_ISSUER", "saas-boilerplate"),
		},
	}, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
