// Package execution composes the driver with the checks that guard an
// invocation. It sits outside the driver so the driver does not depend on the
// operation surface.
package execution

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/operations"
)

// Invoker is the part of the driver resolver this package needs.
type Invoker interface {
	InvokeCapability(context.Context, core.CanonicalID, core.CanonicalID, driver.BindingAdapterResolver, any) (driver.CapabilityInvocation, error)
}

// PeriodGuard holds what is needed to refuse a capability whose period is not
// open. A nil Source disables the guard: the caller has no state, so nothing is
// refused by assumption.
type PeriodGuard struct {
	Surface operations.OperationSurface
	Source  operations.PeriodStateSource
}

// Invoke runs the capability unless its period refuses it. A refusal is returned
// as a value (the invocation is not attempted); an error from the state source is
// returned as an error, because the guard could not be evaluated.
func Invoke(ctx context.Context, invoker Invoker, guard PeriodGuard, applicationID, capabilityID core.CanonicalID, adapters driver.BindingAdapterResolver, input any) (driver.CapabilityInvocation, *operations.PeriodRejection, error) {
	if guard.Source != nil {
		rejection, refused, err := operations.GuardCapabilityWith(ctx, guard.Source, guard.Surface, string(applicationID), capabilityID)
		if err != nil {
			return driver.CapabilityInvocation{}, nil, fmt.Errorf("evaluate the period state: %w", err)
		}
		if refused {
			return driver.CapabilityInvocation{}, &rejection, nil
		}
	}
	invocation, err := invoker.InvokeCapability(ctx, applicationID, capabilityID, adapters, input)
	if err != nil {
		return driver.CapabilityInvocation{}, nil, err
	}
	return invocation, nil, nil
}
