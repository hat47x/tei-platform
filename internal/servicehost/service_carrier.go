package servicehost

import (
	"context"
	"encoding/json"
	"errors"

	"tei/internal/composition"
	pruntime "tei/internal/plugin/runtime"
)

var (
	ErrServiceCarrierGenerationInvalid = errors.New("service generation invocation requires Host coordinator and route source")
	ErrServiceCarrierGenerationMismatch = errors.New("selected service differs from pinned Host generation")
	ErrServiceCarrierGenerationUnavailable = errors.New("Host generation service route unavailable")
)

// SelectedServiceCarrierRoute is a HOST-OWNED mapping from an existing pinned
// Generation binding (logical Provider ID) to its exact physical Service
// selection and Runtime Environment Plan. It is not a new resolver, accepted
// Contract, Conformance issuer or execution permission.
type SelectedServiceCarrierRoute struct {
	Binding   composition.Binding
	Selection pruntime.ServiceProviderSelection
	Plan      pruntime.EnvironmentPlan
}

// ServiceCarrierRouteSource must consult the Host's ready routes for the
// PINNED generation, not look up the newest Provider on every invocation.
// It runs outside the Router's mutex and may perform I/O. Host ownership,
// revision checks, and immutability of returned Plan data are its responsibility.
type ServiceCarrierRouteSource func(context.Context, composition.Binding) (SelectedServiceCarrierRoute, error)

// SelectedServiceCarrierInvoker composes the existing Host generation lease,
// the exact generation-scoped route and the product Runtime's current
// Service Contract/Provider acceptance and execution authority.
//
// It does not authenticate a request, issue a Contract, or perform domain
// request/result shape verification. The caller must be an authorized Host
// entry and must not expose its RouteSource or Registry to an untrusted plugin.
type SelectedServiceCarrierInvoker struct {
	coordinator *composition.Coordinator
	runtime     pruntime.Registry
	routes      ServiceCarrierRouteSource
}

// NewSelectedServiceCarrierInvoker keeps the Coordinator and the route source
// Host-owned. Each invocation uses Coordinator.Acquire rather than a caller-
// supplied lease, so callers cannot fabricate a generation number.
func NewSelectedServiceCarrierInvoker(
	coordinator *composition.Coordinator,
	runtime pruntime.Registry,
	routes ServiceCarrierRouteSource,
) (*SelectedServiceCarrierInvoker, error) {
	if coordinator == nil || routes == nil {
		return nil, ErrServiceCarrierGenerationInvalid
	}
	return &SelectedServiceCarrierInvoker{
		coordinator: coordinator, runtime: runtime, routes: routes,
	}, nil
}

type SelectedServiceCarrierInvocation struct {
	Generation Binding                           `json:"generation"`
	Service    pruntime.ServiceCarrierInvocation `json:"service"`
}

// Invoke pins one Host generation for the entire Service invocation, including
// its current acceptance, artifact verification, permissions, resource lease,
// and process execution. Cutover retires the old generation but cannot silently
// replace the Provider of an in-flight invocation.
func (i *SelectedServiceCarrierInvoker) Invoke(
	ctx context.Context, requestID string, payload json.RawMessage,
) (SelectedServiceCarrierInvocation, error) {
	if i == nil || i.coordinator == nil || i.routes == nil || ctx == nil {
		return SelectedServiceCarrierInvocation{}, ErrServiceCarrierGenerationInvalid
	}
	if err := ctx.Err(); err != nil {
		return SelectedServiceCarrierInvocation{}, err
	}
	lease, err := i.coordinator.Acquire()
	if err != nil {
		return SelectedServiceCarrierInvocation{}, err
	}
	defer lease.Release()
	if err := ctx.Err(); err != nil {
		return SelectedServiceCarrierInvocation{}, err
	}

	route, err := i.routes(ctx, lease.Binding)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return SelectedServiceCarrierInvocation{}, ctxErr
	}
	if err != nil {
		// Route lookup can disclose private Host paths or internal authority
		// metadata; do not expose the cause to the invoking consumer.
		return SelectedServiceCarrierInvocation{}, ErrServiceCarrierGenerationUnavailable
	}
	if route.Binding != lease.Binding ||
		route.Selection.ContractID != lease.Binding.ContractID ||
		route.Selection.ContractVersion != lease.Binding.ContractVersion ||
		route.Selection.ArtifactDigest != lease.Binding.ArtifactDigest {
		return SelectedServiceCarrierInvocation{}, ErrServiceCarrierGenerationMismatch
	}
	// Logical composition ProviderID need not equal a physical plugin ID.
	// Exact matching of route.Binding includes the Host-selected logical
	// ProviderID, while Runtime verifies physical Plugin/Implementation/Digest.
	result, err := i.runtime.InvokeSelectedServiceCarrier(
		ctx, route.Plan, route.Selection, requestID, payload,
	)
	if err != nil {
		return SelectedServiceCarrierInvocation{}, err
	}
	return SelectedServiceCarrierInvocation{
		Generation: lease.Binding, Service: result,
	}, nil
}
