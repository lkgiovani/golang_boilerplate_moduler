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

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Logger   LoggerConfig
	Otel     OtelConfig
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

	return &Config{
		App: AppConfig{
			ServiceName: getEnvOrDefault("SERVICE_NAME", "boilerplate-api"),
			Port:        port,
			Env:         getEnvOrDefault("APP_ENV", "production"),
			Version:     "0.1.0",
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
	}, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
