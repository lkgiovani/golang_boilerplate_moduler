package healthdomain_test

import (
	"errors"
	"testing"

	"golang_boilerplate_module/internal/modules/health/healthdomain"
	"golang_boilerplate_module/internal/shared/domain/errs"
)

func TestHealthDomainErrors_DatabaseUnavailable(t *testing.T) {
	cause := errors.New("connection refused")
	got := healthdomain.DatabaseUnavailable(cause)
	if got == nil {
		t.Fatal("factory returned nil")
	}
	if got.Code != errs.EUNAVAILABLE {
		t.Errorf("Code = %q, want %q", got.Code, errs.EUNAVAILABLE)
	}
	if !got.Reportable {
		t.Errorf("Reportable = false, want true")
	}
	if !errors.Is(got, cause) {
		t.Errorf("errors.Is should walk Unwrap chain to cause")
	}
}
