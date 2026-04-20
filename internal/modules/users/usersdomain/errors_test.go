package usersdomain_test

import (
	"strings"
	"testing"

	"golang_boilerplate_module/internal/modules/users/usersdomain"
	"golang_boilerplate_module/internal/shared/domain/errs"
)

func TestUsersDomainErrors(t *testing.T) {
	if got := usersdomain.EmailTaken("a@b.com"); got.Code != errs.EDUPLICATION {
		t.Errorf("EmailTaken code: got %q, want %q", got.Code, errs.EDUPLICATION)
	} else if !strings.Contains(got.Message, "a@b.com") {
		t.Errorf("EmailTaken message missing email: %q", got.Message)
	}
	if got := usersdomain.UserNotFound(); got.Code != errs.ENOTFOUND {
		t.Errorf("UserNotFound: got %q", got.Code)
	}
	if got := usersdomain.MissingNameOrEmail(); got.Code != errs.EBADREQUEST {
		t.Errorf("MissingNameOrEmail: got %q", got.Code)
	}
	if got := usersdomain.InvalidUserID(); got.Code != errs.EBADREQUEST {
		t.Errorf("InvalidUserID: got %q", got.Code)
	}
	if got := usersdomain.MissingUserIdentity(); got.Code != errs.EUNAUTHORIZED {
		t.Errorf("MissingUserIdentity: got %q", got.Code)
	}
}
