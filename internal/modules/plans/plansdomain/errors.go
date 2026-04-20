package plansdomain

import "golang_boilerplate_module/internal/shared/domain/errs"

func reportable(e *errs.Error) *errs.Error {
	e.Reportable = true
	return e
}

// Validation

func MissingPlanName() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "plan name is required")
}

func MissingPlanSlug() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "plan slug is required")
}

func InvalidBillingInterval() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "billing_interval must be monthly, yearly, or lifetime")
}

func InvalidRequestBody() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "invalid request body")
}

func InvalidPlanID() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "invalid plan ID")
}

func AlreadySubscribed() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "user already has an active subscription")
}

func MissingStripePrice() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "plan has no Stripe price configured")
}

func EmptyWebhookPayload() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "empty webhook payload")
}

func MissingStripeSignature() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "missing Stripe-Signature header")
}

func InvalidWebhookSignature() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "invalid webhook signature")
}

func InvalidWebhookPayload() *errs.Error {
	return errs.Errorf(errs.EBADREQUEST, "invalid webhook payload")
}

// Auth

func AuthRequired() *errs.Error {
	return errs.Errorf(errs.EUNAUTHORIZED, "authentication required")
}

// Forbidden

func AdminAccessRequired() *errs.Error {
	return errs.Errorf(errs.EFORBIDDEN, "admin access required")
}

func ActiveSubscriptionRequired() *errs.Error {
	return errs.Errorf(errs.EFORBIDDEN, "active subscription required")
}

func FeatureNotInPlan(feature string) *errs.Error {
	return errs.Errorf(errs.EFORBIDDEN, "plan does not include feature: %s", feature)
}

// Not found

func PlanNotFound() *errs.Error {
	return errs.Errorf(errs.ENOTFOUND, "Plan not found")
}

func NoActiveSubscription() *errs.Error {
	return errs.Errorf(errs.ENOTFOUND, "no active subscription")
}

func ActiveSubscriptionNotFound() *errs.Error {
	return errs.Errorf(errs.ENOTFOUND, "Active subscription not found")
}

func SubscriptionNotFound() *errs.Error {
	return errs.Errorf(errs.ENOTFOUND, "Subscription not found")
}

func PaymentEventNotFound() *errs.Error {
	return errs.Errorf(errs.ENOTFOUND, "Payment event not found")
}

// Internal (reportable)

func FailedToLoadPlan() *errs.Error {
	return reportable(errs.Errorf(errs.EINTERNAL, "failed to load plan"))
}

func FailedToCreateCustomer() *errs.Error {
	return reportable(errs.Errorf(errs.EINTERNAL, "failed to create payment customer"))
}

func FailedToCreateCheckout() *errs.Error {
	return reportable(errs.Errorf(errs.EINTERNAL, "failed to create checkout session"))
}

func FailedToCreateSubscription() *errs.Error {
	return reportable(errs.Errorf(errs.EINTERNAL, "failed to create subscription record"))
}
