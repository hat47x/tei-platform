package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/resource"
)

type supervisorBroker struct {
	mu        sync.Mutex
	renewed   chan struct{}
	refreshes int
	failRenew bool
}

func (b *supervisorBroker) Acquire(context.Context, resource.AcquireRequest) (resource.Lease, error) {
	return resource.Lease{}, errors.New("not used")
}
func (b *supervisorBroker) Release(context.Context, resource.Lease) error { return nil }
func (b *supervisorBroker) Renew(_ context.Context, _ resource.AcquireRequest, previous resource.Lease) (resource.Lease, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failRenew {
		return resource.Lease{}, errors.New("reference renewal failure")
	}
	previous.Revision++
	previous.ExpiresAt = time.Now().Add(250 * time.Millisecond).UTC().Format(time.RFC3339Nano)
	if b.renewed != nil {
		select {
		case b.renewed <- struct{}{}:
		default:
		}
	}
	return previous, nil
}
func (b *supervisorBroker) Refresh(_ context.Context, _ resource.AcquireRequest, previous resource.Lease) (resource.Lease, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refreshes++
	previous.Revision++
	previous.Resources = append([]resource.Resolved(nil), previous.Resources...)
	for i := range previous.Resources {
		if previous.Resources[i].Kind == permission.KindSecret {
			previous.Resources[i].SecretHandle += "#refresh"
		}
	}
	return previous, nil
}

type immediateRefreshTrigger struct{ fired bool }

func (t *immediateRefreshTrigger) WaitForRefresh(ctx context.Context, _ resource.Lease) error {
	if !t.fired {
		t.fired = true
		return nil
	}
	<-ctx.Done()
	return ctx.Err()
}

func supervisionHas(xs []resource.SupervisionRecord, state resource.SupervisionState) bool {
	for _, x := range xs {
		if x.State == state {
			return true
		}
	}
	return false
}

func TestLeaseSupervisorRenewsWithoutPluginCall(t *testing.T) {
	broker := &supervisorBroker{renewed: make(chan struct{}, 1)}
	initial := resource.Lease{ID: "lease-supervised", Revision: 1, IssuedAt: time.Now().UTC().Format(time.RFC3339Nano), ExpiresAt: time.Now().Add(100 * time.Millisecond).UTC().Format(time.RFC3339Nano), RuntimePolicyFingerprint: "policy"}
	req := resource.AcquireRequest{RuntimePolicyFingerprint: "policy", LeasePolicy: resource.LeasePolicy{MaxLifetime: "2s", RenewBefore: "60ms", RevokeTimeout: "1s", RenewalFailure: resource.RenewalFailClosed}}
	controller := newLeaseController(broker, req, initial)
	s := &leaseSupervisor{controller: controller, recorder: controller, policy: req.LeasePolicy}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.run(ctx) }()
	select {
	case <-broker.renewed:
		cancel()
	case <-time.After(800 * time.Millisecond):
		t.Fatal("supervisor did not renew before expiry")
	}
	if err := <-done; err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	if controller.Current().Revision != 2 {
		t.Fatalf("revision=%d", controller.Current().Revision)
	}
	events := controller.Supervision()
	if !supervisionHas(events, resource.SupervisionRenewalDue) || !supervisionHas(events, resource.SupervisionRenewed) {
		t.Fatalf("supervision=%#v", events)
	}
}

func TestLeaseSupervisorRefreshTriggerIsHostSide(t *testing.T) {
	broker := &supervisorBroker{}
	secretID := core.MustCanonicalID("prod.refresh.secret")
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "credential"}
	initial := resource.Lease{ID: "lease-refresh", Revision: 1, IssuedAt: time.Now().UTC().Format(time.RFC3339Nano), ExpiresAt: time.Now().Add(500 * time.Millisecond).UTC().Format(time.RFC3339Nano), RuntimePolicyFingerprint: "policy", Resources: []resource.Resolved{{Kind: perm.Kind, Access: perm.Access, Scope: perm.Scope, ResourceID: secretID, SecretHandle: "opaque://v1"}}}
	req := resource.AcquireRequest{RuntimePolicyFingerprint: "policy", Permissions: []permission.Spec{perm}, Bindings: []resource.Binding{{Kind: perm.Kind, Scope: perm.Scope, ResourceID: secretID, Refresh: resource.RefreshLive}}, LeasePolicy: resource.LeasePolicy{MaxLifetime: "2s", RenewBefore: "100ms", RevokeTimeout: "1s", RenewalFailure: resource.RenewalFailClosed}}
	controller := newLeaseController(broker, req, initial)
	trigger := &immediateRefreshTrigger{}
	s := &leaseSupervisor{controller: controller, recorder: controller, policy: req.LeasePolicy, trigger: trigger}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.run(ctx) }()
	deadline := time.After(500 * time.Millisecond)
	for controller.Current().Revision < 2 {
		select {
		case err := <-done:
			t.Fatalf("supervisor exited before refresh: %v; events=%#v", err, controller.Supervision())
		case <-deadline:
			t.Fatal("host refresh trigger did not refresh lease")
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	events := controller.Supervision()
	if !supervisionHas(events, resource.SupervisionRefreshDue) || !supervisionHas(events, resource.SupervisionRefreshed) {
		t.Fatalf("supervision=%#v", events)
	}
}

func TestLeaseSupervisorRecordsDeadlineMissed(t *testing.T) {
	broker := &supervisorBroker{}
	initial := resource.Lease{ID: "lease-expired", Revision: 1, IssuedAt: time.Now().Add(-time.Second).UTC().Format(time.RFC3339Nano), ExpiresAt: time.Now().Add(-time.Millisecond).UTC().Format(time.RFC3339Nano), RuntimePolicyFingerprint: "policy"}
	req := resource.AcquireRequest{RuntimePolicyFingerprint: "policy", LeasePolicy: resource.LeasePolicy{MaxLifetime: "2s", RenewBefore: "100ms", RevokeTimeout: "1s", RenewalFailure: resource.RenewalFailClosed}}
	controller := newLeaseController(broker, req, initial)
	s := &leaseSupervisor{controller: controller, recorder: controller, policy: req.LeasePolicy}
	if err := s.run(context.Background()); err == nil {
		t.Fatal("expected renewal deadline error")
	}
	if !supervisionHas(controller.Supervision(), resource.SupervisionRenewalDeadlineMissed) {
		t.Fatalf("supervision=%#v", controller.Supervision())
	}
}
