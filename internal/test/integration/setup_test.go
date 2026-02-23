// Package integration contains end-to-end integration tests.
// Tests spin up a real PostgreSQL container via testcontainers-go,
// apply migrations inline, start the full fx app (identical to production),
// and exercise HTTP endpoints via fiber.Test (in-process, no open port needed).
package integration

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"golang_boilerplate_module/internal/bootstrap"
	"golang_boilerplate_module/internal/shared/domain/providers"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

var (
	fiberApp *fiber.App
	dbURL    string
)

// TestMain boots one PostgreSQL container shared across all tests in this package.
func TestMain(m *testing.M) {
	ctx := context.Background()

	// ── PostgreSQL container ───────────────────────────────────────────────────
	pgContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("boilerplate"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		panic("start postgres: " + err.Error())
	}
	defer func() { _ = pgContainer.Terminate(ctx) }()

	dbURL, err = pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("postgres connection string: " + err.Error())
	}

	// ── Inline migrations ──────────────────────────────────────────────────────
	if err := applyMigrations(dbURL); err != nil {
		panic("migrations: " + err.Error())
	}

	// ── Configure env for config.NewConfig ────────────────────────────────────
	os.Setenv("DATABASE_URL", dbURL)
	os.Setenv("SERVICE_NAME", "boilerplate-api-test")
	os.Setenv("PORT", "3000")
	os.Setenv("APP_ENV", "test")
	os.Setenv("LOG_LEVEL", "error")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "") // disable OTEL in tests

	// ── Build fx app and extract the *fiber.App ────────────────────────────────
	// fxtest.New wires everything identically to production.
	// We capture fiberApp via fx.Invoke so we can call fiber.Test in each test.
	app := fxtest.New(
		&testing.T{},
		bootstrap.App,
		fx.Invoke(func(app *fiber.App, _ providers.LoggerProvider) {
			fiberApp = app
		}),
	)
	app.RequireStart()
	defer app.RequireStop()

	os.Exit(m.Run())
}

// applyMigrations runs all schema SQL inline (avoids requiring Docker-in-Docker for Flyway).
func applyMigrations(url string) error {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id         SERIAL PRIMARY KEY,
			name       VARCHAR(255) NOT NULL,
			email      VARCHAR(255) NOT NULL UNIQUE,
			created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

// truncateUsers resets the users table between tests for isolation.
func truncateUsers(t *testing.T) {
	t.Helper()
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("truncate open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// request runs an HTTP request against the in-process Fiber app.
func request(req *http.Request) (*http.Response, error) {
	return fiberApp.Test(req, 10_000)
}
