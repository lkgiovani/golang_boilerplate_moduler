package plansusecases

import (
	"context"
	"testing"
	"time"

	"golang_boilerplate_module/internal/modules/plans/plansdomain"
	"golang_boilerplate_module/internal/shared/domain/providers"
)

// --- in-memory event repo ---------------------------------------------------

type stubEventRepo struct {
	byKey     map[string]*plansdomain.PaymentEvent
	added     []*plansdomain.PaymentEvent
	processed []int64
	failed    map[int64]string
	nextID    int64
}

func newStubEventRepo() *stubEventRepo {
	return &stubEventRepo{
		byKey:  map[string]*plansdomain.PaymentEvent{},
		failed: map[int64]string{},
	}
}

func key(name, id string) string { return name + "|" + id }

func (s *stubEventRepo) GetByGatewayEventID(_ context.Context, gatewayName, gatewayEventID string) (*plansdomain.PaymentEvent, error) {
	if pe, ok := s.byKey[key(gatewayName, gatewayEventID)]; ok {
		return pe, nil
	}
	return nil, plansdomain.PaymentEventNotFound()
}
func (s *stubEventRepo) Add(_ context.Context, pe *plansdomain.PaymentEvent) (*plansdomain.PaymentEvent, error) {
	s.nextID++
	pe.ID = s.nextID
	s.added = append(s.added, pe)
	s.byKey[key(pe.GatewayName, pe.GatewayEventID)] = pe
	return pe, nil
}
func (s *stubEventRepo) GetByID(_ context.Context, id int64) (*plansdomain.PaymentEvent, error) {
	for _, pe := range s.added {
		if pe.ID == id {
			return pe, nil
		}
	}
	return nil, plansdomain.PaymentEventNotFound()
}
func (s *stubEventRepo) UpdateByID(_ context.Context, _ int64, _ map[string]any) (*plansdomain.PaymentEvent, error) {
	return nil, nil
}
func (s *stubEventRepo) DeleteByID(_ context.Context, _ int64) error { return nil }
func (s *stubEventRepo) DeleteAll(_ context.Context) error           { return nil }
func (s *stubEventRepo) MarkProcessed(_ context.Context, id int64) error {
	s.processed = append(s.processed, id)
	for _, pe := range s.added {
		if pe.ID == id {
			pe.Processed = true
		}
	}
	return nil
}
func (s *stubEventRepo) MarkFailed(_ context.Context, id int64, msg string) error {
	s.failed[id] = msg
	return nil
}

// --- sub repo with a preloaded subscription ---------------------------------

type stubSubRepoForWebhook struct {
	byGatewaySubID  map[string]*plansdomain.Subscription
	byGatewayCustID map[string]*plansdomain.Subscription
	lastUpdates     map[int64]map[string]any
}

func newStubSubRepoForWebhook() *stubSubRepoForWebhook {
	return &stubSubRepoForWebhook{
		byGatewaySubID:  map[string]*plansdomain.Subscription{},
		byGatewayCustID: map[string]*plansdomain.Subscription{},
		lastUpdates:     map[int64]map[string]any{},
	}
}

func (s *stubSubRepoForWebhook) GetActiveByUserID(_ context.Context, _ int64) (*plansdomain.Subscription, error) {
	return nil, plansdomain.ActiveSubscriptionNotFound()
}
func (s *stubSubRepoForWebhook) GetByGatewaySubscriptionID(_ context.Context, gatewayName, id string) (*plansdomain.Subscription, error) {
	if sub, ok := s.byGatewaySubID[key(gatewayName, id)]; ok {
		return sub, nil
	}
	return nil, plansdomain.SubscriptionNotFound()
}
func (s *stubSubRepoForWebhook) GetByGatewayCustomerID(_ context.Context, gatewayName, id string) (*plansdomain.Subscription, error) {
	if sub, ok := s.byGatewayCustID[key(gatewayName, id)]; ok {
		return sub, nil
	}
	return nil, plansdomain.SubscriptionNotFound()
}
func (s *stubSubRepoForWebhook) Add(_ context.Context, sub *plansdomain.Subscription) (*plansdomain.Subscription, error) {
	return sub, nil
}
func (s *stubSubRepoForWebhook) GetByID(_ context.Context, _ int64) (*plansdomain.Subscription, error) {
	return nil, plansdomain.SubscriptionNotFound()
}
func (s *stubSubRepoForWebhook) UpdateByID(_ context.Context, id int64, updates map[string]any) (*plansdomain.Subscription, error) {
	s.lastUpdates[id] = updates
	return nil, nil
}
func (s *stubSubRepoForWebhook) DeleteByID(_ context.Context, _ int64) error { return nil }
func (s *stubSubRepoForWebhook) DeleteAll(_ context.Context) error           { return nil }

