package semanticdistribution

import (
	"errors"
	"fmt"

	"tei/internal/plugin"
	"tei/internal/semantic"
)

// SemanticReferenceOutputKind is an E4c reference profile for data-only build
// outputs. The existing BuildOutputKind is open; this value is intentionally
// not registered as a production Plugin Capability or executable Runtime kind.
const SemanticReferenceOutputKind plugin.BuildOutputKind = "semantic-data"

// InspectLockedSemanticAsset consumes one explicitly selected, exact artifact
// from the existing BuildArtifactLock. It does not choose a Provider, build an
// artifact, trust its publisher, or grant runtime/acceptance authority.
//
// The caller supplies a fetcher belonging to its chosen artifact store. The
// returned bytes are independently rechecked against the exact locked digest
// by InspectSelectedReference, even if the store verified the bytes already.
// Lock provenance alone never attests that the build or publisher is trusted.
func InspectLockedSemanticAsset(
	lock plugin.BuildArtifactLock,
	selected plugin.LockedBuild,
	expected Expectation,
	fetch func(digest string) ([]byte, error),
) (semantic.Namespace, error) {
	if err := lock.Validate(); err != nil {
		return semantic.Namespace{}, fmt.Errorf("invalid existing BuildArtifactLock: %w", err)
	}
	if selected.OutputKind != SemanticReferenceOutputKind {
		return semantic.Namespace{}, fmt.Errorf("data-only semantic export requires output kind %q, got %q", SemanticReferenceOutputKind, selected.OutputKind)
	}
	if string(selected.Plugin.ID) != expected.Package.PackageID ||
		selected.Plugin.Version != expected.Package.Version {
		return semantic.Namespace{}, errors.New("selected build identity differs from explicit package expectation")
	}
	if selected.ArtifactDigest != expected.ArtifactDigest {
		return semantic.Namespace{}, errors.New("selected build artifact digest differs from explicit package expectation")
	}
	found := false
	for _, item := range lock.Builds {
		if item == selected {
			if found {
				return semantic.Namespace{}, errors.New("ambiguous exact semantic build")
			}
			found = true
		}
	}
	if !found {
		return semantic.Namespace{}, errors.New("selected semantic build is not present in BuildArtifactLock")
	}
	if fetch == nil {
		return semantic.Namespace{}, errors.New("artifact fetcher unavailable")
	}
	bytes, err := fetch(selected.ArtifactDigest)
	if err != nil {
		return semantic.Namespace{}, fmt.Errorf("fetch exact locked semantic artifact: %w", err)
	}
	model, err := InspectSelectedReference(bytes, expected)
	if err != nil {
		return semantic.Namespace{}, err
	}
	return model, nil
}
