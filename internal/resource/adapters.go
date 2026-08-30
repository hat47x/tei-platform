package resource

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"tei/internal/core"
	"tei/internal/permission"
)

// AdapterRequest is the host-only request handed to one kind-specific resource
// adapter. It still contains only logical identities; an adapter owns the
// concrete secret/mount/network lookup.
type AdapterRequest struct {
	Environment    core.CanonicalID
	CapabilityKind string
	CapabilityID   core.CanonicalID
	Permission     permission.Spec
	Binding        Binding
	LeaseID        string
	Revision       int
}

// Adapter is the narrow host boundary for one resource kind. Renew may rotate
// concrete handles/endpoints/mounts, but AdapterBroker later verifies that the
// logical ResourceID, permission and binding did not change.
type Adapter interface {
	Kind() permission.Kind
	Acquire(context.Context, AdapterRequest) (Resolved, error)
	Renew(context.Context, AdapterRequest, Resolved) (Resolved, error)
	Release(context.Context, AdapterRequest, Resolved) error
}

// RefreshableAdapter explicitly supports live replacement of a concrete
// resource view (for example a rotated opaque secret handle).
type RefreshableAdapter interface {
	Adapter
	Refresh(context.Context, AdapterRequest, Resolved) (Resolved, error)
}

type adapterLease struct {
	Request    AcquireRequest
	Lease      Lease
	AcquiredAt time.Time
}

// AdapterBroker composes kind-specific host adapters into one invocation lease.
// It is deliberately small: production adapters may wrap Vault/KMS, a service-
// mesh/network-policy controller and a mount namespace/runtime.
type AdapterBroker struct {
	Adapters []Adapter
	TTL      time.Duration
	Now      func() time.Time

	mu     sync.Mutex
	next   int
	active map[string]adapterLease
}

func (b *AdapterBroker) now() time.Time {
	if b.Now != nil {
		return b.Now()
	}
	return time.Now().UTC()
}

func (b *AdapterBroker) ttl() time.Duration {
	if b.TTL > 0 {
		return b.TTL
	}
	return 5 * time.Minute
}

func (b *AdapterBroker) byKind() (map[permission.Kind]Adapter, error) {
	m := map[permission.Kind]Adapter{}
	for _, a := range b.Adapters {
		if a == nil {
			continue
		}
		k := a.Kind()
		if k != permission.KindSecret && k != permission.KindNetwork && k != permission.KindFilesystem {
			return nil, fmt.Errorf("resource adapter declares unsupported kind %s", k)
		}
		if _, dup := m[k]; dup {
			return nil, fmt.Errorf("multiple resource adapters registered for kind %s", k)
		}
		m[k] = a
	}
	return m, nil
}

