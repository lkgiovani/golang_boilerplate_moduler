---
phase: 06-security-monitoring
plan: 01
subsystem: security
tags: [security, auto-block, fx, migrations]
requires: [V3 security tables, providers.BlockChecker interface]
provides:
  - securityusecases.RecordActivityUseCase
  - securityusecases.CheckBlockUseCase
  - securityusecases.UnblockUserUseCase
  - securityusecases.Policy + DefaultPolicy
  - securityprovider.NewBlockChecker (providers.BlockChecker impl)
  - security.Module (fx)
  - securityrepo.ErrAlreadyBlocked sentinel
  - SecurityRepository.AutoUnblock
  - BlockStatus.BlockedUntil
affects:
  - internal/bootstrap/app.go (fx wiring)
  - internal/shared/domain/providers/security_provider.go (interface extension)
  - internal/modules/security/infra/securitypersistence/gorm_security_repository.go
  - migrations/V7__security_timestamptz.sql
tech-stack:
  added: []
  patterns: [fx.Module, OTel spans, GORM unique-violation detection via 23505, handwritten mocks]
key-files:
  created:
    - internal/modules/security/application/securityusecases/policy.go
    - internal/modules/security/application/securityusecases/record_activity.go
    - internal/modules/security/application/securityusecases/check_block.go
    - internal/modules/security/application/securityusecases/unblock_user.go
    - internal/modules/security/application/securityusecases/record_activity_test.go
    - internal/modules/security/securityprovider/block_checker.go
    - internal/modules/security/module.go
    - migrations/V7__security_timestamptz.sql
  modified:
    - internal/shared/domain/providers/security_provider.go
    - internal/modules/security/securitydomain/securityrepo/security_repository.go
    - internal/modules/security/infra/securitypersistence/gorm_security_repository.go
    - internal/bootstrap/app.go
decisions:
  - Auto-block policy defaults: 1h window, CriticalThreshold=3, BlockDuration=24h, HighThreshold=10 (tracked-only)
  - Admin users are exempt from auto-block (checked by UserIsAdmin input flag, not by re-querying user repo to avoid cycle)
  - Repository returns ErrAlreadyBlocked sentinel on 23505 / idx_unique_active_block unique violation; usecase swallows it to stay idempotent under races
  - AutoUnblock uses unblocked_by = NULL to distinguish system expiry from manual admin unblock
  - V7 migration converts V3 timestamp columns to TIMESTAMPTZ for consistent UTC handling
  - security.Module wired before auth.Module in bootstrap so Plan 02 middleware can consume providers.BlockChecker
metrics:
  tasks: 3
  tests_added: 6
  duration: ~
---

# Phase 06 Plan 01: Security Application Layer Summary

Standalone, testable security module delivering RecordActivity auto-block policy, CheckBlock lazy expiry, manual UnblockUser, and the fx-wired BlockChecker provider so the Plan 02 auth middleware can enforce blocks without creating an import cycle.

## Tasks Completed

| # | Task | Commit |
|---|------|--------|
| 1 | Extend BlockChecker interface, add AutoUnblock + ErrAlreadyBlocked, V7 migration | 9cee0da |
| 2 | Policy + RecordActivity/CheckBlock/UnblockUser usecases + 6 unit tests | f7a1d8b |
| 3 | BlockChecker provider impl + security.Module + bootstrap registration | f6467bd |

## Policy Defaults

```go
Policy{
    Window:            time.Hour,
    CriticalThreshold: 3,
    HighThreshold:     10,
    BlockDuration:     24 * time.Hour,
}
```

## ErrAlreadyBlocked Pattern

The V3 migration defines a unique partial index `idx_unique_active_block ON user_security_blocks(user_id) WHERE unblocked_at IS NULL`. When a race causes two concurrent `BlockUser` calls, the second hits a 23505 unique violation. The GORM repository detects this (via substring match on `"23505"` / `"idx_unique_active_block"`) and returns the exported sentinel `securityrepo.ErrAlreadyBlocked`. `RecordActivityUseCase.Execute` uses `errors.Is` to swallow this and return nil, keeping the usecase idempotent.

## Tests (6/6 passing)

- `TestRecordActivity_Low_NoBlock`
- `TestRecordActivity_CriticalBelowThreshold_NoBlock`
- `TestRecordActivity_CriticalAtThreshold_Blocks` (verifies ~24h BlockedUntil, count=3)
- `TestRecordActivity_AdminExempt`
- `TestRecordActivity_ExistingActiveBlock_Skips`
- `TestRecordActivity_ErrAlreadyBlocked_Swallowed`

## Verification

- `go build ./...` green
- `go test -count=1 ./internal/modules/security/...` → PASS (6 tests)
- `grep -r "modules/auth" internal/modules/security` → empty (no cycle)
- `security.Module` present in `internal/bootstrap/app.go` fx.Options

## Deviations from Plan

None - plan executed exactly as written. The `pgconn` import fallback mentioned in the plan's Task 1 action step was resolved by using the documented substring-match fallback (no new dependency needed).

## Self-Check: PASSED

- All created files exist on disk
- All three commits present in git log (9cee0da, f7a1d8b, f6467bd)
- Unit tests pass
- go build succeeds
