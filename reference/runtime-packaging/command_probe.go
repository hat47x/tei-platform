package runtimepackaging

import (
	"fmt"
	"sort"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/realization"
)

// CommandRealizationObservation links an accepted Command Pattern to an
// Application-scoped Custom Go realization declaration. Expected source names
// are conventions, not proof that a compiled binary contains that code.
type CommandRealizationObservation struct {
	PatternID               string `json:"patternId"`
	CapabilityID            string `json:"capabilityId"`
	BindingIdentity         string `json:"bindingIdentity"`
	ExpectedAdapterFilename string `json:"expectedAdapterFilename"`
	ExpectedHookName        string `json:"expectedHookName"`
}

// ObserveCommandRealizations reuses the existing application-scoped Custom Go
// resolver. It does not scan Go source or infer the concrete linked package.
func ObserveCommandRealizations(world ir.World, applicationID string) ([]CommandRealizationObservation, error) {
	id, err := core.ParseCanonicalID(applicationID)
	if err != nil {
		return nil, fmt.Errorf("application: %w", err)
	}
	var match application.Application
	count := 0
	for _, candidate := range world.Applications {
		if candidate.ID == id {
			match, count = candidate, count+1
		}
	}
	if count != 1 {
		return nil, fmt.Errorf("application %s has %d declarations", id, count)
	}
	seen := map[core.CanonicalID]bool{}
	out := make([]CommandRealizationObservation, 0)
	for _, pattern := range match.Patterns {
		if pattern.Kind != application.PatternCommand {
			continue
		}
		if pattern.ID == "" || pattern.Capability == "" || seen[pattern.ID] {
			return nil, fmt.Errorf("command pattern has missing/duplicate identity or capability: %s", pattern.ID)
		}
		seen[pattern.ID] = true
		b, err := realization.ResolveCustomGoBinding(world.Bindings, id, pattern.Capability)
		if err != nil {
			return nil, fmt.Errorf("command pattern %s: %w", pattern.ID, err)
		}
		out = append(out, CommandRealizationObservation{
			PatternID: string(pattern.ID), CapabilityID: string(pattern.Capability),
			BindingIdentity: b.Identity(), ExpectedAdapterFilename: b.ExpectedAdapterFilename(),
			ExpectedHookName: b.ExpectedHookName(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PatternID < out[j].PatternID })
	return out, nil
}