func (b *AdapterBroker) Acquire(ctx context.Context, req AcquireRequest) (Lease, error) {
	if err := ctx.Err(); err != nil {
		return Lease{}, err
	}
	req = NormalizeAcquireRequest(req)
	bindings, err := MatchBindings(req.Permissions, req.Bindings)
	if err != nil {
		return Lease{}, err
	}
	adapters, err := b.byKind()
	if err != nil {
		return Lease{}, err
	}
	bindingByKey := map[string]Binding{}
	for _, v := range bindings {
		bindingByKey[v.Key()] = v
	}

	b.mu.Lock()
	b.next++
	leaseID := fmt.Sprintf("lease-%06d", b.next)
	b.mu.Unlock()

	issued := b.now()
	policy := req.LeasePolicy.WithDefaults()
	if err := policy.Validate(); err != nil {
		return Lease{}, err
	}
	expires := issued.Add(b.ttl())
	deadline := issued.Add(policy.MaxLifetimeDuration())
	if expires.After(deadline) {
		expires = deadline
	}
	lease := Lease{ID: leaseID, Revision: 1, IssuedAt: issued.Format(time.RFC3339Nano), ExpiresAt: expires.Format(time.RFC3339Nano), RuntimePolicyFingerprint: req.RuntimePolicyFingerprint}
	acquired := make([]struct {
		a Adapter
		r AdapterRequest
		v Resolved
	}, 0, len(req.Permissions))
	for _, p := range req.Permissions {
		a, ok := adapters[p.Kind]
		if !ok {
			_ = releaseAcquired(context.WithoutCancel(ctx), acquired)
			return Lease{}, fmt.Errorf("host has no resource adapter for kind %s", p.Kind)
		}
		binding := bindingByKey[string(p.Kind)+"|"+p.Scope]
		ar := AdapterRequest{Environment: req.Environment, CapabilityKind: req.CapabilityKind, CapabilityID: req.CapabilityID, Permission: p, Binding: binding, LeaseID: leaseID, Revision: 1}
		resolved, err := a.Acquire(ctx, ar)
		if err != nil {
			_ = releaseAcquired(context.WithoutCancel(ctx), acquired)
			return Lease{}, fmt.Errorf("resource adapter %s acquire failed for %s: %w", p.Kind, p.Scope, err)
		}
		lease.Resources = append(lease.Resources, resolved)
		acquired = append(acquired, struct {
			a Adapter
			r AdapterRequest
			v Resolved
		}{a: a, r: ar, v: resolved})
	}
	if err := ValidateLease(req, lease); err != nil {
		_ = releaseAcquired(context.WithoutCancel(ctx), acquired)
		return Lease{}, err
	}
	b.mu.Lock()
	if b.active == nil {
		b.active = map[string]adapterLease{}
	}
	b.active[lease.ID] = adapterLease{Request: req, Lease: lease, AcquiredAt: issued}
	b.mu.Unlock()
	return lease, nil
}

