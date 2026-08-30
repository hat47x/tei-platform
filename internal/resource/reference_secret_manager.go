package resource

import (
	"context"
	"fmt"
	"sync"

	"tei/internal/core"
	"tei/internal/permission"
)

// ReferenceSecretManagerAdapter is a concrete, in-memory reference host adapter
// for the secret resource contract. It models a secret-manager that exposes only
// opaque handles to TEI. It is intentionally not a production Vault/KMS client.
// Renew preserves the currently injected handle; Refresh advances the opaque
// handle generation while retaining the same logical ResourceID.
type ReferenceSecretManagerAdapter struct {
	mu      sync.Mutex
	Handles map[core.CanonicalID]string
}

func (a *ReferenceSecretManagerAdapter) Kind() permission.Kind { return permission.KindSecret }

func (a *ReferenceSecretManagerAdapter) base(id core.CanonicalID) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	v := a.Handles[id]
	if v == "" {
		return "", fmt.Errorf("reference secret manager has no opaque handle for %s", id)
	}
	return v, nil
}

func (a *ReferenceSecretManagerAdapter) resolved(req AdapterRequest, handle string) Resolved {
	d := req.Binding.SecretDelivery
	if d == "" {
		d = SecretDeliveryHandle
	}
	return Resolved{Kind: permission.KindSecret, Access: req.Permission.Access, Scope: req.Permission.Scope, ResourceID: req.Binding.ResourceID, SecretHandle: handle, SecretDelivery: d, SecretTarget: req.Binding.SecretTarget}
}

func (a *ReferenceSecretManagerAdapter) Acquire(_ context.Context, req AdapterRequest) (Resolved, error) {
	h, err := a.base(req.Binding.ResourceID)
	if err != nil {
		return Resolved{}, err
	}
	return a.resolved(req, h), nil
}

func (a *ReferenceSecretManagerAdapter) Renew(_ context.Context, _ AdapterRequest, previous Resolved) (Resolved, error) {
	return previous, nil
}

func (a *ReferenceSecretManagerAdapter) Refresh(_ context.Context, req AdapterRequest, previous Resolved) (Resolved, error) {
	base, err := a.base(req.Binding.ResourceID)
	if err != nil {
		return Resolved{}, err
	}
	// The generation is opaque host metadata; no secret value is exposed.
	h := fmt.Sprintf("%s#generation-%d", base, req.Revision)
	if h == previous.SecretHandle {
		return Resolved{}, fmt.Errorf("reference secret refresh did not advance opaque handle")
	}
	return a.resolved(req, h), nil
}

func (a *ReferenceSecretManagerAdapter) Release(context.Context, AdapterRequest, Resolved) error {
	return nil
}