// --- tests ------------------------------------------------------------------

func TestHandleWebhook_CheckoutCompleted(t *testing.T) {
	subRepo := newStubSubRepoForWebhook()
	sub := &plansdomain.Subscription{ID: 7, UserID: 1, PlanID: 2, GatewayName: "stripe"}
	subRepo.byGatewayCustID[key("stripe", "cus_abc")] = sub

	evt := &providers.PaymentEvent{
		Type:                   providers.PaymentEventCheckoutCompleted,
		GatewayEventID:         "evt_1",
		GatewayCustomerRef:     "cus_abc",
		GatewaySubscriptionRef: "sub_xyz",
	}
	uc := NewHandleWebhookUseCase(subRepo, newStubEventRepo(), logProvider{})
	if err := uc.Execute(context.Background(), HandleWebhookInput{GatewayName: "stripe", Event: evt}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	updates, ok := subRepo.lastUpdates[7]
	if !ok {
		t.Fatal("expected UpdateByID to be called on subscription 7")
	}
	if updates["status"] != "active" {
		t.Errorf("expected status=active, got %v", updates["status"])
	}
	if updates["gateway_subscription_id"] != "sub_xyz" {
		t.Errorf("expected gateway_subscription_id=sub_xyz, got %v", updates["gateway_subscription_id"])
	}
}

func TestHandleWebhook_PaymentSucceeded_SetsActiveAndPeriod(t *testing.T) {
	subRepo := newStubSubRepoForWebhook()
	sub := &plansdomain.Subscription{ID: 8, GatewayName: "stripe"}
	subRepo.byGatewaySubID[key("stripe", "sub_abc")] = sub

	start := time.Unix(1700000000, 0)
	end := time.Unix(1702592000, 0)
	evt := &providers.PaymentEvent{
		Type:                   providers.PaymentEventPaymentSucceeded,
		GatewayEventID:         "evt_2",
		GatewaySubscriptionRef: "sub_abc",
		CurrentPeriodStart:     &start,
		CurrentPeriodEnd:       &end,
	}
	uc := NewHandleWebhookUseCase(subRepo, newStubEventRepo(), logProvider{})
	if err := uc.Execute(context.Background(), HandleWebhookInput{GatewayName: "stripe", Event: evt}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	updates := subRepo.lastUpdates[8]
	if updates["status"] != "active" {
		t.Errorf("expected status=active, got %v", updates["status"])
	}
	if updates["current_period_start"] != start {
		t.Errorf("period_start mismatch: %v", updates["current_period_start"])
	}
}

func TestHandleWebhook_UnknownEventSkips(t *testing.T) {
	subRepo := newStubSubRepoForWebhook()
	eventRepo := newStubEventRepo()

	evt := &providers.PaymentEvent{
		Type:           providers.PaymentEventUnknown,
		GatewayEventID: "evt_unknown",
	}
	uc := NewHandleWebhookUseCase(subRepo, eventRepo, logProvider{})
	if err := uc.Execute(context.Background(), HandleWebhookInput{GatewayName: "stripe", Event: evt}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(eventRepo.processed) != 1 {
		t.Errorf("expected exactly one MarkProcessed call, got %d", len(eventRepo.processed))
	}
	if len(subRepo.lastUpdates) != 0 {
		t.Errorf("unknown events must not trigger any subscription update, got %d", len(subRepo.lastUpdates))
	}
}

func TestHandleWebhook_Idempotent(t *testing.T) {
	subRepo := newStubSubRepoForWebhook()
	eventRepo := newStubEventRepo()

	// Pre-insert an already-processed event record for the same (gateway, event id).
	eventRepo.nextID = 99
	processed := &plansdomain.PaymentEvent{
		ID:             100,
		GatewayName:    "stripe",
		GatewayEventID: "evt_dup",
		Processed:      true,
	}
	eventRepo.byKey[key("stripe", "evt_dup")] = processed

	evt := &providers.PaymentEvent{
		Type:                   providers.PaymentEventPaymentSucceeded,
		GatewayEventID:         "evt_dup",
		GatewaySubscriptionRef: "sub_xyz",
	}
	uc := NewHandleWebhookUseCase(subRepo, eventRepo, logProvider{})
	if err := uc.Execute(context.Background(), HandleWebhookInput{GatewayName: "stripe", Event: evt}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subRepo.lastUpdates) != 0 {
		t.Error("idempotent re-delivery must not re-apply updates")
	}
	if len(eventRepo.processed) != 0 {
		t.Error("idempotent re-delivery must not call MarkProcessed again")
	}
}
