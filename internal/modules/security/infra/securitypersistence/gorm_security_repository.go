package securitypersistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang_boilerplate_module/internal/modules/security/securitydomain"
	"golang_boilerplate_module/internal/modules/security/securitydomain/securityrepo"
	"golang_boilerplate_module/internal/shared/domain/errs"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

var securityTracer = otel.Tracer("security.persistence")

// GORMSecurityRepository implements securityrepo.SecurityRepository using GORM.
type GORMSecurityRepository struct {
	db *gorm.DB
}

// NewGORMSecurityRepository creates a new GORM-backed security repository.
func NewGORMSecurityRepository(db *gorm.DB) securityrepo.SecurityRepository {
	return &GORMSecurityRepository{db: db}
}

func wrapSecurityInternal(err error, op string) *errs.Error {
	e := errs.Wrap(errs.EINTERNAL, err, "security_repository.%s failed", op)
	e.Reportable = true
	return e
}

func (r *GORMSecurityRepository) CreateSuspiciousActivity(ctx context.Context, activity *securitydomain.SuspiciousActivity) error {
	ctx, span := securityTracer.Start(ctx, "GORMSecurityRepository.CreateSuspiciousActivity")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("activity.user_id", activity.UserID),
		attribute.String("activity.type", string(activity.ActivityType)),
		attribute.String("activity.severity", string(activity.Severity)),
	)

	if err := r.db.WithContext(ctx).Create(activity).Error; err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return wrapSecurityInternal(err, "CreateSuspiciousActivity")
	}

	span.SetStatus(codes.Ok, "created")
	return nil
}

func (r *GORMSecurityRepository) GetRecentByUserID(ctx context.Context, userID int64, since time.Time) ([]securitydomain.SuspiciousActivity, error) {
	ctx, span := securityTracer.Start(ctx, "GORMSecurityRepository.GetRecentByUserID")
	defer span.End()

	span.SetAttributes(attribute.Int64("activity.user_id", userID))

	var activities []securitydomain.SuspiciousActivity
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND created_at >= ?", userID, since).
		Order("created_at DESC").
		Find(&activities).Error

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, wrapSecurityInternal(err, "GetRecentByUserID")
	}

	span.SetAttributes(attribute.Int("activities.count", len(activities)))
	span.SetStatus(codes.Ok, "found")
	return activities, nil
}

func (r *GORMSecurityRepository) CountByUserAndSeverity(ctx context.Context, userID int64, severity securitydomain.Severity, since time.Time) (int64, error) {
	ctx, span := securityTracer.Start(ctx, "GORMSecurityRepository.CountByUserAndSeverity")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("activity.user_id", userID),
		attribute.String("activity.severity", string(severity)),
	)

	var count int64
	err := r.db.WithContext(ctx).
		Model(&securitydomain.SuspiciousActivity{}).
		Where("user_id = ? AND severity = ? AND created_at >= ?", userID, severity, since).
		Count(&count).Error

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return 0, wrapSecurityInternal(err, "CountByUserAndSeverity")
	}

	span.SetAttributes(attribute.Int64("activities.count", count))
	span.SetStatus(codes.Ok, "counted")
	return count, nil
}

func (r *GORMSecurityRepository) BlockUser(ctx context.Context, block *securitydomain.UserSecurityBlock) error {
	ctx, span := securityTracer.Start(ctx, "GORMSecurityRepository.BlockUser")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("block.user_id", block.UserID),
		attribute.String("block.reason", block.Reason),
	)

	if err := r.db.WithContext(ctx).Create(block).Error; err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		msg := err.Error()
		if strings.Contains(msg, "23505") || strings.Contains(msg, "idx_unique_active_block") {
			return securityrepo.ErrAlreadyBlocked
		}
		return wrapSecurityInternal(err, "BlockUser")
	}

	span.SetStatus(codes.Ok, "blocked")
	return nil
}

func (r *GORMSecurityRepository) AutoUnblock(ctx context.Context, userID int64) error {
	ctx, span := securityTracer.Start(ctx, "GORMSecurityRepository.AutoUnblock")
	defer span.End()

	span.SetAttributes(attribute.Int64("block.user_id", userID))

	result := r.db.WithContext(ctx).
		Model(&securitydomain.UserSecurityBlock{}).
		Where("user_id = ? AND unblocked_at IS NULL", userID).
		Updates(map[string]any{
			"unblocked_at": time.Now(),
			"unblocked_by": nil,
		})

	if result.Error != nil {
		span.SetStatus(codes.Error, result.Error.Error())
		span.RecordError(result.Error)
		return wrapSecurityInternal(result.Error, "AutoUnblock")
	}

	span.SetStatus(codes.Ok, "auto-unblocked")
	return nil
}

func (r *GORMSecurityRepository) GetActiveBlock(ctx context.Context, userID int64) (*securitydomain.UserSecurityBlock, error) {
	ctx, span := securityTracer.Start(ctx, "GORMSecurityRepository.GetActiveBlock")
	defer span.End()

	span.SetAttributes(attribute.Int64("block.user_id", userID))

	var block securitydomain.UserSecurityBlock
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND unblocked_at IS NULL", userID).
		First(&block).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		span.SetStatus(codes.Ok, "no active block")
		return nil, nil
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, wrapSecurityInternal(err, "GetActiveBlock")
	}

	span.SetStatus(codes.Ok, "found active block")
	return &block, nil
}

func (r *GORMSecurityRepository) UnblockUser(ctx context.Context, userID int64, unblockedBy int64) error {
	ctx, span := securityTracer.Start(ctx, "GORMSecurityRepository.UnblockUser")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("block.user_id", userID),
		attribute.Int64("block.unblocked_by", unblockedBy),
	)

	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&securitydomain.UserSecurityBlock{}).
		Where("user_id = ? AND unblocked_at IS NULL", userID).
		Updates(map[string]any{
			"unblocked_at": now,
			"unblocked_by": unblockedBy,
		})

	if result.Error != nil {
		span.SetStatus(codes.Error, result.Error.Error())
		span.RecordError(result.Error)
		return wrapSecurityInternal(result.Error, "UnblockUser")
	}

	if result.RowsAffected == 0 {
		span.SetStatus(codes.Error, "no active block found")
		return securitydomain.NoActiveBlock()
	}

	span.SetStatus(codes.Ok, "unblocked")
	return nil
}
