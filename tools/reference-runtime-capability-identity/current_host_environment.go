package main

import (
	"context"
	"fmt"

	"tei/internal/core"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
)

// currentRuntimeEnvironmentAuthorityReader is supplied by the trusted Host.
// It must consult the existing CURRENT Environment Authority source. The
// project Plugin and its request may not implement or choose this reader.
// This reference adapter is not a second grant store or an issuer validator.
type currentRuntimeEnvironmentAuthorityReader func(
	context.Context, core.CanonicalID,
) (RuntimeEnvironmentAuthority, error)

// invokeWithCurrentHostEnvironment is a narrowly scoped #976 pressure test:
// it re-reads current Host-owned Environment Authority before using the
// existing plan verifier and runtime invocation bridge. Both decisions use
// the SAME Runtime Capability Ref, without inventing a new permission key.
//
// This is NOT an atomic revocation/launch fence. Authority changes AFTER
// the read may still race with launch. A production Host must acquire/hold
// the appropriate current lease or generation gate across the launch boundary,
// using existing runtime mechanisms rather than treating this snapshot as a
// transferable authority token.
func invokeWithCurrentHostEnvironment(
	ctx context.Context,
	hostEnvironment core.CanonicalID,
	plan RuntimeEnvironmentPlan,
	profile ExecutionProfile,
	readCurrent currentRuntimeEnvironmentAuthorityReader,
	realization RuntimeProcessRealization,
	verifier pruntime.ArtifactVerifier,
	broker resource.Broker,
	sandbox pruntime.SandboxLauncher,
	operation, requestID string,
	payload any,
) (GenericInvocation, error) {
	if ctx == nil {
		return GenericInvocation{}, fmt.Errorf("current Host Environment invocation requires context")
	}
	if err := ctx.Err(); err != nil {
		return GenericInvocation{}, err
	}
	if hostEnvironment == "" || plan.Environment != hostEnvironment {
		return GenericInvocation{}, fmt.Errorf("runtime plan Environment differs from trusted Host Environment")
	}
	if readCurrent == nil {
		return GenericInvocation{}, fmt.Errorf("current Host Environment Authority reader is missing")
	}
	current, err := readCurrent(ctx, hostEnvironment)
	if err != nil {
		return GenericInvocation{}, fmt.Errorf("read current Host Environment Authority: %w", err)
	}
	if current.Environment != hostEnvironment {
		return GenericInvocation{}, fmt.Errorf("current Authority belongs to a different Host Environment")
	}
	if err := verifyRuntimeEnvironmentPlan(
		plan, plan.RuntimeCapability, profile, current,
	); err != nil {
		return GenericInvocation{}, fmt.Errorf("current Host Environment Authority refused runtime plan: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return GenericInvocation{}, err
	}
	return invokeWithExistingHostBoundary(
		ctx, plan, realization, verifier, broker, sandbox,
		operation, requestID, payload,
	)
}
