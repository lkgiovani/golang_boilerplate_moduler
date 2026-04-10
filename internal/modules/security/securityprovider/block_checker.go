package securityprovider

import (
	"context"

	"golang_boilerplate_module/internal/modules/security/application/securityusecases"
	"golang_boilerplate_module/internal/shared/domain/providers"
)

type blockCheckerImpl struct {
	checkUC *securityusecases.CheckBlockUseCase
}

// NewBlockChecker returns a providers.BlockChecker backed by CheckBlockUseCase.
func NewBlockChecker(checkUC *securityusecases.CheckBlockUseCase) providers.BlockChecker {
	return &blockCheckerImpl{checkUC: checkUC}
}

func (b *blockCheckerImpl) IsBlocked(ctx context.Context, userID int64) (providers.BlockStatus, error) {
	return b.checkUC.Execute(ctx, userID)
}
