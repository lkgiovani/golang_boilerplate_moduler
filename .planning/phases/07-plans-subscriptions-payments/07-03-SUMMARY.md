---
phase: 07-plans-subscriptions-payments
plan: 03
subsystem: plans-subscriptions
tags: [subscriptions, webhooks, stripe, feature-gating, payments]
dependency_graph:
  requires: [07-01, 07-02]
  provides: [subscription-use-cases, webhook-handler, feature-gate-middleware]
  affects: [plans-module, subscription-endpoints]
tech_stack:
  added: []
  patterns: [idempotent-webhook-processing, lazy-customer-creation, feature-gating-middleware]
key_files:
  created:
    - internal/modules/plans/application/plansusecases/subscribe.go
    - internal/modules/plans/application/plansusecases/cancel_subscription.go
    - internal/modules/plans/application/plansusecases/get_subscription.go
    - internal/modules/plans/application/plansusecases/handle_webhook.go
    - internal/modules/plans/infra/planshttp/feature_gate_middleware.go
    - internal/modules/plans/infra/planshttp/webhook_controller.go
  modified: []
decisions:
  - Webhook handler uses minimal stripeEvent struct for JSON unmarshalling instead of stripe-go types
  - UpdateByID returns (*T, error) per GenericRepository contract; handlers discard the returned entity
  - Feature gate stores subscriptionID and planSlug in Locals for downstream use
metrics:
  duration: 207s
  completed: "2026-04-10T14:20:26Z"
  tasks_completed: 2
  tasks_total: 2
  files_created: 6
  files_modified: 0
---

# Phase 07 Plan 03: Subscription Use Cases, Webhooks, and Feature Gating Summary

Subscription lifecycle use cases (subscribe, cancel, get) with Stripe checkout session creation, idempotent webhook handler processing 5 event types, and feature gate middleware for plan-based access control.

## Completed Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create subscription use cases and webhook handler | 66b1c58 | subscribe.go, cancel_subscription.go, get_subscription.go, handle_webhook.go |
| 2 | Create feature gate middleware and webhook controller | 84ffde5 | feature_gate_middleware.go, webhook_controller.go |

## Implementation Details

### Subscribe Use Case
- Checks for existing active subscription before proceeding
- Validates plan has StripePriceID configured
- Lazily creates Stripe customer via PaymentGateway.CreateCustomer
- Creates checkout session via PaymentGateway.CreateCheckoutSession
- Stores local subscription record with StatusIncomplete and StripeCustomerID

### Cancel Subscription Use Case
- Finds active subscription by user ID
- Calls Stripe CancelSubscription (logs warning on failure, does not block)
- Updates local status to canceled with canceled_at timestamp

### Get Subscription Use Case
- Returns active subscription by user ID with preloaded plan data
- Returns 404 if no active subscription exists

### Webhook Handler
- Verifies signature via PaymentGateway.VerifyWebhookSignature
- Parses verified payload using minimal stripeEvent struct (id, type, data.object)
- Idempotency: checks payment_events table by stripe_event_id before processing
- Routes 5 event types: checkout.session.completed, invoice.paid, invoice.payment_failed, customer.subscription.deleted, customer.subscription.updated
- Marks events as processed or failed in payment_events table

### Feature Gate Middleware
- Extracts userID from Locals (set by auth middleware)
- Checks active subscription exists via SubscriptionRepository
- Loads plan and checks HasFeature for the required feature key
- Returns 403 Forbidden if subscription missing or feature not in plan
- Sets subscriptionID and planSlug in Locals for downstream handlers

### Webhook Controller
- Reads raw c.Body() before any parsing for signature verification
- Validates Stripe-Signature header presence
- Uses shared tracer var from plans_controller.go (no duplicate declaration)
- Returns 200 OK on success

## Deviations from Plan

None - plan executed exactly as written.

## Self-Check: PASSED