func releaseAcquired(ctx context.Context, acquired []struct {
	a Adapter
	r AdapterRequest
	v Resolved
}) error {
	var errs []error
	for i := len(acquired) - 1; i >= 0; i-- {
		if err := acquired[i].a.Release(ctx, acquired[i].r, acquired[i].v); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (b *AdapterBroker) Renew(ctx context.Context, req AcquireRequest, previous Lease) (Lease, error) {
	if err := ctx.Err(); err != nil {
		return Lease{}, err
	}
	req = NormalizeAcquireRequest(req)
	b.mu.Lock()
	active, ok := b.active[previous.ID]
	b.mu.Unlock()
	if !ok {
		return Lease{}, fmt.Errorf("resource lease %s is not active", previous.ID)
	}
	if AcquireRequestFingerprint(active.Request) != AcquireRequestFingerprint(req) {
		return Lease{}, fmt.Errorf("resource lease %s renewal request differs from original capability/permission/resource policy", previous.ID)
	}
	if active.Lease.Revision != previous.Revision || active.Lease.ID != previous.ID {
		return Lease{}, fmt.Errorf("resource lease %s renewal uses stale revision %d; active revision is %d", previous.ID, previous.Revision, active.Lease.Revision)
	}
	if active.Lease.FencingToken != 0 && previous.FencingToken != active.Lease.FencingToken {
		return Lease{}, fmt.Errorf("resource lease %s renewal uses stale fencing token %d; active token is %d", previous.ID, previous.FencingToken, active.Lease.FencingToken)
	}
	policy := req.LeasePolicy.WithDefaults()
	if err := policy.Validate(); err != nil {
		return Lease{}, err
	}
	now := b.now()
	if previous.ExpiresAt != "" {
		exp, err := time.Parse(time.RFC3339, previous.ExpiresAt)
		if err != nil {
			return Lease{}, fmt.Errorf("invalid previous lease expiry: %w", err)
		}
		if now.Before(exp.Add(-policy.RenewBeforeDuration())) {
			return Lease{}, fmt.Errorf("resource lease %s renewal is too early; renew window opens at %s", previous.ID, exp.Add(-policy.RenewBeforeDuration()).Format(time.RFC3339))
		}
	}
	deadline := active.AcquiredAt.Add(policy.MaxLifetimeDuration())
	if !now.Before(deadline) {
		return Lease{}, fmt.Errorf("resource lease %s reached max lifetime at %s", previous.ID, deadline.Format(time.RFC3339))
	}
	adapters, err := b.byKind()
	if err != nil {
		return Lease{}, err
	}
	bindings, err := MatchBindings(req.Permissions, req.Bindings)
	if err != nil {
		return Lease{}, err
	}
	bindingByKey := map[string]Binding{}
	for _, v := range bindings {
		bindingByKey[v.Key()] = v
	}
	oldByKey := map[string]Resolved{}
	for _, v := range active.Lease.Resources {
		oldByKey[string(v.Kind)+"|"+v.Scope] = v
	}
	nextRevision := active.Lease.Revision + 1
	renewed := Lease{ID: active.Lease.ID, Revision: nextRevision, FencingToken: active.Lease.FencingToken, RuntimePolicyFingerprint: req.RuntimePolicyFingerprint}
	renewed.IssuedAt = active.Lease.IssuedAt
	nextExpiry := now.Add(b.ttl())
	if nextExpiry.After(deadline) {
		nextExpiry = deadline
	}
	if previous.ExpiresAt != "" {
		prevExp, _ := time.Parse(time.RFC3339, previous.ExpiresAt)
		if !nextExpiry.After(prevExp) {
			return Lease{}, fmt.Errorf("resource lease %s cannot be renewed beyond max lifetime", previous.ID)
		}
	}
	renewed.ExpiresAt = nextExpiry.Format(time.RFC3339Nano)
	for _, p := range req.Permissions {
		a, ok := adapters[p.Kind]
		if !ok {
			return Lease{}, fmt.Errorf("host has no resource adapter for kind %s", p.Kind)
		}
		binding := bindingByKey[string(p.Kind)+"|"+p.Scope]
		old := oldByKey[string(p.Kind)+"|"+p.Scope]
		ar := AdapterRequest{Environment: req.Environment, CapabilityKind: req.CapabilityKind, CapabilityID: req.CapabilityID, Permission: p, Binding: binding, LeaseID: previous.ID, Revision: nextRevision}
		v, err := a.Renew(ctx, ar, old)
		if err != nil {
			return Lease{}, fmt.Errorf("resource adapter %s renew failed for %s: %w", p.Kind, p.Scope, err)
		}
		renewed.Resources = append(renewed.Resources, v)
	}
	if err := ValidateInPlaceRenewal(req, active.Lease, renewed); err != nil {
		return Lease{}, err
	}
	b.mu.Lock()
	b.active[renewed.ID] = adapterLease{Request: req, Lease: renewed, AcquiredAt: active.AcquiredAt}
	b.mu.Unlock()
	return renewed, nil
}

func (b *AdapterBroker) Refresh(ctx context.Context, req AcquireRequest, previous Lease) (Lease, error) {
	if err := ctx.Err(); err != nil {
		return Lease{}, err
	}
	req = NormalizeAcquireRequest(req)
	b.mu.Lock()
	active, ok := b.active[previous.ID]
	b.mu.Unlock()
	if !ok {
		return Lease{}, fmt.Errorf("resource lease %s is not active", previous.ID)
	}
	if AcquireRequestFingerprint(active.Request) != AcquireRequestFingerprint(req) {
		return Lease{}, fmt.Errorf("resource lease %s refresh request differs from original capability/permission/resource policy", previous.ID)
	}
	if active.Lease.Revision != previous.Revision {
		return Lease{}, fmt.Errorf("resource lease %s refresh uses stale revision %d; active revision is %d", previous.ID, previous.Revision, active.Lease.Revision)
	}
	if active.Lease.FencingToken != 0 && previous.FencingToken != active.Lease.FencingToken {
		return Lease{}, fmt.Errorf("resource lease %s refresh uses stale fencing token %d; active token is %d", previous.ID, previous.FencingToken, active.Lease.FencingToken)
	}
	adapters, err := b.byKind()
	if err != nil {
		return Lease{}, err
	}
	bindings, err := MatchBindings(req.Permissions, req.Bindings)
	if err != nil {
		return Lease{}, err
	}
	bindingByKey := map[string]Binding{}
	for _, v := range bindings {
		bindingByKey[v.Key()] = v
	}
	oldByKey := map[string]Resolved{}
	for _, v := range active.Lease.Resources {
		oldByKey[string(v.Kind)+"|"+v.Scope] = v
	}
	nextRevision := active.Lease.Revision + 1
	refreshed := Lease{ID: active.Lease.ID, Revision: nextRevision, FencingToken: active.Lease.FencingToken, IssuedAt: active.Lease.IssuedAt, ExpiresAt: active.Lease.ExpiresAt, RuntimePolicyFingerprint: req.RuntimePolicyFingerprint}
	for _, p := range req.Permissions {
		binding := bindingByKey[string(p.Kind)+"|"+p.Scope]
		old := oldByKey[string(p.Kind)+"|"+p.Scope]
		if binding.Refresh != RefreshLive {
			refreshed.Resources = append(refreshed.Resources, old)
			continue
		}
		a := adapters[p.Kind]
		ra, ok := a.(RefreshableAdapter)
		if !ok {
			return Lease{}, fmt.Errorf("resource adapter %s does not support live refresh for %s", p.Kind, p.Scope)
		}
		ar := AdapterRequest{Environment: req.Environment, CapabilityKind: req.CapabilityKind, CapabilityID: req.CapabilityID, Permission: p, Binding: binding, LeaseID: previous.ID, Revision: nextRevision}
		v, err := ra.Refresh(ctx, ar, old)
		if err != nil {
			return Lease{}, fmt.Errorf("resource adapter %s refresh failed for %s: %w", p.Kind, p.Scope, err)
		}
		refreshed.Resources = append(refreshed.Resources, v)
	}
	if err := ValidateRefresh(req, active.Lease, refreshed); err != nil {
		return Lease{}, err
	}
	b.mu.Lock()
	b.active[refreshed.ID] = adapterLease{Request: req, Lease: refreshed, AcquiredAt: active.AcquiredAt}
	b.mu.Unlock()
	return refreshed, nil
}

// Fence installs a newer supervisor fencing token without changing the lease
// revision or logical authority. Tokens are monotonic and a stale owner can
// never move the fence backwards.
func (b *AdapterBroker) Fence(ctx context.Context, lease Lease, token uint64) (Lease, error) {
	if err := ctx.Err(); err != nil {
		return Lease{}, err
	}
	if token == 0 {
		return Lease{}, fmt.Errorf("resource lease fencing token must be positive")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	active, ok := b.active[lease.ID]
	if !ok {
		return Lease{}, fmt.Errorf("resource lease %s is not active", lease.ID)
	}
	if lease.Revision != 0 && lease.Revision != active.Lease.Revision {
		return Lease{}, fmt.Errorf("resource lease %s fence uses stale revision %d; active revision is %d", lease.ID, lease.Revision, active.Lease.Revision)
	}
	if token <= active.Lease.FencingToken {
		if token == active.Lease.FencingToken {
			return active.Lease, nil
		}
		return Lease{}, fmt.Errorf("resource lease %s cannot move fencing token backwards from %d to %d", lease.ID, active.Lease.FencingToken, token)
	}
	active.Lease.FencingToken = token
	b.active[lease.ID] = active
	return active.Lease, nil
}

func (b *AdapterBroker) Release(ctx context.Context, lease Lease) error {
	b.mu.Lock()
	active, ok := b.active[lease.ID]
	b.mu.Unlock()
	if !ok {
		return fmt.Errorf("resource lease %s is not active", lease.ID)
	}
	if lease.Revision != 0 && lease.Revision != active.Lease.Revision {
		return fmt.Errorf("resource lease %s release uses stale revision %d; active revision is %d", lease.ID, lease.Revision, active.Lease.Revision)
	}
	if active.Lease.FencingToken != 0 && lease.FencingToken != active.Lease.FencingToken {
		return fmt.Errorf("resource lease %s release uses stale fencing token %d; active token is %d", lease.ID, lease.FencingToken, active.Lease.FencingToken)
	}
	adapters, err := b.byKind()
	if err != nil {
		return err
	}
	bindings, err := MatchBindings(active.Request.Permissions, active.Request.Bindings)
	if err != nil {
		return err
	}
	bindingByKey := map[string]Binding{}
	for _, v := range bindings {
		bindingByKey[v.Key()] = v
	}
	perms := permission.Normalize(active.Request.Permissions)
	var errs []error
	for i := len(perms) - 1; i >= 0; i-- {
		p := perms[i]
		a := adapters[p.Kind]
		binding := bindingByKey[string(p.Kind)+"|"+p.Scope]
		var resolved Resolved
		for _, v := range active.Lease.Resources {
			if v.Kind == p.Kind && v.Scope == p.Scope {
				resolved = v
				break
			}
		}
		ar := AdapterRequest{Environment: active.Request.Environment, CapabilityKind: active.Request.CapabilityKind, CapabilityID: active.Request.CapabilityID, Permission: p, Binding: binding, LeaseID: active.Lease.ID, Revision: active.Lease.Revision}
		if err := a.Release(ctx, ar, resolved); err != nil {
			errs = append(errs, fmt.Errorf("resource adapter %s release failed for %s: %w", p.Kind, p.Scope, err))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	b.mu.Lock()
	delete(b.active, lease.ID)
	b.mu.Unlock()
	return nil
}

func (b *AdapterBroker) ActiveLeaseCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.active)
}

func (b *AdapterBroker) ActiveLease(id string) (Lease, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	v, ok := b.active[id]
	return v.Lease, ok
}

// StaticResource is a non-secret reference fixture used by the reference host.
// SecretHandle is still opaque; no secret value is stored here.
type StaticResource struct {
	ResourceID        core.CanonicalID
	SecretHandle      string
	NetworkEndpoint   string
	MountSourceHandle string
	MountPath         string
}

// StaticAdapter is intentionally a test/reference adapter. It demonstrates the
// Adapter boundary without claiming to be a production Vault/network/mount
// implementation.
type StaticAdapter struct {
	ResourceKind  permission.Kind
	Entries       []StaticResource
	RenewSuffix   string
	RefreshSuffix string
}

func (a *StaticAdapter) Kind() permission.Kind { return a.ResourceKind }

func (a *StaticAdapter) entry(id core.CanonicalID) (StaticResource, bool) {
	for _, e := range a.Entries {
		if e.ResourceID == id {
			return e, true
		}
	}
	return StaticResource{}, false
}

func (a *StaticAdapter) resolved(req AdapterRequest, suffix string) (Resolved, error) {
	e, ok := a.entry(req.Binding.ResourceID)
	if !ok {
		return Resolved{}, fmt.Errorf("resource %s is not configured", req.Binding.ResourceID)
	}
	r := Resolved{Kind: req.Permission.Kind, Access: req.Permission.Access, Scope: req.Permission.Scope, ResourceID: req.Binding.ResourceID, SecretDelivery: req.Binding.SecretDelivery, SecretTarget: req.Binding.SecretTarget}
	if r.Kind == permission.KindSecret && r.SecretDelivery == "" {
		r.SecretDelivery = SecretDeliveryHandle
	}
	switch a.ResourceKind {
	case permission.KindSecret:
		r.SecretHandle = e.SecretHandle + suffix
	case permission.KindNetwork:
		r.NetworkEndpoint = e.NetworkEndpoint
	case permission.KindFilesystem:
		r.MountSourceHandle = e.MountSourceHandle
		r.MountPath = e.MountPath
	default:
		return Resolved{}, fmt.Errorf("unsupported static adapter kind %s", a.ResourceKind)
	}
	return r, nil
}

func (a *StaticAdapter) Acquire(_ context.Context, req AdapterRequest) (Resolved, error) {
	return a.resolved(req, "")
}
func (a *StaticAdapter) Renew(_ context.Context, _ AdapterRequest, previous Resolved) (Resolved, error) {
	return previous, nil
}
func (a *StaticAdapter) Refresh(_ context.Context, req AdapterRequest, _ Resolved) (Resolved, error) {
	suffix := a.RefreshSuffix
	if suffix == "" {
		suffix = "#refresh-"
	}
	return a.resolved(req, fmt.Sprintf("%s%d", suffix, req.Revision))
}
func (a *StaticAdapter) Release(context.Context, AdapterRequest, Resolved) error { return nil }

// StableRecords is useful to compare a lease before/after renewal while hiding
// rotating concrete handles/endpoints/mounts.
func StableRecords(l Lease) []Record {
	out := Records(l.Resources)
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}
