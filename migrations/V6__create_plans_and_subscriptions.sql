CREATE TABLE plans (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    price_cents BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'BRL',
    billing_interval VARCHAR(20) NOT NULL DEFAULT 'monthly',
    features JSONB NOT NULL DEFAULT '{}',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    stripe_price_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT chk_billing_interval CHECK (billing_interval IN ('monthly', 'yearly', 'lifetime'))
);

CREATE INDEX idx_plans_active ON plans(active);
CREATE INDEX idx_plans_slug ON plans(slug);

CREATE TABLE subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id BIGINT NOT NULL REFERENCES plans(id),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    stripe_subscription_id VARCHAR(255),
    stripe_customer_id VARCHAR(255),
    current_period_start TIMESTAMP,
    current_period_end TIMESTAMP,
    cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE,
    canceled_at TIMESTAMP,
    trial_end TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT chk_subscription_status CHECK (status IN ('active', 'trialing', 'past_due', 'canceled', 'incomplete', 'expired'))
);

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_plan_id ON subscriptions(plan_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE UNIQUE INDEX idx_subscriptions_stripe_id ON subscriptions(stripe_subscription_id) WHERE stripe_subscription_id IS NOT NULL;

CREATE TABLE payment_events (
    id BIGSERIAL PRIMARY KEY,
    stripe_event_id VARCHAR(255) NOT NULL UNIQUE,
    event_type VARCHAR(100) NOT NULL,
    subscription_id BIGINT REFERENCES subscriptions(id),
    user_id BIGINT REFERENCES users(id),
    payload JSONB NOT NULL,
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    processed_at TIMESTAMP,
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payment_events_stripe_event_id ON payment_events(stripe_event_id);
CREATE INDEX idx_payment_events_subscription_id ON payment_events(subscription_id);
CREATE INDEX idx_payment_events_processed ON payment_events(processed);

COMMENT ON TABLE plans IS 'Planos de assinatura disponiveis';
COMMENT ON TABLE subscriptions IS 'Assinaturas de usuarios vinculadas a planos';
COMMENT ON TABLE payment_events IS 'Log de eventos do Stripe para idempotencia';
