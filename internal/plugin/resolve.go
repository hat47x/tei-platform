package plugin

import (
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/environment"
)

type Catalog struct {
	Plugins []Definition
	Bundles []Bundle
}

func (c Catalog) resolvePlugin(req Requirement) (Definition, error) {
	var matches []Definition
	for _, p := range c.Plugins {
		if p.ID == req.Plugin && CompatibleMajor(p.Version, req.CompatibleMajor) {
			matches = append(matches, p)
		}
	}
	if len(matches) == 0 {
		return Definition{}, fmt.Errorf("no plugin %s compatible with major %d", req.Plugin, req.CompatibleMajor)
	}
	sort.Slice(matches, func(i, j int) bool { return CompareVersion(matches[i].Version, matches[j].Version) > 0 })
	if len(matches) > 1 && matches[0].Version == matches[1].Version {
		return Definition{}, fmt.Errorf("plugin %s version %s is declared more than once", req.Plugin, matches[0].Version)
	}
	return NormalizeDefinition(matches[0]), nil
}

func (c Catalog) resolveBundle(req environment.ExtensionRequirement) (Bundle, error) {
	var matches []Bundle
	for _, b := range c.Bundles {
		if b.ID == req.ID && CompatibleMajor(b.Version, req.CompatibleMajor) {
			matches = append(matches, b)
		}
	}
	if len(matches) == 0 {
		return Bundle{}, fmt.Errorf("no bundle %s compatible with major %d", req.ID, req.CompatibleMajor)
	}
	sort.Slice(matches, func(i, j int) bool { return CompareVersion(matches[i].Version, matches[j].Version) > 0 })
	if len(matches) > 1 && matches[0].Version == matches[1].Version {
		return Bundle{}, fmt.Errorf("bundle %s version %s is declared more than once", req.ID, matches[0].Version)
	}
	return matches[0], nil
}

func (c Catalog) ResolveEnvironment(env environment.Environment) (ResolvedSet, error) {
	var queue []Requirement
	for _, p := range env.Plugins {
		queue = append(queue, Requirement{Plugin: p.ID, CompatibleMajor: p.CompatibleMajor})
	}
	for _, b := range env.Bundles {
		bundle, err := c.resolveBundle(b)
		if err != nil {
			return ResolvedSet{}, err
		}
		queue = append(queue, bundle.Plugins...)
	}

	resolved := map[core.CanonicalID]Definition{}
	visiting := map[core.CanonicalID]bool{}
	var visit func(Requirement) error
	visit = func(req Requirement) error {
		if existing, ok := resolved[req.Plugin]; ok {
			if !CompatibleMajor(existing.Version, req.CompatibleMajor) {
				return fmt.Errorf("plugin %s already resolved to %s, incompatible with required major %d", req.Plugin, existing.Version, req.CompatibleMajor)
			}
			return nil
		}
		if visiting[req.Plugin] {
			return fmt.Errorf("plugin dependency cycle at %s", req.Plugin)
		}
		visiting[req.Plugin] = true
		p, err := c.resolvePlugin(req)
		if err != nil {
			return err
		}
		for _, child := range p.Requires {
			if err := visit(child); err != nil {
				return fmt.Errorf("plugin %s requires: %w", p.ID, err)
			}
		}
		visiting[req.Plugin] = false
		resolved[p.ID] = p
		return nil
	}
	for _, req := range queue {
		if err := visit(req); err != nil {
			return ResolvedSet{}, err
		}
	}

	out := ResolvedSet{Environment: env.ID}
	var defs []Definition
	for _, p := range resolved {
		defs = append(defs, p)
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].ID < defs[j].ID })
	for _, p := range defs {
		ident := Identity{ID: p.ID, Version: p.Version}
		out.Plugins = append(out.Plugins, ident)
		for _, cap := range p.Provides {
			out.Capabilities = append(out.Capabilities, ResolvedCapability{Plugin: ident, Capability: cap})
		}
	}
	sort.Slice(out.Capabilities, func(i, j int) bool {
		a, b := out.Capabilities[i], out.Capabilities[j]
		if a.Capability.Kind != b.Capability.Kind {
			return a.Capability.Kind < b.Capability.Kind
		}
		if a.Capability.ID != b.Capability.ID {
			return a.Capability.ID < b.Capability.ID
		}
		return a.Plugin.ID < b.Plugin.ID
	})
	return out, nil
}

func (r ResolvedSet) Capability(kind CapabilityKind, id core.CanonicalID) (ResolvedCapability, bool) {
	var found *ResolvedCapability
	for i := range r.Capabilities {
		c := r.Capabilities[i]
		if c.Capability.Kind != kind || c.Capability.ID != id {
			continue
		}
		if found != nil {
			return ResolvedCapability{}, false
		}
		copy := c
		found = &copy
	}
	if found == nil {
		return ResolvedCapability{}, false
	}
	return *found, true
}
