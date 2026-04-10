---
phase: 07-plans-subscriptions-payments
plan: 01
subsystem: plans-domain
tags: [domain, entities, repositories, payment-gateway, config]
dependency_graph:
  requires: [V6 migration]
  provides: [Plan entity, Subscription entity, PaymentEvent entity, PlanRepository, SubscriptionRepository, PaymentEventRepository, PaymentGateway interface, StripeConfig]
  affects: [plans module, shared providers, config]
tech_stack:
  added: []
  patterns: [GORM entities with OTel tracing, adapter interface pattern, generic repository extension]
key_files:
  created:
    - internal/shared/domain/providers/payment_gateway_provider.go
    - internal/modules/plans/plansdomain/plan.go
    - internal/modules/plans/plansdomain/subscription.go
    - internal/modules/plans/plansdomain/payment_event.go
    - internal/modules/plans/plansdomain/plansrepo/plan_repository.go
    - internal/modules/plans/plansdomain/plansrepo/subscription_repository.go
    - internal/modules/plans/plansdomain/plansrepo/payment_event_repository.go
    - internal/modules/plans/infra/planspersistence/gorm_plan_repository.go
    - internal/modules/plans/infra/planspersistence/gorm_subscription_repository.go
    - internal/modules/plans/infra/planspersistence/gorm_payment_event_repository.go
  modified:
    - internal/config/config.go
decisions:
  - Used same OTel tracer pattern as users.persistence with package-scoped dbTracer
  - Plan.HasFeature supports bool, numeric, and string truthy checks from JSONB
metrics:
  duration: 4m12s
  completed: "2026-04-10T13:56:11Z"
---

# Phase 07 Plan 01: Plans Domain Layer Summary

Plans domain layer with GORM entities matching V6 migration, repository interfaces extending GenericRepository, GORM implementations with OTel tracing, and PaymentGateway adapter interface for Stripe integration.

## Task Results

| Task | Name | Commit | Status |
|------|------|--------|--------|
| 1 | PaymentGateway interface, Config extension, domain entities | 67ac2ed | Done |
| 2 | Repository interfaces and GORM implementations with OTel | e079253 | Done |

## Key Artifacts

### PaymentGateway Interface
- `CreateCustomer`, `CreateCheckoutSession`, `CancelSubscription`, `VerifyWebhookSignature`
- Follows same adapter pattern as EmailProvider in shared providers

### Domain Entities
- **Plan**: GORM entity with `HasFeature(string) bool` for JSONB feature checks
- **Subscription**: Status constants (active, trialing, past_due, canceled, incomplete, expired), Plan preload relation
- **PaymentEvent**: Stripe event log for webhook idempotency

### Repository Interfaces
- **PlanRepository**: GenericRepository + GetBySlug, ListActive
- **SubscriptionRepository**: GenericRepository + GetActiveByUserID, GetByStripeSubscriptionID, GetByStripeCustomerID
- **PaymentEventRepository**: GenericRepository + GetByStripeEventID, MarkProcessed, MarkFailed

### Config Extension
- Added `StripeConfig` with `SecretKey` and `WebhookSecret` loaded from `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET` env vars

## Deviations from Plan

None - plan executed exactly as written.

## Verification

- `go build ./internal/shared/domain/providers/...` -- PASS
- `go build ./internal/config/...` -- PASS
- `go build ./internal/modules/plans/...` -- PASS (all domain + persistence)
