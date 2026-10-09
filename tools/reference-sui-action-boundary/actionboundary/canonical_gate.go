package actionboundary

import (
 "context"
 "fmt"

 "tei/internal/application"
 "tei/internal/core"
 "tei/internal/driver"
)

// PatternResolver is the narrow TEI canonical lookup used by the reference host.
// It checks the application's declared command intent; it does not authorize
// the caller or execute the capability.
type PatternResolver interface {
 ResolvePattern(core.CanonicalID, core.CanonicalID) (driver.PatternResolution, error)
}

// CanonicalCommandGate rejects an action that is not an Application-owned
// command pattern. The ActionID must be the canonical Pattern ID.
// This gate must be combined with a runtime Authorizer before mutation.
type CanonicalCommandGate struct {
 Resolver PatternResolver
}

func (g CanonicalCommandGate) Check(_ context.Context, in Intent) error {
 if g.Resolver == nil { return fmt.Errorf("%w: canonical resolver unavailable", ErrDenied) }
 resolution, err := g.Resolver.ResolvePattern(core.CanonicalID(in.ApplicationID),core.CanonicalID(in.ActionID))
 if err != nil { return fmt.Errorf("%w: command binding: %v",ErrDenied,err) }
 if resolution.Application.ID != core.CanonicalID(in.ApplicationID) ||
  resolution.Pattern.ID != core.CanonicalID(in.ActionID) ||
  resolution.Pattern.Kind != application.PatternCommand ||
  resolution.Capability == nil {
  return fmt.Errorf("%w: action is not a resolved command",ErrDenied)
 }
 return nil
}
