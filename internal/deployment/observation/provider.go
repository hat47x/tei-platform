package observation

import (
	"context"
	"fmt"

	"tei/internal/core"
	dbrelease "tei/internal/database/release"
	"tei/internal/deployment"
	"tei/internal/environment"
)

type Request struct {
	Environment core.CanonicalID           `json:"environment"`
	Kind        deployment.ObservationKind `json:"kind"`
}

type Fragment struct {
	ProviderID         core.CanonicalID                     `json:"providerId"`
	Environment        core.CanonicalID                     `json:"environment"`
	Kind               deployment.ObservationKind           `json:"kind"`
	ObservedAt         string                               `json:"observedAt,omitempty"`
	ApplicationRuntime []deployment.ApplicationRuntimeState `json:"applicationRuntime,omitempty"`
	Databases          []dbrelease.StateSnapshot            `json:"databases,omitempty"`
}

type Provider interface {
	ID() string
	Supports(deployment.ObservationKind) bool
	Observe(context.Context, Request) (Fragment, error)
}

type Router struct {
	Providers map[core.CanonicalID]Provider
}

func Binding(env environment.Environment, kind deployment.ObservationKind) (core.CanonicalID, error) {
	var id core.CanonicalID
	for _, b := range env.ObservationProviders {
		if b.Kind != kind {
			continue
		}
		if id != "" {
			return "", fmt.Errorf("environment %s has ambiguous observation providers for %s", env.ID, kind)
		}
		id = b.Provider
	}
	if id == "" {
		return "", fmt.Errorf("environment %s has no observation provider for %s", env.ID, kind)
	}
	return id, nil
}

func (r Router) Observe(ctx context.Context, env environment.Environment, kind deployment.ObservationKind) (Fragment, error) {
	id, err := Binding(env, kind)
	if err != nil {
		return Fragment{}, err
	}
	p, ok := r.Providers[id]
	if !ok {
		return Fragment{}, fmt.Errorf("observation provider %s is bound but not registered", id)
	}
	if p.ID() != string(id) || !p.Supports(kind) {
		return Fragment{}, fmt.Errorf("observation provider %s does not conform to binding for %s", id, kind)
	}
	f, err := p.Observe(ctx, Request{Environment: env.ID, Kind: kind})
	if err != nil {
		return Fragment{}, err
	}
	if f.ProviderID == "" {
		f.ProviderID = id
	}
	if f.Environment == "" {
		f.Environment = env.ID
	}
	if f.Kind == "" {
		f.Kind = kind
	}
	if f.ProviderID != id || f.Environment != env.ID || f.Kind != kind {
		return Fragment{}, fmt.Errorf("observation provider %s returned mismatched fragment identity", id)
	}
	return f, nil
}

// Merge applies authoritative observation facets to a previous snapshot.
// There is exactly one provider binding per ObservationKind in the current
// model, so a fragment replaces that whole facet rather than behaving like a
// patch. This is necessary to observe disappearance (for example, a retired
// Application release no longer being present).
func Merge(base deployment.Snapshot, fragments ...Fragment) (deployment.Snapshot, error) {
	out := base.Normalize()
	latestAt := out.ObservedAt
	seenKinds := map[deployment.ObservationKind]bool{}
	for _, f := range fragments {
		if f.Environment != out.Environment {
			return deployment.Snapshot{}, fmt.Errorf("observation fragment environment %s does not match %s", f.Environment, out.Environment)
		}
		if seenKinds[f.Kind] {
			return deployment.Snapshot{}, fmt.Errorf("multiple authoritative observation fragments supplied for kind %s", f.Kind)
		}
		seenKinds[f.Kind] = true
		if f.ObservedAt > latestAt {
			latestAt = f.ObservedAt
		}
		switch f.Kind {
		case deployment.ObservationApplicationRuntime:
			out.ApplicationRuntime = append([]deployment.ApplicationRuntimeState(nil), f.ApplicationRuntime...)
			out.ActiveApplications = nil
			for _, a := range out.ApplicationRuntime {
				out.ActiveApplications = append(out.ActiveApplications, deployment.ApplicationReleaseRef{Application: a.Application, Version: a.Version})
			}
		case deployment.ObservationDatabaseState:
			out.Databases = append([]dbrelease.StateSnapshot(nil), f.Databases...)
		default:
			return deployment.Snapshot{}, fmt.Errorf("unsupported observation kind %s", f.Kind)
		}
	}
	out.ObservedAt = latestAt
	return out.Normalize(), nil
}
