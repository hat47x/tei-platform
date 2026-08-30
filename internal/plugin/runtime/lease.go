package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"tei/internal/resource"
)

type leaseController struct {
	broker resource.Broker
	req    resource.AcquireRequest

	mu              sync.Mutex
	current         resource.Lease
	renewals        []resource.RenewalRecord
	refreshes       []resource.RefreshRecord
	supervision     []resource.SupervisionRecord
	ownershipStore  resource.SupervisorOwnershipStore
	ownership       *resource.SupervisorOwnership
	ownershipTTL    time.Duration
	ownershipEvents []resource.OwnershipRecord
	fatalErr        error
}

func newLeaseController(b resource.Broker, req resource.AcquireRequest, initial resource.Lease) *leaseController {
	c := &leaseController{broker: b, req: resource.NormalizeAcquireRequest(req), current: initial}
	c.recordSupervision(resource.SupervisionActive, "lease acquired")
	return c
}

func normalizedLeaseRevision(l resource.Lease) int {
	if l.Revision == 0 {
		return 1
	}
	return l.Revision
}

func (c *leaseController) attachOwnership(store resource.SupervisorOwnershipStore, ownership resource.SupervisorOwnership, ttl time.Duration, takeover bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.ownershipStore = store
	x := ownership
	c.ownership = &x
	c.ownershipTTL = ttl
	state := resource.OwnershipClaimed
	msg := "supervisor ownership claimed"
	if takeover {
		state = resource.OwnershipTakeover
		msg = "expired supervisor ownership taken over"
	}
	c.ownershipEvents = append(c.ownershipEvents, resource.OwnershipRecord{State: state, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), OwnerID: ownership.OwnerID, FencingToken: ownership.FencingToken, OwnerExpiresAt: ownership.OwnerExpiresAt, Message: msg})
	c.mu.Unlock()
}

func (c *leaseController) Ownership() (resource.SupervisorOwnership, bool) {
	if c == nil {
		return resource.SupervisorOwnership{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ownership == nil {
		return resource.SupervisorOwnership{}, false
	}
	return *c.ownership, true
}

func (c *leaseController) OwnershipHeartbeatInterval() time.Duration {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ownership == nil || c.ownershipTTL <= 0 {
		return 0
	}
	d := c.ownershipTTL / 3
	if d <= 0 {
		d = time.Millisecond
	}
	return d
}

func (c *leaseController) OwnershipRecords() []resource.OwnershipRecord {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]resource.OwnershipRecord(nil), c.ownershipEvents...)
}

func (c *leaseController) HeartbeatOwnership(ctx context.Context) error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	store, ttl := c.ownershipStore, c.ownershipTTL
	var current resource.SupervisorOwnership
	if c.ownership != nil {
		current = *c.ownership
	}
	c.mu.Unlock()
	if store == nil || current.LeaseID == "" {
		return nil
	}
	next, err := store.Heartbeat(ctx, current, ttl)
	if err != nil {
		// Cancellation of the invocation/supervisor context is the normal signal
		// used to stop the heartbeat loop. If the store reports that same context
		// termination, ownership has not been proven lost and fail-closed state
		// must not be synthesized from shutdown itself.
		if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
			return nil
		}
		c.mu.Lock()
		c.ownershipEvents = append(c.ownershipEvents, resource.OwnershipRecord{State: resource.OwnershipLost, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), OwnerID: current.OwnerID, FencingToken: current.FencingToken, OwnerExpiresAt: current.OwnerExpiresAt, Message: err.Error()})
		c.mu.Unlock()
		c.markFailure(fmt.Errorf("supervisor ownership heartbeat failed: %w", err))
		return err
	}
	c.mu.Lock()
	x := next
	c.ownership = &x
	c.ownershipEvents = append(c.ownershipEvents, resource.OwnershipRecord{State: resource.OwnershipHeartbeat, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), OwnerID: next.OwnerID, FencingToken: next.FencingToken, OwnerExpiresAt: next.OwnerExpiresAt, Message: "supervisor ownership heartbeat"})
	c.mu.Unlock()
	return nil
}

func (c *leaseController) ReleaseOwnership(ctx context.Context) error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	store := c.ownershipStore
	var current resource.SupervisorOwnership
	if c.ownership != nil {
		current = *c.ownership
	}
	c.mu.Unlock()
	if store == nil || current.LeaseID == "" {
		return nil
	}
	if err := store.Release(ctx, current); err != nil {
		c.mu.Lock()
		c.ownershipEvents = append(c.ownershipEvents, resource.OwnershipRecord{State: resource.OwnershipLost, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), OwnerID: current.OwnerID, FencingToken: current.FencingToken, OwnerExpiresAt: current.OwnerExpiresAt, Message: "ownership release failed: " + err.Error()})
		c.mu.Unlock()
		return err
	}
	c.mu.Lock()
	c.ownershipEvents = append(c.ownershipEvents, resource.OwnershipRecord{State: resource.OwnershipReleased, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), OwnerID: current.OwnerID, FencingToken: current.FencingToken, OwnerExpiresAt: current.OwnerExpiresAt, Message: "supervisor ownership released"})
	c.ownership = nil
	c.mu.Unlock()
	return nil
}

