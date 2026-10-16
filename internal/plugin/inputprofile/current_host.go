package inputprofile

import (
	"context"
	"errors"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

var (
	ErrCurrentPackageSelectionInvalid     = errors.New("current Plugin package selection requires a trusted Host source and scope")
	ErrCurrentPackageSelectionUnavailable = errors.New("current Host Plugin package selection unavailable")
	ErrCurrentPackageSelectionScope       = errors.New("current Host Plugin package selection differs from requested Host scope")
)

// CurrentPackageBuildBinding is a read-only projection of a CURRENT Host
// choice, not a signature or a new acceptance decision. Selection.Profile
// must be explicit; a missing value is not interpreted as a v2 opt-in.
type CurrentPackageBuildBinding struct {
	Application core.CanonicalID
	Environment core.CanonicalID
	Selection   BuildInputSelection
}

// CurrentPackageBuildSource must be supplied by an authenticated Host, not
// exposed to callers or built from request-supplied fingerprints. The Host
// implementation must check the CURRENT Application entitlement, Source/Plan
// locks, and revocation on every read. This callback does not establish those
// policies or authorize building, executing or accepting the output.
type CurrentPackageBuildSource func(context.Context, core.CanonicalID, core.CanonicalID) (CurrentPackageBuildBinding, error)

// PrepareCurrentHostPackageInputs consults the Host-owned current selection
// before calling the existing opt-in input adapter. It never silently chooses
// v2, falls back to another Source, or copies a caller-supplied lock into Host
// authority. The Source and Plan supplied by the Host must still be pinned
// through its actual execution; a read followed by execution is not an atomic
// trust or revocation guard.
func PrepareCurrentHostPackageInputs(
	ctx context.Context,
	application, environment core.CanonicalID,
	current CurrentPackageBuildSource,
	source transform.Source,
	plan plugin.BuildPlan,
	rawDefinition []byte,
) (PreparedPackageInputs, error) {
	if ctx == nil || current == nil {
		return PreparedPackageInputs{}, ErrCurrentPackageSelectionInvalid
	}
	if err := ctx.Err(); err != nil {
		return PreparedPackageInputs{}, err
	}
	// CanonicalID is a Go string alias, so callers can supply arbitrary or
	// whitespace-padded values without using core.ParseCanonicalID. Do not
	// pass malformed Host scope identifiers into the trusted Host callback.
	for _, id := range []core.CanonicalID{application, environment} {
		parsed, err := core.ParseCanonicalID(string(id))
		if err != nil || parsed != id {
			return PreparedPackageInputs{}, ErrCurrentPackageSelectionInvalid
		}
	}
	if err := ctx.Err(); err != nil {
		return PreparedPackageInputs{}, err
	}
	binding, err := current(ctx, application, environment)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return PreparedPackageInputs{}, ctxErr
	}
	if err != nil {
		// Host errors may include secret policy details or internal paths.
		return PreparedPackageInputs{}, ErrCurrentPackageSelectionUnavailable
	}
	if binding.Application != application || binding.Environment != environment {
		return PreparedPackageInputs{}, ErrCurrentPackageSelectionScope
	}
	return PreparePluginPackageInputs(binding.Selection, source, plan, rawDefinition)
}
