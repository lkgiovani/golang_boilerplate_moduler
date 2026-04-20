package authdomain_test

import (
	"testing"

	"golang_boilerplate_module/internal/modules/auth/authdomain"
	"golang_boilerplate_module/internal/shared/domain/errs"
)

func TestAuthDomainErrors(t *testing.T) {
	cases := []struct {
		name       string
		got        *errs.Error
		wantCode   errs.Code
		reportable bool
	}{
		{"InvalidCredentials", authdomain.InvalidCredentials(), errs.EUNAUTHORIZED, false},
		{"AccountDisabled", authdomain.AccountDisabled(), errs.EUNAUTHORIZED, false},
		{"TokenAlreadyUsed", authdomain.TokenAlreadyUsed(), errs.ECONFLICT, false},
		{"TokenExpired", authdomain.TokenExpired(), errs.EEXPIRED, false},
		{"NoLocalPassword", authdomain.NoLocalPassword(), errs.EPRECONDITION, false},
		{"UserNotFound", authdomain.UserNotFound(), errs.ENOTFOUND, false},
		{"FailedToHashPassword", authdomain.FailedToHashPassword(), errs.EINTERNAL, true},
		{"FailedToGenerateTokens", authdomain.FailedToGenerateTokens(), errs.EINTERNAL, true},
		{"FailedToSendVerificationEmail", authdomain.FailedToSendVerificationEmail(), errs.EINTERNAL, true},
		{"MissingCredentials", authdomain.MissingCredentials(), errs.EBADREQUEST, false},
		{"InvalidRequestBody", authdomain.InvalidRequestBody(), errs.EBADREQUEST, false},
		{"MissingAuthToken", authdomain.MissingAuthToken(), errs.EUNAUTHORIZED, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got == nil {
				t.Fatalf("factory returned nil")
			}
			if tc.got.Code != tc.wantCode {
				t.Errorf("code: got %q, want %q", tc.got.Code, tc.wantCode)
			}
			if tc.got.Reportable != tc.reportable {
				t.Errorf("reportable: got %v, want %v", tc.got.Reportable, tc.reportable)
			}
			if tc.got.Message == "" {
				t.Error("message is empty")
			}
		})
	}
}
