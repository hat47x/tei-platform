package evidence

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/durable"
	"tei/internal/environment"
)

type Request struct {
	Environment        core.CanonicalID        `json:"environment"`
	ActionID           string                  `json:"actionId"`
	Kind               deployment.EvidenceKind `json:"kind"`
	Application        core.CanonicalID        `json:"application,omitempty"`
	Version            string                  `json:"version,omitempty"`
	TrafficBasisPoints int                     `json:"trafficBasisPoints,omitempty"`
}

type Provider interface {
	ID() string
	Supports(deployment.EvidenceKind) bool
	Collect(context.Context, Request) (durable.Evidence, error)
}

type Router struct{ Providers map[core.CanonicalID]Provider }

func Binding(env environment.Environment, kind deployment.EvidenceKind) (core.CanonicalID, error) {
	var id core.CanonicalID
	for _, b := range env.EvidenceProviders {
		if b.Kind != kind {
			continue
		}
		if id != "" {
			return "", fmt.Errorf("environment %s has ambiguous evidence providers for %s", env.ID, kind)
		}
		id = b.Provider
	}
	if id == "" {
		return "", fmt.Errorf("environment %s has no evidence provider for %s", env.ID, kind)
	}
	return id, nil
}

func (r Router) Collect(ctx context.Context, env environment.Environment, rec durable.ActionRecord) (durable.Evidence, error) {
	if rec.Mode != durable.ModeEvidence {
		return durable.Evidence{}, fmt.Errorf("action %s is not an evidence gate", rec.ID)
	}
	id, err := Binding(env, rec.EvidenceKind)
	if err != nil {
		return durable.Evidence{}, err
	}
	p, ok := r.Providers[id]
	if !ok {
		return durable.Evidence{}, fmt.Errorf("evidence provider %s is bound but not registered", id)
	}
	if p.ID() != string(id) || !p.Supports(rec.EvidenceKind) {
		return durable.Evidence{}, fmt.Errorf("evidence provider %s does not conform to binding for %s", id, rec.EvidenceKind)
	}
	e, err := p.Collect(ctx, Request{Environment: env.ID, ActionID: rec.ID, Kind: rec.EvidenceKind, Application: rec.Application, Version: rec.Version, TrafficBasisPoints: rec.TrafficBasisPoints})
	if err != nil {
		return durable.Evidence{}, err
	}
	if e.ActionID == "" {
		e.ActionID = rec.ID
	}
	if e.Kind == "" {
		e.Kind = rec.EvidenceKind
	}
	if e.ActionID != rec.ID || e.Kind != rec.EvidenceKind {
		return durable.Evidence{}, fmt.Errorf("evidence provider %s returned mismatched evidence target", id)
	}
	if e.SourceRef == "" {
		e.SourceRef = "provider://" + string(id)
	}
	return e, nil
}
