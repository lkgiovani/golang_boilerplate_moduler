package securityusecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"golang_boilerplate_module/internal/modules/security/securitydomain"
	"golang_boilerplate_module/internal/modules/security/securitydomain/securityrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var securityTracer = otel.Tracer("security.usecases")

// RecordActivityInput carries the fields needed to record a suspicious activity.
type RecordActivityInput struct {
	UserID       int64
	UserIsAdmin  bool
	ActivityType securitydomain.ActivityType
	Severity     securitydomain.Severity
	Endpoint     string
	IPAddress    *string
	UserAgent    *string
	Details      json.RawMessage
}

// RecordActivityUseCase persists a suspicious activity and auto-blocks the
// user when the configured CRITICAL threshold is reached within the window.
type RecordActivityUseCase struct {
	repo   securityrepo.SecurityRepository
	policy Policy
	logger providers.LoggerProvider
}

func NewRecordActivityUseCase(repo securityrepo.SecurityRepository, policy Policy, logger providers.LoggerProvider) *RecordActivityUseCase {
	return &RecordActivityUseCase{repo: repo, policy: policy, logger: logger}
}

func (uc *RecordActivityUseCase) Execute(ctx context.Context, input RecordActivityInput) error {
	ctx, span := securityTracer.Start(ctx, "RecordActivityUseCase.Execute")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("user_id", input.UserID),
		attribute.String("activity_type", string(input.ActivityType)),
		attribute.String("severity", string(input.Severity)),
	)

	activity := &securitydomain.SuspiciousActivity{
		UserID:       input.UserID,
		ActivityType: input.ActivityType,
		Endpoint:     input.Endpoint,
		IPAddress:    input.IPAddress,
		UserAgent:    input.UserAgent,
		Severity:     input.Severity,
		RequestCount: 1,
		Details:      input.Details,
		CreatedAt:    time.Now(),
	}

	if err := uc.repo.CreateSuspiciousActivity(ctx, activity); err != nil {
		return err
	}

	if input.UserIsAdmin {
		uc.logger.Info("admin exempt from auto-block", "userID", input.UserID)
		return nil
	}

	if input.Severity != securitydomain.SeverityCritical {
		return nil
	}

	since := time.Now().Add(-uc.policy.Window)
	count, err := uc.repo.CountByUserAndSeverity(ctx, input.UserID, securitydomain.SeverityCritical, since)
	if err != nil {
		return err
	}

	if count < uc.policy.CriticalThreshold {
		return nil
	}

	active, err := uc.repo.GetActiveBlock(ctx, input.UserID)
	if err != nil {
		return err
	}
	if active != nil {
		return nil
	}

	until := time.Now().Add(uc.policy.BlockDuration)
	block := &securitydomain.UserSecurityBlock{
		UserID:                  input.UserID,
		Reason:                  fmt.Sprintf("Auto-block: %d CRITICAL activities in %s", count, uc.policy.Window),
		SuspiciousActivityCount: int(count),
		BlockedAt:               time.Now(),
		BlockedUntil:            &until,
	}

	if err := uc.repo.BlockUser(ctx, block); err != nil {
		if errors.Is(err, securityrepo.ErrAlreadyBlocked) {
			uc.logger.Info("block already exists (race)", "userID", input.UserID)
			return nil
		}
		return err
	}

	uc.logger.Warn("user auto-blocked", "userID", input.UserID, "count", count, "until", until)
	return nil
}
