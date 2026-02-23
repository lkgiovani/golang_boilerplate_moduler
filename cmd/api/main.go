package main

import (
	"golang_boilerplate_module/internal/bootstrap"

	"github.com/joho/godotenv"
	"go.uber.org/fx"
)

func main() {
	// Load .env file if present (development convenience).
	// In production, environment variables are expected to be set externally.
	_ = godotenv.Load()

	app := fx.New(bootstrap.App)
	app.Run()
}
