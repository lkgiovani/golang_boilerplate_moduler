package planspersistence

import (
	"context"
	"errors"
	"time"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/plans/plansdomain/plansrepo"
	"golang_boilerplate_module/internal/shared/domain/exceptions"
	sharedrepo "golang_boilerplate_module/internal/shared/infra/persistence/repositories"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

// GORMPaymentEventRepository implements PaymentEventRepository using GORM.
type GORMPaymentEventRepository struct {
	*sharedrepo.GORMGenericRepository[plansdomain.PaymentEvent, int64]
	db *gorm.DB
}

// NewGORMPaymentEventRepository creates a new GORM-based PaymentEventRepository.
func NewGORMPaymentEventRepository(db *gorm.DB) plansrepo.PaymentEventRepository {
	return &GORMPaymentEventRepository{
		GORMGenericRepository: sharedrepo.NewGORMGenericRepository[plansdomain.PaymentEvent, int64](db),
		db:                    db,
	}
}

// GetByStripeEventID retrieves a payment event by its Stripe event ID.
func (r *GORMPaymentEventRepository) GetByStripeEventID(ctx context.Context, stripeEventID string) (*plansdomain.PaymentEvent, error) {
	ctx, span := dbTracer.Start(ctx, "GORMPaymentEventRepository.GetByStripeEventID")
	defer span.End()

	span.SetAttributes(attribute.String("db.operation", "GetByStripeEventID"))

	var event plansdomain.PaymentEvent
	err := r.db.WithContext(ctx).Where("stripe_event_id = ?", stripeEventID).First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		span.SetStatus(codes.Error, "not found")
		return nil, exceptions.NewNotFoundException("Payment event not found", nil)
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, exceptions.NewInternalException(map[string]any{"error": err.Error()})
	}

	span.SetAttributes(attribute.Int64("payment_event.id", event.ID))
	return &event, nil
}

// MarkProcessed marks a payment event as successfully processed.
func (r *GORMPaymentEventRepository) MarkProcessed(ctx context.Context, id int64) error {
	ctx, span := dbTracer.Start(ctx, "GORMPaymentEventRepository.MarkProcessed")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.operation", "MarkProcessed"),
		attribute.Int64("payment_event.id", id),
	)

	now := time.Now()
	err := r.db.WithContext(ctx).
		Model(&plansdomain.PaymentEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"processed":    true,
			"processed_at": now,
		}).Error
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return exceptions.NewInternalException(map[string]any{"error": err.Error()})
	}

	return nil
}

// MarkFailed marks a payment event as processed with an error message.
func (r *GORMPaymentEventRepository) MarkFailed(ctx context.Context, id int64, errMsg string) error {
	ctx, span := dbTracer.Start(ctx, "GORMPaymentEventRepository.MarkFailed")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.operation", "MarkFailed"),
		attribute.Int64("payment_event.id", id),
	)

	now := time.Now()
	err := r.db.WithContext(ctx).
		Model(&plansdomain.PaymentEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"processed":    true,
			"processed_at": now,
			"error":        errMsg,
		}).Error
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return exceptions.NewInternalException(map[string]any{"error": err.Error()})
	}

	return nil
}
