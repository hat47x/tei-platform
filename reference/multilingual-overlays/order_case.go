package multilingualoverlays

import (
	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/modelref"
	"tei/internal/plugin"
	"tei/internal/subject"
	resources "tei/reference/multilingual-resources"
)

// ReferenceOrderCase uses the real plugin catalog and lock machinery over
// synthetic source definitions. The ordinary plugin declares no i18n support.
func ReferenceOrderCase() (plugin.Catalog, environment.Environment, plugin.Lock, modelref.Graph, []Surface, []Pack, []Grant, Lookup, error) {
	ordinary := plugin.Identity{ID: core.CanonicalID("std.order_entry"), Version: "1.2.0"}
	localizer := plugin.Identity{ID: core.CanonicalID("std.lang_en"), Version: "1.0.0"}
	catalog := plugin.Catalog{Plugins: []plugin.Definition{
		{ID: ordinary.ID, Version: ordinary.Version, Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
		{ID: localizer.ID, Version: localizer.Version, Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
	}}
	env := environment.Environment{ID: core.CanonicalID("env.order_reference"), Plugins: []environment.ExtensionRequirement{
		{ID: ordinary.ID, CompatibleMajor: 1},
		{ID: localizer.ID, CompatibleMajor: 1},
	}}
	lock, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		return plugin.Catalog{}, environment.Environment{}, plugin.Lock{}, modelref.Graph{}, nil, nil, nil, Lookup{}, err
	}
	key := modelref.Key{Scope: "sales", Subject: subject.Subject{Kind: "function", ID: "sales.order.submit"}}
	graph := modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{{Key: key, DisplayName: "受注を確定"}}}
	surfaces := []Surface{{Target: ordinary, Subject: key, Field: "display-name", SourceLocale: "ja-JP"}}
	resource := resources.Key{Owner: string(ordinary.ID), Target: key, Field: "display-name"}
	packs := []Pack{{Provider: localizer, Target: ordinary, Key: resource,
		Locale: "en-US", Text: "Confirm order", BasedOn: Fingerprint("受注を確定"), Review: "confirmed"}}
	grants := []Grant{{Environment: string(env.ID), Provider: localizer, Target: ordinary, Key: resource, Locale: "en-US"}}
	request := Lookup{Target: ordinary, Key: resource, Locale: "en-US"}
	return catalog, env, lock, graph, surfaces, packs, grants, request, nil
}

// ReferenceOrderDisplay provides the exact validated manifest for the
// unchanged reference order form, not a dynamic translation service.
func ReferenceOrderDisplay(locale string) (DisplayManifest, error) {
	c, env, lock, graph, surfaces, packs, grants, lookup, err := ReferenceOrderCase()
	if err != nil {
		return DisplayManifest{}, err
	}
	lookup.Locale = locale
	return BuildLockedDisplay(c, env, lock, graph, surfaces, packs, grants, nil, lookup)
}
