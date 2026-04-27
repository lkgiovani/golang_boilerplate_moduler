package plansusecases

import (
	"context"
	"strings"
	"testing"

	"golang_boilerplate_module/internal/config"
	paymentmock "golang_boilerplate_module/internal/modules/payments/infra/mock"
	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/modules/users/usersdomain"
	"golang_boilerplate_module/internal/shared/domain/providers"
)

// --- in-memory test doubles -------------------------------------------------

type stubPlanRepo struct {
	plan *plansdomain.Plan
}

func (s *stubPlanRepo) GetBySlug(_ context.Context, slug string) (*plansdomain.Plan, error) {
	if s.plan != nil && s.plan.Slug == slug {
		return s.plan, nil
	}
	return nil, plansdomain.PlanNotFound()
}
func (s *stubPlanRepo) ListActive(_ context.Context) ([]plansdomain.Plan, error) { return nil, nil }
func (s *stubPlanRepo) Add(_ context.Context, p *plansdomain.Plan) (*plansdomain.Plan, error) {
	s.plan = p
	return p, nil
}
func (s *stubPlanRepo) GetByID(_ context.Context, _ int64) (*plansdomain.Plan, error) {
	if s.plan != nil {
		return s.plan, nil
	}
	return nil, plansdomain.PlanNotFound()
}
func (s *stubPlanRepo) UpdateByID(_ context.Context, _ int64, _ map[string]any) (*plansdomain.Plan, error) {
	return s.plan, nil
}
func (s *stubPlanRepo) DeleteByID(_ context.Context, _ int64) error { return nil }
func (s *stubPlanRepo) DeleteAll(_ context.Context) error           { return nil }

type stubSubRepo struct {
	activeByUser map[int64]*plansdomain.Subscription
	added        []*plansdomain.Subscription
}

func (s *stubSubRepo) GetActiveByUserID(_ context.Context, userID int64) (*plansdomain.Subscription, error) {
	if sub, ok := s.activeByUser[userID]; ok {
		return sub, nil
	}
	return nil, plansdomain.ActiveSubscriptionNotFound()
}
func (s *stubSubRepo) GetByGatewaySubscriptionID(_ context.Context, _, _ string) (*plansdomain.Subscription, error) {
	return nil, plansdomain.SubscriptionNotFound()
}
func (s *stubSubRepo) GetByGatewayCustomerID(_ context.Context, _, _ string) (*plansdomain.Subscription, error) {
	return nil, plansdomain.SubscriptionNotFound()
}
func (s *stubSubRepo) Add(_ context.Context, sub *plansdomain.Subscription) (*plansdomain.Subscription, error) {
	sub.ID = int64(len(s.added) + 1)
	s.added = append(s.added, sub)
	return sub, nil
}
func (s *stubSubRepo) GetByID(_ context.Context, id int64) (*plansdomain.Subscription, error) {
	for _, sub := range s.added {
		if sub.ID == id {
			return sub, nil
		}
	}
	return nil, plansdomain.SubscriptionNotFound()
}
func (s *stubSubRepo) UpdateByID(_ context.Context, id int64, updates map[string]any) (*plansdomain.Subscription, error) {
	for _, sub := range s.added {
		if sub.ID == id {
			if v, ok := updates["status"].(string); ok {
				sub.Status = plansdomain.SubscriptionStatus(v)
			}
			return sub, nil
		}
	}
	return nil, plansdomain.SubscriptionNotFound()
}
func (s *stubSubRepo) DeleteByID(_ context.Context, _ int64) error { return nil }
func (s *stubSubRepo) DeleteAll(_ context.Context) error           { return nil }

type stubUserRepo struct {
	user *usersdomain.User
}

func (s *stubUserRepo) GetByEmail(_ context.Context, _ string) (*usersdomain.User, error) {
	return s.user, nil
}
func (s *stubUserRepo) GetByID(_ context.Context, id int64) (*usersdomain.User, error) {
	if s.user != nil && s.user.ID == id {
		return s.user, nil
	}
	return nil, usersdomain.UserNotFound()
}
func (s *stubUserRepo) Add(_ context.Context, u *usersdomain.User) (*usersdomain.User, error) {
	u.ID = 1
	s.user = u
	return u, nil
}
func (s *stubUserRepo) UpdateByID(_ context.Context, _ int64, _ map[string]any) (*usersdomain.User, error) {
	return s.user, nil
}
func (s *stubUserRepo) DeleteByID(_ context.Context, _ int64) error { return nil }
func (s *stubUserRepo) DeleteAll(_ context.Context) error           { return nil }
func (s *stubUserRepo) UpdateGatewayCustomer(_ context.Context, _ int64, gatewayName, gatewayCustomerID string) error {
	s.user.GatewayName = &gatewayName
	s.user.GatewayCustomerID = &gatewayCustomerID
	return nil
}

