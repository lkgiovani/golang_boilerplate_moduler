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
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"golang_boilerplate_module/internal/bootstrap"
	paymentmock "golang_boilerplate_module/internal/modules/payments/infra/mock"
	"golang_boilerplate_module/internal/shared/domain/providers"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

var (
	fiberApp       *fiber.App
	dbURL          string
	cacheProvider  providers.CacheProvider
	emailProvider  providers.EmailProvider
	mailhogAPIAddr string
	mockGateway    *paymentmock.Gateway
)

func TestMain(m *testing.M) {
	ctx := context.Background()

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

	redisContainer, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		panic("start redis: " + err.Error())
	}
	defer func() { _ = redisContainer.Terminate(ctx) }()

	redisURL, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		panic("redis connection string: " + err.Error())
	}

	mailhogContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mailhog/mailhog:latest",
			ExposedPorts: []string{"1025/tcp", "8025/tcp"},
			WaitingFor:   wait.ForListeningPort("8025/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		panic("start mailhog: " + err.Error())
	}
	defer func() { _ = mailhogContainer.Terminate(ctx) }()

	mailhogSMTPHost, err := mailhogContainer.Host(ctx)
	if err != nil {
		panic("mailhog host: " + err.Error())
	}
	mailhogSMTPPort, err := mailhogContainer.MappedPort(ctx, "1025/tcp")
	if err != nil {
		panic("mailhog smtp port: " + err.Error())
	}
	mailhogHTTPPort, err := mailhogContainer.MappedPort(ctx, "8025/tcp")
	if err != nil {
		panic("mailhog http port: " + err.Error())
	}
	mailhogAPIAddr = fmt.Sprintf("%s:%s", mailhogSMTPHost, mailhogHTTPPort.Port())

	if err := applyMigrations(dbURL); err != nil {
		panic("migrations: " + err.Error())
	}

	os.Setenv("DATABASE_URL", dbURL)
	os.Setenv("REDIS_URL", redisURL)
	os.Setenv("SERVICE_NAME", "boilerplate-api-test")
	os.Setenv("PORT", "3000")
	os.Setenv("APP_ENV", "test")
	os.Setenv("LOG_LEVEL", "error")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	os.Setenv("SMTP_HOST", mailhogSMTPHost)
	os.Setenv("SMTP_PORT", mailhogSMTPPort.Port())
	os.Setenv("SMTP_USER", "")
	os.Setenv("SMTP_PASSWORD", "")
	os.Setenv("EMAIL_FROM", "noreply@example.com")
	os.Setenv("EMAIL_FROM_NAME", "SaaS Boilerplate")
	os.Setenv("JWT_ACCESS_SECRET", "test-access-secret-32-characters!")
	os.Setenv("JWT_REFRESH_SECRET", "test-refresh-secret-32-characters")
	os.Setenv("JWT_ACCESS_EXPIRY_MIN", "15")
	os.Setenv("JWT_REFRESH_EXPIRY_DAYS", "7")
	os.Setenv("JWT_ISSUER", "saas-boilerplate-test")
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_fake_for_integration_tests")
	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_fake_for_integration_tests")
	os.Setenv("PAYMENT_GATEWAY", "stripe")

	app := fxtest.New(
		&testing.T{},
		bootstrap.App,
		fx.Decorate(func() providers.PaymentGateway {
			mockGateway = paymentmock.New()
			return mockGateway
		}),
		fx.Invoke(func(a *fiber.App, _ providers.LoggerProvider, cp providers.CacheProvider, ep providers.EmailProvider) {
			fiberApp = a
			cacheProvider = cp
			emailProvider = ep
		}),
	)
	app.RequireStart()
	defer app.RequireStop()

	os.Exit(m.Run())
}

