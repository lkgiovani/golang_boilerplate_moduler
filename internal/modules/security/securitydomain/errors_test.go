package securitydomain_test

import (
	"testing"

	"golang_boilerplate_module/internal/modules/security/securitydomain"
	"golang_boilerplate_module/internal/shared/domain/errs"
)

func TestSecurityDomainErrors_NoActiveBlock(t *testing.T) {
	got := securitydomain.NoActiveBlock()
	if got == nil {
		t.Fatal("factory returned nil")
	}
	if got.Code != errs.ENOTFOUND {
		t.Errorf("Code = %q, want %q", got.Code, errs.ENOTFOUND)
	}
	if got.Reportable {
		t.Errorf("Reportable = true, want false")
	}
}