// logProvider is a no-op providers.LoggerProvider for tests.
type logProvider struct{}

func (logProvider) Info(string, ...any)                  {}
func (logProvider) Warn(string, ...any)                  {}
func (logProvider) Error(string, ...any)                 {}
func (logProvider) Debug(string, ...any)                 {}
func (logProvider) With(...any) providers.LoggerProvider { return logProvider{} }
func (logProvider) Sync() error                          { return nil }

// --- tests ------------------------------------------------------------------

func newSubscribeUC(t *testing.T, user *usersdomain.User, plan *plansdomain.Plan, mock *paymentmock.Gateway) (*SubscribeUseCase, *stubUserRepo, *stubSubRepo) {
	t.Helper()
	cfg := &config.Config{PaymentGateway: config.PaymentGatewayConfig{Name: "stripe"}}
	planRepo := &stubPlanRepo{plan: plan}
	subRepo := &stubSubRepo{activeByUser: map[int64]*plansdomain.Subscription{}}
	userRepo := &stubUserRepo{user: user}
	uc := NewSubscribeUseCase(cfg, planRepo, subRepo, userRepo, mock, logProvider{})
	return uc, userRepo, subRepo
}

func priceID(v string) *string { return &v }
func strPtr(v string) *string  { return &v }

func TestSubscribe_CreatesCustomerWhenAbsent(t *testing.T) {
	mock := paymentmock.New()
	plan := &plansdomain.Plan{ID: 10, Slug: "basic", GatewayPriceID: priceID("price_123"), GatewayName: "stripe"}
	user := &usersdomain.User{ID: 42, Email: "new@example.com", Name: "New"}

	uc, userRepo, _ := newSubscribeUC(t, user, plan, mock)
	_, err := uc.Execute(context.Background(), SubscribeInput{
		UserID:    42,
		UserEmail: "new@example.com",
		UserName:  "New",
		PlanSlug:  "basic",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.CreateCustomerCalled {
		t.Error("expected CreateCustomer to be called when user has no GatewayCustomerID")
	}
	if userRepo.user.GatewayCustomerID == nil || *userRepo.user.GatewayCustomerID == "" {
		t.Error("expected userRepo.UpdateGatewayCustomer to persist new customer id")
	}
}

func TestSubscribe_ReusesGatewayCustomer(t *testing.T) {
	mock := paymentmock.New()
	plan := &plansdomain.Plan{ID: 10, Slug: "basic", GatewayPriceID: priceID("price_123"), GatewayName: "stripe"}
	existingCustomer := "cus_existing_stripe"
	user := &usersdomain.User{
		ID:                42,
		Email:             "ret@example.com",
		GatewayCustomerID: strPtr(existingCustomer),
		GatewayName:       strPtr("stripe"),
	}

	uc, _, _ := newSubscribeUC(t, user, plan, mock)
	_, err := uc.Execute(context.Background(), SubscribeInput{
		UserID:    42,
		UserEmail: "ret@example.com",
		PlanSlug:  "basic",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.CreateCustomerCalled {
		t.Error("expected CreateCustomer to be SKIPPED when user already has GatewayCustomerID for the active gateway")
	}
	if mock.LastCheckoutInput == nil || mock.LastCheckoutInput.CustomerID != existingCustomer {
		t.Errorf("expected checkout to use existing customer %q, got %+v", existingCustomer, mock.LastCheckoutInput)
	}
}

func TestSubscribe_IgnoresWrongGatewayReuse(t *testing.T) {
	mock := paymentmock.New()
	plan := &plansdomain.Plan{ID: 10, Slug: "basic", GatewayPriceID: priceID("price_123"), GatewayName: "stripe"}
	// User has a customer id from a DIFFERENT gateway; active is stripe.
	user := &usersdomain.User{
		ID:                42,
		Email:             "cross@example.com",
		GatewayCustomerID: strPtr("cus_mp_xyz"),
		GatewayName:       strPtr("mercadopago"),
	}

	uc, _, _ := newSubscribeUC(t, user, plan, mock)
	_, err := uc.Execute(context.Background(), SubscribeInput{
		UserID:    42,
		UserEmail: "cross@example.com",
		PlanSlug:  "basic",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.CreateCustomerCalled {
		t.Error("expected CreateCustomer to be called (wrong-gateway customer id must not be reused)")
	}
	if mock.LastCheckoutInput == nil || strings.HasPrefix(mock.LastCheckoutInput.CustomerID, "cus_mp_") {
		t.Errorf("checkout should not use the wrong-gateway customer id; got %+v", mock.LastCheckoutInput)
	}
}
