package plansdomain_test

import (
	"testing"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/shared/domain/errs"
)

func TestPlansDomainErrors(t *testing.T) {
	cases := []struct {
		name       string
		got        *errs.Error
		wantCode   errs.Code
		reportable bool
	}{
		{"MissingPlanName", plansdomain.MissingPlanName(), errs.EBADREQUEST, false},
		{"AlreadySubscribed", plansdomain.AlreadySubscribed(), errs.EBADREQUEST, false},
		{"InvalidWebhookSignature", plansdomain.InvalidWebhookSignature(), errs.EBADREQUEST, false},
		{"AuthRequired", plansdomain.AuthRequired(), errs.EUNAUTHORIZED, false},
		{"AdminAccessRequired", plansdomain.AdminAccessRequired(), errs.EFORBIDDEN, false},
		{"ActiveSubscriptionRequired", plansdomain.ActiveSubscriptionRequired(), errs.EFORBIDDEN, false},
		{"FeatureNotInPlan", plansdomain.FeatureNotInPlan("premium"), errs.EFORBIDDEN, false},
		{"PlanNotFound", plansdomain.PlanNotFound(), errs.ENOTFOUND, false},
		{"NoActiveSubscription", plansdomain.NoActiveSubscription(), errs.ENOTFOUND, false},
		{"FailedToLoadPlan", plansdomain.FailedToLoadPlan(), errs.EINTERNAL, true},
		{"FailedToCreateCheckout", plansdomain.FailedToCreateCheckout(), errs.EINTERNAL, true},
		{"FailedToCreateCustomer", plansdomain.FailedToCreateCustomer(), errs.EINTERNAL, true},
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
