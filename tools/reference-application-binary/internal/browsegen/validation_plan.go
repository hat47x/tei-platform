package browsegen

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

// ErrValidationKindUnsupported is returned before anything is written when the
// Application declares a Validation whose kind this generator cannot realise.
var ErrValidationKindUnsupported = errors.New("VALIDATION_KIND_UNSUPPORTED")

type Rule struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Attribute string `json:"attribute"`
	Relation  string `json:"relation,omitempty"`
}

// ValidationPlan is the rule gate of one Application: the Validations it
// declares, in declaration order. Effects are not part of it.
type ValidationPlan struct {
	ApplicationID  string     `json:"applicationId"`
	Rules          []Rule     `json:"rules"`
	Omissions      []Omission `json:"omissions"`
	SourceRevision string     `json:"sourceRevision"`
	ModelDigest    string     `json:"modelDigest"`
}

// NewValidationPlan loads the Workspace and resolves the Validations the
// Application declares with `uses validation`, in the declared order.
func NewValidationPlan(workspaceRoot, applicationID string) (*ValidationPlan, error) {
	loaded, err := workspace.Load(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("load workspace: %w", err)
	}
	resolver := driver.NewResolver(loaded.World)
	rules, err := resolver.ResolveApplicationValidations(core.CanonicalID(applicationID))
	if err != nil {
		return nil, err
	}
	p := &ValidationPlan{ApplicationID: applicationID}
	if p.Rules, err = realisableRules(rules); err != nil {
		return nil, err
	}
	app, err := applicationOf(loaded, applicationID)
	if err != nil {
		return nil, err
	}
	for _, pt := range app.Patterns {
		reason := "not-selected"
		if pt.Kind == application.PatternCommand {
			// The effect behind a Command is not in the Source; saying so is not
			// the same as leaving it out silently.
			reason = "effect-not-in-source"
		}
		p.Omissions = append(p.Omissions, Omission{"pattern", string(pt.ID), reason})
	}
	sort.Slice(p.Omissions, func(i, j int) bool { return p.Omissions[i].ID < p.Omissions[j].ID })
	idx, err := workspace.BuildSourceIndexFromLoad(workspaceRoot, loaded)
	if err != nil {
		return nil, fmt.Errorf("source index (fail closed): %w", err)
	}
	p.SourceRevision = indexRevision(idx, "")
	b, _ := json.Marshal(p.Rules)
	p.ModelDigest = "sha256:" + sha(b)
	return p, nil
}

func applicationOf(loaded workspace.Result, id string) (application.Application, error) {
	for _, a := range loaded.World.Applications {
		if string(a.ID) == id {
			return a, nil
		}
	}
	return application.Application{}, fmt.Errorf("application %q not found", id)
}

// realisableRules keeps the declared order and refuses, before anything is
// written, a rule whose kind this generator has no evaluation for.
func realisableRules(rules []semantic.Validation) ([]Rule, error) {
	out := make([]Rule, 0, len(rules))
	for _, r := range rules {
		if r.Kind != semantic.ValidationPositive && r.Kind != semantic.ValidationRelationUnique {
			return nil, fmt.Errorf("%w: %s has kind %q", ErrValidationKindUnsupported, r.ID, r.Kind)
		}
		out = append(out, Rule{ID: string(r.ID), Kind: string(r.Kind), Attribute: string(r.Attribute), Relation: string(r.Relation)})
	}
	return out, nil
}
