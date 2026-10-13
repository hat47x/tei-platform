// Package semanticbuildbridge is a read-only proof that an existing Plugin
// BuildArtifactLock can bind a data-only semantic export without granting
// Runtime Plugin Authority. It is not a production asset format or loader.
package semanticbuildbridge

import (
	"errors"
	"fmt"

	"tei/internal/plugin"
	"tei/internal/semantic"
	semanticdistribution "tei/reference/semantic-distribution"
)

// ReferenceOutputKind is a trial BuildOutputKind, not a new Core or Runtime
// CapabilityKind. The existing BuildPlan/BuildArtifactLock types accept
// independent builder-defined output kinds.
const ReferenceOutputKind plugin.BuildOutputKind = "semantic-export-reference"

// ReferenceTarget prevents a runtime-target build from being read as data.
// This label is local to this proof and does not establish a supported target.
const ReferenceTarget = "data-only"

// InspectLockedExport verifies an explicitly supplied exact build selection,
// then inspects its data-only semantic bytes. No provider discovery, runtime
// activation, signature trust, source ownership or project Acceptance happens.
//
// The caller MUST independently establish the trust of the BuildArtifactLock,
// BuildPlan, builder and artifact bytes. A matching set of self-declared values
// alone is not a cryptographic attestation of an authorized distribution.
func InspectLockedExport(
	plan plugin.BuildPlan,
	lock plugin.BuildArtifactLock,
	selected plugin.LockedBuild,
	payload []byte,
	expected semanticdistribution.Expectation,
) (semantic.Namespace, error) {
	if err := plan.Validate(); err != nil {
		return semantic.Namespace{}, fmt.Errorf("invalid build plan: %w", err)
	}
	if err := lock.Validate(); err != nil {
		return semantic.Namespace{}, fmt.Errorf("invalid build artifact lock: %w", err)
	}
	if selected.OutputKind != ReferenceOutputKind || selected.Target != ReferenceTarget {
		return semantic.Namespace{}, errors.New("selected build is not a data-only semantic reference output")
	}
	if plan.OutputKind != ReferenceOutputKind || plan.Target != ReferenceTarget {
		return semantic.Namespace{}, errors.New("build plan is not a data-only semantic reference output")
	}
	// A specific selector must be present verbatim in the existing lock. Do
	// not fall back to an older revision or the numerically highest version.
	count := 0
	for _, entry := range lock.Builds {
		if entry == selected {
			count++
		}
	}
	if count != 1 {
		return semantic.Namespace{}, errors.New("exact selected build is not present in build artifact lock")
	}
	// Validate both the pinned build inputs and actual bytes using existing
	// plugin functions, without introducing another artifact registry or lock.
	bound, err := plugin.LockBuildOutput(plan, plugin.BuildOutputDigest(payload))
	if err != nil {
		return semantic.Namespace{}, fmt.Errorf("build output binding: %w", err)
	}
	if bound != selected {
		return semantic.Namespace{}, errors.New("selected build plan or artifact digest mismatch")
	}
	if string(selected.Plugin.ID) != expected.Package.PackageID ||
		selected.Plugin.Version != expected.Package.Version {
		return semantic.Namespace{}, errors.New("selected build package identity mismatch")
	}
	if selected.ArtifactDigest != expected.ArtifactDigest {
		return semantic.Namespace{}, errors.New("selected build and semantic export artifact digests differ")
	}
	return semanticdistribution.InspectSelectedReference(payload, expected)
}