func applyMigrations(url string) error {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	// V1 + V8 additions: users (gateway_customer_id/gateway_name from V8)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id                  BIGSERIAL PRIMARY KEY,
			name                VARCHAR(255) NOT NULL,
			email               VARCHAR(255) NOT NULL UNIQUE,
			password            VARCHAR(255),
			img_url             VARCHAR(500),
			admin               BOOLEAN      NOT NULL DEFAULT FALSE,
			active              BOOLEAN      NOT NULL DEFAULT TRUE,
			email_verified      BOOLEAN      NOT NULL DEFAULT FALSE,
			source              VARCHAR(50)  NOT NULL DEFAULT 'LOCAL',
			metadata            JSONB,
			last_access         TIMESTAMP,
			gateway_customer_id VARCHAR(255),
			gateway_name        VARCHAR(32),
			created_at          TIMESTAMP    NOT NULL DEFAULT NOW(),
			updated_at          TIMESTAMP    DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create users: %w", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`)
	if err != nil {
		return fmt.Errorf("create idx_users_email: %w", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_active ON users(active)`)
	if err != nil {
		return fmt.Errorf("create idx_users_active: %w", err)
	}

	// V2: refresh_tokens
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS refresh_tokens (
			id           UUID PRIMARY KEY,
			user_id      BIGINT NOT NULL,
			user_email   VARCHAR(255) NOT NULL,
			device_id    VARCHAR(255) NOT NULL,
			jti          VARCHAR(255) NOT NULL UNIQUE,
			family_id    UUID NOT NULL,
			token_hash   VARCHAR(255) NOT NULL UNIQUE,
			expires_at   TIMESTAMP NOT NULL,
			created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
			used         BOOLEAN NOT NULL DEFAULT FALSE,
			used_at      TIMESTAMP,
			rotated_from UUID,
			revoked_at   TIMESTAMP,
			user_agent   VARCHAR(500),
			ip_address   VARCHAR(45),
			CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("create refresh_tokens: %w", err)
	}

	// V3: suspicious_activities + user_security_blocks
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS suspicious_activities (
			id            BIGSERIAL PRIMARY KEY,
			user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			activity_type VARCHAR(50) NOT NULL,
			endpoint      VARCHAR(500) NOT NULL,
			ip_address    VARCHAR(45),
			user_agent    TEXT,
			request_count INTEGER DEFAULT 1,
			details       JSONB,
			severity      VARCHAR(20) NOT NULL DEFAULT 'LOW',
			created_at    TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create suspicious_activities: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_security_blocks (
			id                        BIGSERIAL PRIMARY KEY,
			user_id                   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			reason                    VARCHAR(500) NOT NULL,
			suspicious_activity_count INTEGER NOT NULL DEFAULT 0,
			blocked_at                TIMESTAMP NOT NULL DEFAULT NOW(),
			blocked_until             TIMESTAMP,
			unblocked_at              TIMESTAMP,
			unblocked_by              BIGINT REFERENCES users(id) ON DELETE SET NULL,
			notes                     TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("create user_security_blocks: %w", err)
	}

	// V4: email_verification_tokens
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS email_verification_tokens (
			id          BIGSERIAL PRIMARY KEY,
			user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			email       VARCHAR(255) NOT NULL,
			token       VARCHAR(255) NOT NULL UNIQUE,
			expires_at  TIMESTAMPTZ NOT NULL,
			verified_at TIMESTAMPTZ,
			used        BOOLEAN NOT NULL DEFAULT FALSE,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("create email_verification_tokens: %w", err)
	}

	// V5: password_reset_tokens
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS password_reset_tokens (
			id         BIGSERIAL PRIMARY KEY,
			user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			email      VARCHAR(255) NOT NULL,
			token      VARCHAR(255) NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			used       BOOLEAN NOT NULL DEFAULT FALSE,
			used_at    TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("create password_reset_tokens: %w", err)
	}

	// V6 + V8: plans, subscriptions, payment_events with gateway_* columns
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS plans (
			id               BIGSERIAL PRIMARY KEY,
			name             VARCHAR(255) NOT NULL,
			slug             VARCHAR(255) NOT NULL UNIQUE,
			description      TEXT,
			price_cents      BIGINT NOT NULL DEFAULT 0,
			currency         VARCHAR(3) NOT NULL DEFAULT 'BRL',
			billing_interval VARCHAR(20) NOT NULL DEFAULT 'monthly',
			features         JSONB NOT NULL DEFAULT '{}',
			active           BOOLEAN NOT NULL DEFAULT TRUE,
			sort_order       INTEGER NOT NULL DEFAULT 0,
			gateway_price_id VARCHAR(255),
			gateway_name     VARCHAR(32) NOT NULL DEFAULT 'stripe',
			created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at       TIMESTAMP DEFAULT NOW(),
			CONSTRAINT chk_billing_interval CHECK (billing_interval IN ('monthly', 'yearly', 'lifetime'))
		)
	`)
	if err != nil {
		return fmt.Errorf("create plans: %w", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_plans_active ON plans(active)`)
	if err != nil {
		return fmt.Errorf("create idx_plans_active: %w", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_plans_slug ON plans(slug)`)
	if err != nil {
		return fmt.Errorf("create idx_plans_slug: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS subscriptions (
			id                      BIGSERIAL PRIMARY KEY,
			user_id                 BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			plan_id                 BIGINT NOT NULL REFERENCES plans(id),
			status                  VARCHAR(50) NOT NULL DEFAULT 'active',
			gateway_subscription_id VARCHAR(255),
			gateway_customer_id     VARCHAR(255),
			gateway_name            VARCHAR(32) NOT NULL DEFAULT 'stripe',
			current_period_start    TIMESTAMP,
			current_period_end      TIMESTAMP,
			cancel_at_period_end    BOOLEAN NOT NULL DEFAULT FALSE,
			canceled_at             TIMESTAMP,
			trial_end               TIMESTAMP,
			created_at              TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at              TIMESTAMP DEFAULT NOW(),
			CONSTRAINT chk_subscription_status CHECK (status IN ('active', 'trialing', 'past_due', 'canceled', 'incomplete', 'expired'))
		)
	`)
	if err != nil {
		return fmt.Errorf("create subscriptions: %w", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id)`)
	if err != nil {
		return fmt.Errorf("create idx_subscriptions_user_id: %w", err)
	}

	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_gateway_id ON subscriptions(gateway_subscription_id) WHERE gateway_subscription_id IS NOT NULL`)
	if err != nil {
		return fmt.Errorf("create idx_subscriptions_gateway_id: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS payment_events (
			id               BIGSERIAL PRIMARY KEY,
			gateway_event_id VARCHAR(255) NOT NULL,
			event_type       VARCHAR(100) NOT NULL,
			subscription_id  BIGINT REFERENCES subscriptions(id),
			user_id          BIGINT REFERENCES users(id),
			payload          JSONB NOT NULL,
			processed        BOOLEAN NOT NULL DEFAULT FALSE,
			processed_at     TIMESTAMP,
			error            TEXT,
			gateway_name     VARCHAR(32) NOT NULL DEFAULT 'stripe',
			created_at       TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create payment_events: %w", err)
	}

	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_events_gateway_event ON payment_events(gateway_name, gateway_event_id)`)
	if err != nil {
		return fmt.Errorf("create idx_payment_events_gateway_event: %w", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_payment_events_gateway_event_id ON payment_events(gateway_event_id)`)
	if err != nil {
		return fmt.Errorf("create idx_payment_events_gateway_event_id: %w", err)
	}

	return nil
}

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

func request(req *http.Request) (*http.Response, error) {
	return fiberApp.Test(req, 10_000)
}

func truncatePlans(t *testing.T) {
	t.Helper()
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("truncate open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec("TRUNCATE TABLE payment_events, subscriptions, plans RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate plans: %v", err)
	}
}

// createAdminToken registers a user, sets admin=true in DB, logs in, and returns the access token.
func createAdminToken(t *testing.T) string {
	t.Helper()

	reg := registerUser(t, "Admin User", "admin@example.com", "password123")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec("UPDATE users SET admin = true WHERE id = $1", reg.User.ID); err != nil {
		t.Fatalf("set admin: %v", err)
	}

	login := loginUser(t, "admin@example.com", "password123")
	return login.TokenPair.AccessToken
}

// createUserToken registers a regular user, logs in, and returns the access token and user ID.
func createUserToken(t *testing.T, email string) (string, int64) {
	t.Helper()

	reg := registerUser(t, "Regular User", email, "password123")
	login := loginUser(t, email, "password123")
	return login.TokenPair.AccessToken, reg.User.ID
}

// resetMockGateway zeroes all Called/LastInput/Return fields on the global mock
// so each test starts from a clean state. Call from inside a test or t.Cleanup.
func resetMockGateway(t *testing.T) {
	t.Helper()
	if mockGateway == nil {
		t.Fatal("mockGateway not initialized — fx wiring missing?")
	}
	*mockGateway = paymentmock.Gateway{SignatureHeaderValue: "Stripe-Signature"}
}

// seedPlan inserts a plan row directly via SQL and returns its ID.
func seedPlan(t *testing.T, slug, gatewayPriceID string) int64 {
	t.Helper()
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	var id int64
	err = db.QueryRow(`
		INSERT INTO plans (name, slug, price_cents, currency, billing_interval, features, gateway_price_id, gateway_name)
		VALUES ($1, $1, 2990, 'BRL', 'monthly', '{}', $2, 'stripe') RETURNING id
	`, slug, gatewayPriceID).Scan(&id)
	if err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	return id
}

// seedActiveSubscription inserts an active subscription row for the given user/plan.
// gatewaySubscriptionID may be empty for "no remote sub yet" scenarios.
func seedActiveSubscription(t *testing.T, userID, planID int64, gatewayCustomerID, gatewaySubscriptionID string) int64 {
	t.Helper()
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var subSQLValue any
	if gatewaySubscriptionID == "" {
		subSQLValue = nil
	} else {
		subSQLValue = gatewaySubscriptionID
	}

	var id int64
	err = db.QueryRow(`
		INSERT INTO subscriptions (user_id, plan_id, status, gateway_subscription_id, gateway_customer_id, gateway_name)
		VALUES ($1, $2, 'active', $3, $4, 'stripe') RETURNING id
	`, userID, planID, subSQLValue, gatewayCustomerID).Scan(&id)
	if err != nil {
		t.Fatalf("seed subscription: %v", err)
	}
	return id
}

// promoteToAdmin sets users.admin = true for a given user ID.
func promoteToAdmin(t *testing.T, userID int64) {
	t.Helper()
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec("UPDATE users SET admin = true WHERE id = $1", userID); err != nil {
		t.Fatalf("promote admin: %v", err)
	}
}
