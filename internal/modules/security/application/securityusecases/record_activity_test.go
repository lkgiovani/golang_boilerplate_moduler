package securityusecases

import (
	"context"
	"testing"
	"time"

	"golang_boilerplate_module/internal/modules/security/securitydomain"
	"golang_boilerplate_module/internal/modules/security/securitydomain/securityrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
)

// --- no-op logger ---

type noopLogger struct{}

func (n *noopLogger) Info(string, ...any)                   {}
func (n *noopLogger) Warn(string, ...any)                   {}
func (n *noopLogger) Error(string, ...any)                  {}
func (n *noopLogger) Debug(string, ...any)                  {}
func (n *noopLogger) With(...any) providers.LoggerProvider  { return n }
func (n *noopLogger) Sync() error                           { return nil }

// --- mock repo ---

type mockSecurityRepo struct {
	activities    []securitydomain.SuspiciousActivity
	criticalCount int64
	activeBlock   *securitydomain.UserSecurityBlock
	blockCalled   int
	blockUserErr  error
	lastBlock     *securitydomain.UserSecurityBlock
	autoUnblocked int
}

func (m *mockSecurityRepo) CreateSuspiciousActivity(_ context.Context, a *securitydomain.SuspiciousActivity) error {
	m.activities = append(m.activities, *a)
	return nil
}
func (m *mockSecurityRepo) GetRecentByUserID(context.Context, int64, time.Time) ([]securitydomain.SuspiciousActivity, error) {
	return nil, nil
}
func (m *mockSecurityRepo) CountByUserAndSeverity(_ context.Context, _ int64, _ securitydomain.Severity, _ time.Time) (int64, error) {
	return m.criticalCount, nil
}
func (m *mockSecurityRepo) BlockUser(_ context.Context, b *securitydomain.UserSecurityBlock) error {
	m.blockCalled++
	if m.blockUserErr != nil {
		return m.blockUserErr
	}
	m.lastBlock = b
	return nil
}
func (m *mockSecurityRepo) GetActiveBlock(context.Context, int64) (*securitydomain.UserSecurityBlock, error) {
	return m.activeBlock, nil
}
func (m *mockSecurityRepo) UnblockUser(context.Context, int64, int64) error { return nil }
func (m *mockSecurityRepo) AutoUnblock(context.Context, int64) error {
	m.autoUnblocked++
	return nil
}

// --- helpers ---

func newUC(repo *mockSecurityRepo) *RecordActivityUseCase {
	return NewRecordActivityUseCase(repo, DefaultPolicy(), &noopLogger{})
}

func baseInput(severity securitydomain.Severity, admin bool) RecordActivityInput {
	return RecordActivityInput{
		UserID:       42,
		UserIsAdmin:  admin,
		ActivityType: securitydomain.ActivityUnauthorizedAccess,
		Severity:     severity,
		Endpoint:     "/api/test",
	}
}

// --- tests ---

func TestRecordActivity_Low_NoBlock(t *testing.T) {
	repo := &mockSecurityRepo{criticalCount: 0}
	if err := newUC(repo).Execute(context.Background(), baseInput(securitydomain.SeverityLow, false)); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.blockCalled != 0 {
		t.Fatalf("expected 0 blocks, got %d", repo.blockCalled)
	}
	if len(repo.activities) != 1 {
		t.Fatalf("expected 1 activity recorded")
	}
}

func TestRecordActivity_CriticalBelowThreshold_NoBlock(t *testing.T) {
	repo := &mockSecurityRepo{criticalCount: 2}
	if err := newUC(repo).Execute(context.Background(), baseInput(securitydomain.SeverityCritical, false)); err != nil {
		t.Fatal(err)
	}
	if repo.blockCalled != 0 {
		t.Fatalf("expected 0 blocks, got %d", repo.blockCalled)
	}
}

func TestRecordActivity_CriticalAtThreshold_Blocks(t *testing.T) {
	repo := &mockSecurityRepo{criticalCount: 3}
	if err := newUC(repo).Execute(context.Background(), baseInput(securitydomain.SeverityCritical, false)); err != nil {
		t.Fatal(err)
	}
	if repo.blockCalled != 1 {
		t.Fatalf("expected 1 block, got %d", repo.blockCalled)
	}
	if repo.lastBlock == nil || repo.lastBlock.BlockedUntil == nil {
		t.Fatalf("expected block with BlockedUntil")
	}
	delta := time.Until(*repo.lastBlock.BlockedUntil)
	if delta < 23*time.Hour || delta > 25*time.Hour {
		t.Fatalf("expected ~24h block, got %v", delta)
	}
	if repo.lastBlock.SuspiciousActivityCount != 3 {
		t.Fatalf("expected count=3, got %d", repo.lastBlock.SuspiciousActivityCount)
	}
}

func TestRecordActivity_AdminExempt(t *testing.T) {
	repo := &mockSecurityRepo{criticalCount: 5}
	if err := newUC(repo).Execute(context.Background(), baseInput(securitydomain.SeverityCritical, true)); err != nil {
		t.Fatal(err)
	}
	if repo.blockCalled != 0 {
		t.Fatalf("admin should not be blocked, got %d", repo.blockCalled)
	}
}

func TestRecordActivity_ExistingActiveBlock_Skips(t *testing.T) {
	until := time.Now().Add(1 * time.Hour)
	repo := &mockSecurityRepo{
		criticalCount: 5,
		activeBlock:   &securitydomain.UserSecurityBlock{UserID: 42, BlockedUntil: &until},
	}
	if err := newUC(repo).Execute(context.Background(), baseInput(securitydomain.SeverityCritical, false)); err != nil {
		t.Fatal(err)
	}
	if repo.blockCalled != 0 {
		t.Fatalf("expected skip, got %d blocks", repo.blockCalled)
	}
}

func TestRecordActivity_ErrAlreadyBlocked_Swallowed(t *testing.T) {
	repo := &mockSecurityRepo{criticalCount: 3, blockUserErr: securityrepo.ErrAlreadyBlocked}
	if err := newUC(repo).Execute(context.Background(), baseInput(securitydomain.SeverityCritical, false)); err != nil {
		t.Fatalf("expected swallow, got %v", err)
	}
	if repo.blockCalled != 1 {
		t.Fatalf("expected 1 block attempt, got %d", repo.blockCalled)
	}
}
