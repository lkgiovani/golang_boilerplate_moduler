package securityusecases

import (
	"context"
	"time"

	"golang_boilerplate_module/internal/modules/security/securitydomain/securityrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
)

// CheckBlockUseCase looks up the active block for a user and lazily expires
// it if BlockedUntil has passed.
type CheckBlockUseCase struct {
	repo   securityrepo.SecurityRepository
	logger providers.LoggerProvider
}

func NewCheckBlockUseCase(repo securityrepo.SecurityRepository, logger providers.LoggerProvider) *CheckBlockUseCase {
	return &CheckBlockUseCase{repo: repo, logger: logger}
}

func (uc *CheckBlockUseCase) Execute(ctx context.Context, userID int64) (providers.BlockStatus, error) {
	ctx, span := securityTracer.Start(ctx, "CheckBlockUseCase.Execute")
	defer span.End()

	active, err := uc.repo.GetActiveBlock(ctx, userID)
	if err != nil {
		return providers.BlockStatus{}, err
	}
	if active == nil {
		return providers.BlockStatus{Blocked: false}, nil
	}

	if active.BlockedUntil != nil && active.BlockedUntil.Before(time.Now()) {
		_ = uc.repo.AutoUnblock(ctx, userID)
		uc.logger.Info("lazy auto-unblock", "userID", userID)
		return providers.BlockStatus{Blocked: false}, nil
	}

	return providers.BlockStatus{
		Blocked:      true,
		Reason:       active.Reason,
		BlockedUntil: active.BlockedUntil,
	}, nil
}
