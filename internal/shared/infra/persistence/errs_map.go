package persistence

import (
	"context"
	"errors"

	"golang_boilerplate_module/internal/shared/domain/errs"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Postgres SQLSTATE codes we care about.
// Reference: https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgCheckViolation      = "23514"
)

// FromGORM maps a GORM/pq/context error to an *errs.Error with the
// appropriate Code, wrapping the original cause so errors.Is still
// works on gorm.ErrRecordNotFound, gorm.ErrDuplicatedKey, etc.
//
// entityName is the human-readable name of the entity being operated
// on ("User", "Plan", ...) used to build the Message.
//
// Returns nil when err is nil — callers usually short-circuit earlier
// but the helper is defensive.
func FromGORM(err error, entityName string) *errs.Error {
	if err == nil {
		return nil
	}

	// Record not found — the most common GORM error; use ENOTFOUND.
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errs.Wrap(errs.ENOTFOUND, err, "%s not found", entityName)
	}

	// Duplicated key via GORM's dialect-agnostic sentinel.
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return errs.Wrap(errs.EDUPLICATION, err, "%s already exists", entityName)
	}

	// Postgres-specific SQLSTATE detection via lib/pq.
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch string(pqErr.Code) {
		case pgUniqueViolation:
			return errs.Wrap(errs.EDUPLICATION, err, "%s already exists", entityName)
		case pgForeignKeyViolation:
			return errs.Wrap(errs.EPRECONDITION, err, "%s references a missing row", entityName)
		case pgCheckViolation:
			return errs.Wrap(errs.EINVALID, err, "%s violates a check constraint", entityName)
		}
	}

	// Context cancellation / deadline — classify as timeout upstream.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return errs.Wrap(errs.ETIMEOUT, err, "%s operation timed out", entityName)
	}

	// Fallback: unknown DB error. Reportable so error_handler logs it.
	wrapped := errs.Wrap(errs.EINTERNAL, err, "db operation on %s failed", entityName)
	wrapped.Reportable = true
	return wrapped
}
