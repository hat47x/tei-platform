package dependency

import (
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/plugin"
)

func id(s string) core.CanonicalID { return core.MustCanonicalID(s) }

func mapPluginScope(v string) (Scope, bool) {
	s := Scope(v)
	switch s {
	case ScopeEditor, ScopeBuild, ScopeTest, ScopeRuntime, ScopeMigration:
		return s, true
	default:
		return "", false
	}
}

func mapPluginKind(v string) (Kind, bool) {
	k := Kind(v)
	switch k {
	case KindToolchain, KindRuntime, KindLanguagePackage, KindTool:
		return k, true
	default:
		return "", false
	}
}

func Derive(w ir.World) []Requirement {
	byKey := map[string]*Requirement{}
	add := func(environment string, scope Scope, kind Kind, name, introducedBy, reason string) {
		key := fmt.Sprintf("%s|%s|%s|%s", environment, scope, kind, name)
		r, ok := byKey[key]
		if !ok {
			r = &Requirement{Environment: id(environment), Scope: scope, Kind: kind, Name: name}
			byKey[key] = r
		}
		for _, o := range r.Origins {
			if o.IntroducedBy == introducedBy && o.Reason == reason {
				return
			}
		}
		r.Origins = append(r.Origins, Origin{IntroducedBy: introducedBy, Reason: reason})
	}

	bindingsByEnv := map[string]map[string]struct{}{}
	for _, env := range w.Environments {
		apps := map[string]struct{}{}
		for _, b := range env.Bindings {
			apps[string(b.Application)] = struct{}{}
		}
		bindingsByEnv[string(env.ID)] = apps
	}

	catalog := plugin.Catalog{Plugins: w.Plugins, Bundles: w.Bundles}
	for _, env := range w.Environments {
		resolved, err := catalog.ResolveEnvironment(env)
		if err != nil {
			continue
		} // validation reports resolution failures
		seenPlugin := map[core.CanonicalID]struct{}{}
		for _, rp := range resolved.Plugins {
			if _, ok := seenPlugin[rp.ID]; ok {
				continue
			}
			seenPlugin[rp.ID] = struct{}{}
			var def *plugin.Definition
			for i := range w.Plugins {
				if w.Plugins[i].ID == rp.ID && w.Plugins[i].Version == rp.Version {
					def = &w.Plugins[i]
					break
				}
			}
			if def == nil {
				continue
			}
			for _, ext := range def.External {
				scope, ok := mapPluginScope(ext.Scope)
				if !ok {
					continue
				}
				kind, ok := mapPluginKind(ext.Kind)
				if !ok {
					continue
				}
				add(string(env.ID), scope, kind, ext.Name, "plugin:"+string(def.ID)+"@"+def.Version, "runtime requirement declared by resolved TEI plugin")
			}
		}
	}

	for _, b := range w.Bindings {
		for _, env := range w.Environments {
			if _, ok := bindingsByEnv[string(env.ID)][string(b.Application)]; !ok {
				continue
			}
			add(string(env.ID), ScopeBuild, KindToolchain, "go", "custom-go:"+string(b.Capability), "build custom Go realization")
		}
	}

	out := make([]Requirement, 0, len(byKey))
	for _, r := range byKey {
		sort.Slice(r.Origins, func(i, j int) bool { return r.Origins[i].IntroducedBy < r.Origins[j].IntroducedBy })
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Environment != b.Environment {
			return a.Environment < b.Environment
		}
		if a.Scope != b.Scope {
			return a.Scope < b.Scope
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Name < b.Name
	})
	return out
}
