package resource

import (
	"context"
	"fmt"
	"sync"

	"tei/internal/core"
	"tei/internal/permission"
)

// CatalogEntry is a host-side resource mapping. It contains no secret value:
// SecretHandle is an opaque broker/vault reference. NetworkEndpoint and
// MountPath are the concrete least-authority views exposed for one lease.
type CatalogEntry struct {
	ResourceID      core.CanonicalID `json:"resourceId"`
	Kind            permission.Kind  `json:"kind"`
	SecretHandle    string           `json:"secretHandle,omitempty"`
	NetworkEndpoint string           `json:"networkEndpoint,omitempty"`
	MountPath       string           `json:"mountPath,omitempty"`
}

// CatalogBroker is a small host-side broker useful for local execution and
// conformance tests. Production hosts may replace it with Vault/KMS, mount
// namespace, service-mesh, or policy-engine backed brokers without changing TEI
// semantic definitions.
type CatalogBroker struct {
	Entries []CatalogEntry

	mu     sync.Mutex
	next   int
	active map[string]Lease
}

func (b *CatalogBroker) Acquire(ctx context.Context, req AcquireRequest) (Lease, error) {
	if err := ctx.Err(); err != nil {
		return Lease{}, err
	}
	bindings, err := MatchBindings(req.Permissions, req.Bindings)
	if err != nil {
		return Lease{}, err
	}
	entryByID := map[core.CanonicalID]CatalogEntry{}
	for _, e := range b.Entries {
		if _, dup := entryByID[e.ResourceID]; dup {
			return Lease{}, fmt.Errorf("host resource catalog contains duplicate id %s", e.ResourceID)
		}
		entryByID[e.ResourceID] = e
	}
	bindingByKey := map[string]Binding{}
	for _, binding := range bindings {
		bindingByKey[binding.Key()] = binding
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.next++
	lease := Lease{ID: fmt.Sprintf("lease-%06d", b.next), Revision: 1, RuntimePolicyFingerprint: req.RuntimePolicyFingerprint}
	for _, p := range permission.Normalize(req.Permissions) {
		binding := bindingByKey[string(p.Kind)+"|"+p.Scope]
		entry, ok := entryByID[binding.ResourceID]
		if !ok {
			return Lease{}, fmt.Errorf("host resource %s is not configured", binding.ResourceID)
		}
		if entry.Kind != p.Kind {
			return Lease{}, fmt.Errorf("host resource %s kind %s does not match permission kind %s", binding.ResourceID, entry.Kind, p.Kind)
		}
		r := Resolved{Kind: p.Kind, Access: p.Access, Scope: p.Scope, ResourceID: binding.ResourceID, SecretHandle: entry.SecretHandle, SecretDelivery: binding.SecretDelivery, SecretTarget: binding.SecretTarget, NetworkEndpoint: entry.NetworkEndpoint, MountPath: entry.MountPath}
		lease.Resources = append(lease.Resources, r)
	}
	if err := ValidateLease(req, lease); err != nil {
		return Lease{}, err
	}
	if b.active == nil {
		b.active = map[string]Lease{}
	}
	b.active[lease.ID] = lease
	return lease, nil
}

func (b *CatalogBroker) Release(ctx context.Context, lease Lease) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.active == nil {
		return fmt.Errorf("resource lease %s is not active", lease.ID)
	}
	if _, ok := b.active[lease.ID]; !ok {
		return fmt.Errorf("resource lease %s is not active", lease.ID)
	}
	delete(b.active, lease.ID)
	return nil
}

func (b *CatalogBroker) ActiveLeaseCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.active)
}
