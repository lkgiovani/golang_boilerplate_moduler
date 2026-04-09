package securityusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/security/securitydomain/securityrepo"
	"golang_boilerplate_module/internal/shared/domain/providers"
)

// UnblockUserUseCase performs an admin-initiated manual unblock.
type UnblockUserUseCase struct {
	repo   securityrepo.SecurityRepository
	logger providers.LoggerProvider
}

func NewUnblockUserUseCase(repo securityrepo.SecurityRepository, logger providers.LoggerProvider) *UnblockUserUseCase {
	return &UnblockUserUseCase{repo: repo, logger: logger}
}

func (uc *UnblockUserUseCase) Execute(ctx context.Context, userID int64, unblockedBy int64) error {
	ctx, span := securityTracer.Start(ctx, "UnblockUserUseCase.Execute")
	defer span.End()

	if err := uc.repo.UnblockUser(ctx, userID, unblockedBy); err != nil {
		return err
	}
	uc.logger.Warn("user unblocked", "userID", userID, "unblockedBy", unblockedBy)
	return nil
}
