package servicehost

import (
	"context"
	"errors"
	"strings"

	"tei/internal/composition"
	"tei/internal/core"
)

// CurrentApplicationServiceRoute is a Host-owned read of its existing current
// deployment/route source. The Application and Environment arguments are fixed
// by the Host at composition time, never inferred from a Service request.
// This callback must enforce Application entitlement and current revision; the
// adapter does not create a second resolver, issuer or acceptance authority.
type CurrentApplicationServiceRoute func(
	context.Context, core.CanonicalID, core.CanonicalID, composition.Binding,
) (SelectedServiceCarrierRoute, error)

// BindHostApplicationEnvironmentRoute creates a ServiceCarrierRouteSource for
// one trusted Host-selected Application/Environment pair. It consults the
// supplied current source on every invocation, keeps browser-supplied identity
// out of route selection, and rejects an EnvironmentPlan from another
// Environment before Service acceptance, verification or process invocation.
//
// The Host MUST authenticate and authorize its Action before invoking the
// resulting service invoker. This is not proof of Application entitlement:
// the supplied current source remains responsible for checking its existing
// authoritative Application/Deployment binding and generation.
func BindHostApplicationEnvironmentRoute(
	application, environment core.CanonicalID,
	current CurrentApplicationServiceRoute,
) (ServiceCarrierRouteSource, error) {
	if strings.TrimSpace(string(application)) == "" ||
		strings.TrimSpace(string(environment)) == "" || current == nil {
		return nil, ErrServiceCarrierGenerationInvalid
	}
	return func(ctx context.Context, pinned composition.Binding) (SelectedServiceCarrierRoute, error) {
		if ctx == nil {
			return SelectedServiceCarrierRoute{}, ErrServiceCarrierGenerationInvalid
		}
		if err := ctx.Err(); err != nil {
			return SelectedServiceCarrierRoute{}, err
		}
		route, err := current(ctx, application, environment, pinned)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return SelectedServiceCarrierRoute{}, ctxErr
		}
		if err != nil {
			return SelectedServiceCarrierRoute{}, err
		}
		if route.Plan.Environment != environment {
			return SelectedServiceCarrierRoute{}, errors.New("Host route Environment differs from trusted Host selection")
		}
		return route, nil
	}, nil
}