func (c *leaseController) installFence(ctx context.Context, token uint64) (resource.Lease, error) {
	if c == nil {
		return resource.Lease{}, fmt.Errorf("resource lease controller is unavailable")
	}
	fb, ok := c.broker.(resource.FenceableBroker)
	if !ok {
		return resource.Lease{}, fmt.Errorf("resource broker is not fencing-capable")
	}
	c.mu.Lock()
	previous := c.current
	c.mu.Unlock()
	fenced, err := fb.Fence(ctx, previous, token)
	if err != nil {
		return resource.Lease{}, err
	}
	c.mu.Lock()
	if normalizedLeaseRevision(c.current) != normalizedLeaseRevision(previous) {
		c.mu.Unlock()
		return resource.Lease{}, fmt.Errorf("resource lease controller changed while installing supervisor fence")
	}
	c.current = fenced
	c.mu.Unlock()
	return fenced, nil
}

func (c *leaseController) leasePointer() *resource.Lease {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	x := c.current
	return &x
}

func (c *leaseController) Current() resource.Lease {
	if c == nil {
		return resource.Lease{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

func (c *leaseController) FatalError() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fatalErr
}

func (c *leaseController) Refreshes() []resource.RefreshRecord {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]resource.RefreshRecord(nil), c.refreshes...)
}

func (c *leaseController) Supervision() []resource.SupervisionRecord {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]resource.SupervisionRecord(nil), c.supervision...)
}

func (c *leaseController) recordSupervision(state resource.SupervisionState, message string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	current := c.current
	c.supervision = append(c.supervision, resource.SupervisionRecord{
		State: state, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Revision: normalizedLeaseRevision(current), ExpiresAt: current.ExpiresAt, Message: message,
	})
	c.mu.Unlock()
}

func (c *leaseController) markFailure(err error) {
	if err == nil || c == nil {
		return
	}
	if c.req.LeasePolicy.WithDefaults().RenewalFailure == resource.RenewalFailClosed {
		c.mu.Lock()
		if c.fatalErr == nil {
			c.fatalErr = fmt.Errorf("resource lease policy fail-closed: %w", err)
		}
		c.mu.Unlock()
	}
}

func (c *leaseController) Renewals() []resource.RenewalRecord {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]resource.RenewalRecord(nil), c.renewals...)
}

func (c *leaseController) Renew(ctx context.Context) (resource.Lease, error) {
	if c == nil {
		return resource.Lease{}, fmt.Errorf("resource lease controller is unavailable")
	}
	rb, ok := c.broker.(resource.RenewableBroker)
	if !ok {
		err := fmt.Errorf("resource broker does not support lease renewal")
		c.markFailure(err)
		return resource.Lease{}, err
	}
	c.mu.Lock()
	previous := c.current
	c.mu.Unlock()
	renewed, err := rb.Renew(ctx, c.req, previous)
	if err != nil {
		c.markFailure(err)
		return resource.Lease{}, err
	}
	if err := resource.ValidateInPlaceRenewal(c.req, previous, renewed); err != nil {
		c.markFailure(err)
		return resource.Lease{}, err
	}
	renewedAt := time.Now().UTC().Format(time.RFC3339Nano)
	c.mu.Lock()
	// A concurrent renewal may have advanced the controller while the host broker
	// call was in flight. Do not silently overwrite a newer revision.
	if normalizedLeaseRevision(c.current) != normalizedLeaseRevision(previous) {
		c.mu.Unlock()
		err := fmt.Errorf("resource lease controller changed during renewal")
		c.markFailure(err)
		return resource.Lease{}, err
	}
	c.current = renewed
	c.renewals = append(c.renewals, resource.RenewalRecord{Revision: normalizedLeaseRevision(renewed), RenewedAt: renewedAt, ExpiresAt: renewed.ExpiresAt})
	c.mu.Unlock()
	return renewed, nil
}

func (c *leaseController) Refresh(ctx context.Context) (resource.Lease, error) {
	if c == nil {
		return resource.Lease{}, fmt.Errorf("resource lease controller is unavailable")
	}
	rb, ok := c.broker.(resource.RefreshableBroker)
	if !ok {
		err := fmt.Errorf("resource broker does not support live refresh")
		c.markFailure(err)
		return resource.Lease{}, err
	}
	c.mu.Lock()
	previous := c.current
	c.mu.Unlock()
	refreshed, err := rb.Refresh(ctx, c.req, previous)
	if err != nil {
		c.markFailure(err)
		return resource.Lease{}, err
	}
	if err := resource.ValidateRefresh(c.req, previous, refreshed); err != nil {
		c.markFailure(err)
		return resource.Lease{}, err
	}
	var scopes []string
	a, b := resource.NormalizeResolved(previous.Resources), resource.NormalizeResolved(refreshed.Resources)
	for i := range a {
		if a[i] != b[i] {
			scopes = append(scopes, a[i].Scope)
		}
	}
	c.mu.Lock()
	if normalizedLeaseRevision(c.current) != normalizedLeaseRevision(previous) {
		c.mu.Unlock()
		err := fmt.Errorf("resource lease controller changed during refresh")
		c.markFailure(err)
		return resource.Lease{}, err
	}
	c.current = refreshed
	c.refreshes = append(c.refreshes, resource.RefreshRecord{Revision: normalizedLeaseRevision(refreshed), RefreshedAt: time.Now().UTC().Format(time.RFC3339Nano), Scopes: scopes})
	c.mu.Unlock()
	return refreshed, nil
}
