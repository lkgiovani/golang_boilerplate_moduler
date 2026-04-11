package usersusecases

import (
	"context"

	"golang_boilerplate_module/internal/modules/users/usersdomain"
	"golang_boilerplate_module/internal/modules/users/usersdomain/usersrepo"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
	"golang_boilerplate_module/internal/shared/domain/providers"
	"golang_boilerplate_module/internal/shared/infra/observability"

	"go.opentelemetry.io/otel/attribute"
)

// UpdateMeInput holds the mutable profile fields for the authenticated user.
type UpdateMeInput struct {
	UserID int64   `json:"-"`
	Name   *string `json:"name,omitempty"`
	ImgURL *string `json:"img_url,omitempty"`
}

// UpdateMeUseCase updates the authenticated user's name and/or avatar.
type UpdateMeUseCase struct {
	userRepo usersrepo.UserRepository
	logger   providers.LoggerProvider
}

// NewUpdateMeUseCase creates a new UpdateMeUseCase.
func NewUpdateMeUseCase(userRepo usersrepo.UserRepository, logger providers.LoggerProvider) *UpdateMeUseCase {
	return &UpdateMeUseCase{userRepo: userRepo, logger: logger}
}

// Execute builds a partial update map and applies it.
func (uc *UpdateMeUseCase) Execute(ctx context.Context, input UpdateMeInput) (*usersdomain.User, error) {
	ctx, span := userTracer.Start(ctx, "UpdateMeUseCase.Execute")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", input.UserID))

	log := observability.LoggerWithTrace(ctx, uc.logger).With("usecase", "UpdateMe", "userId", input.UserID)

	updates := map[string]any{}
	if input.Name != nil {
		if *input.Name == "" {
			err := exceptions.NewBadRequestException("Name cannot be empty", nil)
			observability.RecordError(span, err)
			return nil, err
		}
		updates["name"] = *input.Name
	}
	if input.ImgURL != nil {
		updates["img_url"] = *input.ImgURL
	}

	if len(updates) == 0 {
		err := exceptions.NewBadRequestException("No fields to update", nil)
		observability.RecordError(span, err)
		return nil, err
	}

	updated, err := uc.userRepo.UpdateByID(ctx, input.UserID, updates)
	if err != nil {
		log.Error("failed to update user", "error", err.Error())
		observability.RecordError(span, err)
		return nil, err
	}

	log.Info("user profile updated")
	return updated, nil
}
