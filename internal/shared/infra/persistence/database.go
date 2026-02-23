package persistence

import (
	"database/sql"
	"fmt"

	"golang_boilerplate_module/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/opentelemetry/tracing"
)

func NewDB(cfg *config.Config) (*gorm.DB, error) {
	gormLogger := logger.Default.LogMode(logger.Silent)

	db, err := gorm.Open(postgres.Open(cfg.Database.URL), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.Database.MaxConnections)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxConnections / 2)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	if err := db.Use(tracing.NewPlugin()); err != nil {
		return nil, fmt.Errorf("failed to register OTEL tracing plugin: %w", err)
	}

	return db, nil
}

func CloseDB(db *gorm.DB) error {
	var sqlDB *sql.DB
	var err error

	if sqlDB, err = db.DB(); err != nil {
		return err
	}

	return sqlDB.Close()
}