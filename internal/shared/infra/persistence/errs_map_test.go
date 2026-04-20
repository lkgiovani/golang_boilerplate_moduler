package persistence

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"golang_boilerplate_module/internal/shared/domain/errs"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

func TestFromGORM_Nil(t *testing.T) {
	if got := FromGORM(nil, "User"); got != nil {
		t.Fatalf("FromGORM(nil) = %v, want nil", got)
	}
}

func TestFromGORM_RecordNotFound(t *testing.T) {
	e := FromGORM(gorm.ErrRecordNotFound, "User")
	if e == nil {
		t.Fatal("expected error, got nil")
	}
	if e.Code != errs.ENOTFOUND {
		t.Errorf("Code = %q, want %q", e.Code, errs.ENOTFOUND)
	}
	if e.Cause != gorm.ErrRecordNotFound {
		t.Errorf("Cause = %v, want gorm.ErrRecordNotFound", e.Cause)
	}
}

func TestFromGORM_RecordNotFound_UnwrapChain(t *testing.T) {
	e := FromGORM(gorm.ErrRecordNotFound, "User")
	if !errors.Is(e, gorm.ErrRecordNotFound) {
		t.Fatal("errors.Is should walk Unwrap chain to gorm.ErrRecordNotFound")
	}
}

func TestFromGORM_DuplicatedKey(t *testing.T) {
	e := FromGORM(gorm.ErrDuplicatedKey, "User")
	if e.Code != errs.EDUPLICATION {
		t.Fatalf("Code = %q, want %q", e.Code, errs.EDUPLICATION)
	}
}

func TestFromGORM_PqUniqueViolation(t *testing.T) {
	pqErr := &pq.Error{Code: "23505"}
	e := FromGORM(pqErr, "User")
	if e.Code != errs.EDUPLICATION {
		t.Fatalf("Code = %q, want %q", e.Code, errs.EDUPLICATION)
	}
}

func TestFromGORM_PqForeignKeyViolation(t *testing.T) {
	pqErr := &pq.Error{Code: "23503"}
	e := FromGORM(pqErr, "User")
	if e.Code != errs.EPRECONDITION {
		t.Fatalf("Code = %q, want %q", e.Code, errs.EPRECONDITION)
	}
}

func TestFromGORM_PqCheckViolation(t *testing.T) {
	pqErr := &pq.Error{Code: "23514"}
	e := FromGORM(pqErr, "User")
	if e.Code != errs.EINVALID {
		t.Fatalf("Code = %q, want %q", e.Code, errs.EINVALID)
	}
}

func TestFromGORM_ContextCanceled(t *testing.T) {
	e := FromGORM(context.Canceled, "User")
	if e.Code != errs.ETIMEOUT {
		t.Fatalf("Code = %q, want %q", e.Code, errs.ETIMEOUT)
	}
}

func TestFromGORM_ContextDeadline(t *testing.T) {
	e := FromGORM(context.DeadlineExceeded, "User")
	if e.Code != errs.ETIMEOUT {
		t.Fatalf("Code = %q, want %q", e.Code, errs.ETIMEOUT)
	}
}

func TestFromGORM_Unknown(t *testing.T) {
	raw := errors.New("boom")
	e := FromGORM(raw, "User")
	if e.Code != errs.EINTERNAL {
		t.Errorf("Code = %q, want %q", e.Code, errs.EINTERNAL)
	}
	if !e.Reportable {
		t.Errorf("Reportable = false, want true")
	}
	if e.Cause != raw {
		t.Errorf("Cause = %v, want %v", e.Cause, raw)
	}
}

func TestFromGORM_WrappedUnique(t *testing.T) {
	wrapped := fmt.Errorf("ctx: %w", &pq.Error{Code: "23505"})
	e := FromGORM(wrapped, "User")
	if e.Code != errs.EDUPLICATION {
		t.Fatalf("Code = %q, want %q", e.Code, errs.EDUPLICATION)
	}
}
