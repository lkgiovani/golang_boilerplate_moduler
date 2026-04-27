-- Phase 9: Gateway-agnostic payment provider abstraction
-- Renames every stripe_* column to gateway_*, adds gateway_name discriminator,
-- rebuilds uniqueness indexes, and adds customer-reuse columns to users.
-- Safe because project is pre-production (CONTEXT D-10) -- existing rows default to 'stripe'.

-- 1. Rename columns on plans, subscriptions, payment_events
ALTER TABLE plans          RENAME COLUMN stripe_price_id         TO gateway_price_id;
ALTER TABLE subscriptions  RENAME COLUMN stripe_subscription_id  TO gateway_subscription_id;
ALTER TABLE subscriptions  RENAME COLUMN stripe_customer_id      TO gateway_customer_id;
ALTER TABLE payment_events RENAME COLUMN stripe_event_id         TO gateway_event_id;

-- 2. Add gateway_name discriminator (NOT NULL with 'stripe' default for legacy rows)
ALTER TABLE plans          ADD COLUMN gateway_name VARCHAR(32) NOT NULL DEFAULT 'stripe';
ALTER TABLE subscriptions  ADD COLUMN gateway_name VARCHAR(32) NOT NULL DEFAULT 'stripe';
ALTER TABLE payment_events ADD COLUMN gateway_name VARCHAR(32) NOT NULL DEFAULT 'stripe';

-- 3. Add customer-reuse columns to users (nullable -- set on first subscribe per gateway)
ALTER TABLE users ADD COLUMN gateway_customer_id VARCHAR(255);
ALTER TABLE users ADD COLUMN gateway_name        VARCHAR(32);

-- 4. Rebuild payment_events uniqueness as composite (gateway_name, gateway_event_id)
ALTER TABLE payment_events DROP CONSTRAINT IF EXISTS payment_events_stripe_event_id_key;
DROP INDEX IF EXISTS idx_payment_events_stripe_event_id;
CREATE UNIQUE INDEX idx_payment_events_gateway_event ON payment_events(gateway_name, gateway_event_id);
CREATE INDEX idx_payment_events_gateway_event_id ON payment_events(gateway_event_id);

-- 5. Rebuild subscriptions partial unique on gateway_subscription_id
DROP INDEX IF EXISTS idx_subscriptions_stripe_id;
CREATE UNIQUE INDEX idx_subscriptions_gateway_id
    ON subscriptions(gateway_subscription_id)
    WHERE gateway_subscription_id IS NOT NULL;

-- 6. Comments
COMMENT ON COLUMN users.gateway_customer_id      IS 'Customer ID at payment gateway; reused across subscribe calls (fixes duplicate-customer bug)';
COMMENT ON COLUMN users.gateway_name             IS 'Which gateway owns the above customer_id (null when unset)';
COMMENT ON COLUMN plans.gateway_name             IS 'Which gateway owns the gateway_price_id mapping';
COMMENT ON COLUMN subscriptions.gateway_name     IS 'Which gateway created this subscription';
COMMENT ON COLUMN payment_events.gateway_name    IS 'Which gateway produced this event';
