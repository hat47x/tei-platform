// Package semanticbuildbridge is an isolated, read-only reference that adds
// BuildPlan preflight to the existing semanticdistribution locked-asset reader.
// It does not implement a new asset store, selector, Plugin ABI, or Runtime.
package semanticbuildbridge

import (
	"errors"
	"fmt"

	"tei/internal/plugin"
	"tei/internal/semantic"
	semanticdistribution "tei/reference/semantic-distribution"
)

// ReferenceTarget constrains the data-only test fixture's BuildPlan to a
// non-executable target. It is not a production target or contract.
const ReferenceTarget = "semantic/reference"

// InspectLockedExport validates the exact build input snapshot against the
// explicitly selected LockedBuild, then delegates artifact/lock/semantic
// verification to the single existing semanticdistribution adapter.
//
// The caller must independently establish trusted builder/lock provenance,
// artifact availability and project Acceptance. This does not grant execution
// authority and does not infer source ownership from an artifact.
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
	if selected.OutputKind != semanticdistribution.SemanticReferenceOutputKind || selected.Target != ReferenceTarget {
		return semantic.Namespace{}, errors.New("selected build is not a data-only semantic reference output")
	}
	if plan.OutputKind != semanticdistribution.SemanticReferenceOutputKind || plan.Target != ReferenceTarget {
		return semantic.Namespace{}, errors.New("build plan is not a data-only semantic reference output")
	}
	// The existing BuildArtifactLock does not itself retain input plan bytes.
	// Before reading any artifact, compare the selected exact build binding to
	// a fresh fingerprint of the claimed plan and the actual supplied bytes.
	bound, err := plugin.LockBuildOutput(plan, plugin.BuildOutputDigest(payload))
	if err != nil {
		return semantic.Namespace{}, fmt.Errorf("build output binding: %w", err)
	}
	if bound != selected {
		return semantic.Namespace{}, errors.New("selected build plan or artifact digest mismatch")
	}
	return semanticdistribution.InspectLockedSemanticAsset(
		lock,
		selected,
		expected,
		func(digest string) ([]byte, error) {
			if digest != plugin.BuildOutputDigest(payload) {
				return nil, errors.New("exact selected semantic artifact bytes are unavailable")
			}
			return payload, nil
		},
	)
}
