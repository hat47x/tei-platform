package plugin_test

import (
	"testing"

	"tei/internal/environment"
	"tei/internal/plugin"
)

// providing builds a builtin plugin that provides one capability.
func providing(t *testing.T, pluginID, capabilityID string) plugin.Definition {
	t.Helper()
	return plugin.Definition{
		ID:      cid(pluginID),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Provides: []plugin.Capability{{
			Kind: plugin.CapabilityDatabaseProvider,
			ID:   cid(capabilityID),
		}},
	}
}

// selecting builds an environment that selects the named plugins at major 1.
func selecting(t *testing.T, names ...string) environment.Environment {
	t.Helper()
	env := environment.Environment{ID: cid("prod")}
	for _, name := range names {
		env.Plugins = append(env.Plugins, environment.ExtensionRequirement{ID: cid(name), CompatibleMajor: 1})
	}
	return env
}

// Two plugins that each provide the same (kind, id) are both resolved, because
// uniqueness is enforced per declaration and not across the resolved set. The
// lookup then has to answer, and it answers "not found" — the same answer as a
// capability nobody provides.
//
// That is the property worth pinning: a caller cannot tell an ambiguous
// capability from an absent one, so it must not treat `false` as "nobody
// provides this" without a separate check.
func TestAnAmbiguousCapabilityLooksTheSameAsAnAbsentOne(t *testing.T) {
	c := plugin.Catalog{Plugins: []plugin.Definition{
		providing(t, "tei.a", "database.postgres"),
		providing(t, "tei.b", "database.postgres"),
	}}
	set, err := c.ResolveEnvironment(selecting(t, "tei.a", "tei.b"))
	if err != nil {
		t.Fatalf("two providers resolve, since uniqueness is per declaration: %v", err)
	}

	// Both capabilities are in the set, so the ambiguity is real and not an
	// artifact of the resolver dropping one.
	if len(set.Capabilities) != 2 {
		t.Fatalf("want two resolved capabilities, got %d: %+v", len(set.Capabilities), set.Capabilities)
	}

	got, ok := set.Capability(plugin.CapabilityDatabaseProvider, cid("database.postgres"))
	if ok {
		t.Fatalf("an ambiguous capability must not be returned as a single answer: %+v", got)
	}

	// The same answer an absent capability gives.
	missing, missingOK := set.Capability(plugin.CapabilityDatabaseProvider, cid("database.absent"))
	if missingOK {
		t.Fatalf("an absent capability must not resolve: %+v", missing)
	}
	if ok != missingOK {
		t.Fatal("ambiguity and absence must not be distinguishable by this call alone")
	}
}

// The kind is part of the identity, so the same id under a different kind is
// not ambiguous. A lookup that keyed on id alone would call this ambiguous.
func TestTheSameIDUnderDifferentKindsIsNotAmbiguous(t *testing.T) {
	shared := plugin.Definition{
		ID:      cid("tei.a"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Provides: []plugin.Capability{
			{Kind: plugin.CapabilityDatabaseProvider, ID: cid("shared.id")},
			{Kind: plugin.CapabilityObservationProvider, ID: cid("shared.id")},
		},
	}
	c := plugin.Catalog{Plugins: []plugin.Definition{shared}}
	set, err := c.ResolveEnvironment(selecting(t, "tei.a"))
	if err != nil {
		t.Fatalf("ResolveEnvironment: %v", err)
	}
	if _, ok := set.Capability(plugin.CapabilityDatabaseProvider, cid("shared.id")); !ok {
		t.Fatal("the database provider entry must resolve")
	}
	if _, ok := set.Capability(plugin.CapabilityObservationProvider, cid("shared.id")); !ok {
		t.Fatal("the observation provider entry must resolve")
	}
}

// A single provider answers with its own plugin identity, which is what a
// caller needs to name the owner of the capability.
func TestASingleProviderAnswersWithItsIdentity(t *testing.T) {
	c := plugin.Catalog{Plugins: []plugin.Definition{providing(t, "tei.a", "database.postgres")}}
	set, err := c.ResolveEnvironment(selecting(t, "tei.a"))
	if err != nil {
		t.Fatalf("ResolveEnvironment: %v", err)
	}
	got, ok := set.Capability(plugin.CapabilityDatabaseProvider, cid("database.postgres"))
	if !ok {
		t.Fatal("the only provider must resolve")
	}
	if got.Plugin.ID != cid("tei.a") || got.Plugin.Version != "1.0.0" {
		t.Fatalf("the answer must name the providing plugin: %+v", got.Plugin)
	}
}

// Two plugins providing the same capability under different kinds each answer,
// so the ambiguity check above is about the pair and not about the count.
func TestTwoProvidersOfDifferentCapabilitiesBothResolve(t *testing.T) {
	c := plugin.Catalog{Plugins: []plugin.Definition{
		providing(t, "tei.a", "database.postgres"),
		{
			ID:      cid("tei.b"),
			Version: "1.0.0",
			Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
			Provides: []plugin.Capability{{
				Kind: plugin.CapabilityEditorIntelligence,
				ID:   cid("editor.assist"),
			}},
		},
	}}
	set, err := c.ResolveEnvironment(selecting(t, "tei.a", "tei.b"))
	if err != nil {
		t.Fatalf("ResolveEnvironment: %v", err)
	}
	if _, ok := set.Capability(plugin.CapabilityDatabaseProvider, cid("database.postgres")); !ok {
		t.Fatal("the database capability must resolve")
	}
	if _, ok := set.Capability(plugin.CapabilityEditorIntelligence, cid("editor.assist")); !ok {
		t.Fatal("the editor capability must resolve")
	}
}

// A single declaration repeating (kind, id) is not refused by resolution. The
// uniqueness rule lives in ValidateCapabilityStructure, which a package load
// calls; a Catalog assembled in code is trusted to have been checked. This test
// records where the rule is enforced, so a reader does not assume the resolver
// is the gate.
func TestResolutionDoesNotCheckCapabilityUniqueness(t *testing.T) {
	repeated := plugin.Definition{
		ID:      cid("tei.a"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Provides: []plugin.Capability{
			{Kind: plugin.CapabilityDatabaseProvider, ID: cid("database.postgres")},
			{Kind: plugin.CapabilityDatabaseProvider, ID: cid("database.postgres")},
		},
	}

	// The declaration rule does refuse it, and that is the only place it is
	// enforced.
	if err := plugin.ValidateCapabilityStructure(repeated.Provides); err == nil {
		t.Fatal("the declaration rule must refuse a repeated capability")
	}

	// Resolution accepts it, and both entries reach the set.
	c := plugin.Catalog{Plugins: []plugin.Definition{repeated}}
	set, err := c.ResolveEnvironment(selecting(t, "tei.a"))
	if err != nil {
		t.Fatalf("resolution trusts the catalog and does not re-check: %v", err)
	}
	if len(set.Capabilities) != 2 {
		t.Fatalf("both declared entries reach the set: %+v", set.Capabilities)
	}

	// The consequence: the lookup answers "not found" for a capability the one
	// plugin declared twice, exactly as it does for two plugins that each
	// declare it once. A caller cannot tell the two apart here.
	if _, ok := set.Capability(plugin.CapabilityDatabaseProvider, cid("database.postgres")); ok {
		t.Fatal("a repeated declaration is ambiguous to the lookup, like two providers")
	}
}
