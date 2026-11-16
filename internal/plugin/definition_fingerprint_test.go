package plugin_test

import (
	"reflect"
	"testing"

	"tei/internal/database/migration"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
)

// fingerprintFixture returns a definition whose every ordered list is in
// reverse-sorted order. Sorting it would change the order, so a normalizer that
// forgets any one field is visible: without that field's sort the fingerprint
// keeps the reversed order and differs from the sorted one.
func fingerprintFixture(t *testing.T) plugin.Definition {
	t.Helper()
	d := plugin.Definition{
		ID:      cid("tei.multi"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Provides: []plugin.Capability{
			{
				Kind:              plugin.CapabilityObservationProvider,
				ID:                cid("observe.b"),
				Actions:           []deployment.ActionKind{deployment.ActionTrafficShift, deployment.ActionDatabaseRelease},
				Observations:      []deployment.ObservationKind{deployment.ObservationDatabaseState, deployment.ObservationApplicationRuntime},
				Evidence:          []deployment.EvidenceKind{deployment.EvidenceRolloutHealth, deployment.EvidenceKind("deploy-plan")},
				DatabaseProviders: []environment.DatabaseProvider{environment.ProviderPostgres, environment.ProviderMySQL},
				MigrationTools:    []migration.Tool{migration.ToolFlyway, migration.ToolAtlas},
				Features:          []environment.Feature{environment.FeatureSQLAnalysis, environment.FeatureDatabaseWorkbench},
				Permissions: []permission.Spec{
					{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "service"},
					{Kind: permission.KindFilesystem, Access: permission.AccessRead, Scope: "data"},
				},
			},
			{Kind: plugin.CapabilityDatabaseProvider, ID: cid("database.a")},
		},
		Requires: []plugin.Requirement{
			{Plugin: cid("tei.z"), CompatibleMajor: 1},
			{Plugin: cid("tei.a"), CompatibleMajor: 2},
		},
		External: []plugin.ExternalRequirement{
			{Scope: "runtime", Kind: "binary", Name: "z"},
			{Scope: "build", Kind: "toolchain", Name: "a"},
		},
	}
	// Confirm the lists are in non-sorted order, so the test below is not
	// vacuous for any field.
	if isSortedStrings([]string{string(d.Provides[0].Actions[0]), string(d.Provides[0].Actions[1])}) {
		t.Fatalf("fixture actions are already sorted: %v", d.Provides[0].Actions)
	}
	if isSortedStrings([]string{string(d.Provides[0].Observations[0]), string(d.Provides[0].Observations[1])}) {
		t.Fatalf("fixture observations are already sorted: %v", d.Provides[0].Observations)
	}
	if isSortedStrings([]string{string(d.Provides[0].Evidence[0]), string(d.Provides[0].Evidence[1])}) {
		t.Fatalf("fixture evidence is already sorted: %v", d.Provides[0].Evidence)
	}
	if isSortedStrings([]string{string(d.Provides[0].DatabaseProviders[0]), string(d.Provides[0].DatabaseProviders[1])}) {
		t.Fatalf("fixture providers are already sorted: %v", d.Provides[0].DatabaseProviders)
	}
	if isSortedStrings([]string{string(d.Provides[0].MigrationTools[0]), string(d.Provides[0].MigrationTools[1])}) {
		t.Fatalf("fixture migration tools are already sorted: %v", d.Provides[0].MigrationTools)
	}
	if isSortedStrings([]string{string(d.Provides[0].Features[0]), string(d.Provides[0].Features[1])}) {
		t.Fatalf("fixture features are already sorted: %v", d.Provides[0].Features)
	}
	return d
}

// isSortedStrings reports whether a two-element slice is already in order.
func isSortedStrings(s []string) bool { return len(s) < 2 || s[0] <= s[1] }

// The plugin fingerprint is what a lock pins, so two declarations that differ
// only in the order of their lists describe the same plugin and must not look
// like a divergence. A normalizer that misses one field makes a workspace whose
// lists were reordered fail its own lock.
//
// Each case reverses one field and checks both that the reversal actually
// changed that field and that the fingerprint did not change. The first check
// is what stops the case from being vacuous: a two-element list that is already
// in reverse-sorted order reverses into the sorted order, so a missing sort
// would go unnoticed without it.
func TestDefinitionFingerprintIsOrderInvariant(t *testing.T) {
	cases := []struct {
		name     string
		reversed func(*plugin.Definition)
	}{
		{name: "provides", reversed: func(d *plugin.Definition) { reverse(d.Provides) }},
		{name: "actions", reversed: func(d *plugin.Definition) { reverse(d.Provides[0].Actions) }},
		{name: "observations", reversed: func(d *plugin.Definition) { reverse(d.Provides[0].Observations) }},
		{name: "evidence", reversed: func(d *plugin.Definition) { reverse(d.Provides[0].Evidence) }},
		{name: "database providers", reversed: func(d *plugin.Definition) { reverse(d.Provides[0].DatabaseProviders) }},
		{name: "migration tools", reversed: func(d *plugin.Definition) { reverse(d.Provides[0].MigrationTools) }},
		{name: "features", reversed: func(d *plugin.Definition) { reverse(d.Provides[0].Features) }},
		{name: "permissions", reversed: func(d *plugin.Definition) { reverse(d.Provides[0].Permissions) }},
		{name: "requires", reversed: func(d *plugin.Definition) { reverse(d.Requires) }},
		{name: "external", reversed: func(d *plugin.Definition) { reverse(d.External) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := fingerprintFixture(t)
			after := fingerprintFixture(t)
			tc.reversed(&after)

			// The reversal must have changed the field, or this case would pass
			// whether or not that field is normalized.
			if fieldsEqual(before, after) {
				t.Fatalf("reversing %s changed nothing, so the case cannot prove the sort", tc.name)
			}

			if got, want := plugin.DefinitionFingerprint(after), plugin.DefinitionFingerprint(before); got != want {
				t.Fatalf("reordering %s changed the fingerprint:\n got  %s\n want %s", tc.name, got, want)
			}
		})
	}
}

// fieldsEqual reports whether two fixtures are structurally identical. The
// fixture is built by value, so a plain comparison is enough.
func fieldsEqual(a, b plugin.Definition) bool {
	return reflect.DeepEqual(a, b)
}

// Normalizing twice must be the same as normalizing once, or the fingerprint
// would depend on how many times a definition happened to pass through.
func TestNormalizeDefinitionIsIdempotent(t *testing.T) {
	once := plugin.NormalizeDefinition(fingerprintFixture(t))
	twice := plugin.NormalizeDefinition(once)
	if plugin.DefinitionFingerprint(once) != plugin.DefinitionFingerprint(twice) {
		t.Fatal("a second normalization changed the fingerprint")
	}
}

// The fingerprint must still change when the declaration itself changes, or it
// would not identify anything. These are the differences a lock exists to catch.
func TestDefinitionFingerprintChangesWithTheDeclaration(t *testing.T) {
	base := fingerprintFixture(t)
	want := plugin.DefinitionFingerprint(base)

	cases := []struct {
		name string
		edit func(*plugin.Definition)
	}{
		{name: "version", edit: func(d *plugin.Definition) { d.Version = "1.0.1" }},
		{name: "runtime kind", edit: func(d *plugin.Definition) {
			d.Runtime = plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "./p"}
		}},
		{name: "runtime command", edit: func(d *plugin.Definition) {
			d.Runtime = plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "./other"}
		}},
		{name: "requirement major", edit: func(d *plugin.Definition) {
			d.Requires[0].CompatibleMajor = 9
		}},
		{name: "dropped requirement", edit: func(d *plugin.Definition) {
			d.Requires = d.Requires[:1]
		}},
		{name: "dropped capability", edit: func(d *plugin.Definition) {
			d.Provides = d.Provides[:1]
		}},
		{name: "added action", edit: func(d *plugin.Definition) {
			d.Provides[0].Actions = append(d.Provides[0].Actions, deployment.ActionApplicationStart)
		}},
		{name: "added permission", edit: func(d *plugin.Definition) {
			d.Provides[0].Permissions = append(d.Provides[0].Permissions,
				permission.Spec{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "other"})
		}},
		{name: "added external", edit: func(d *plugin.Definition) {
			d.External = append(d.External, plugin.ExternalRequirement{Scope: "runtime", Kind: "binary", Name: "extra"})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			changed := fingerprintFixture(t)
			tc.edit(&changed)
			if got := plugin.DefinitionFingerprint(changed); got == want {
				t.Fatalf("changing the %s did not change the fingerprint", tc.name)
			}
		})
	}
}

// A lock built from one ordering must still match the same declaration written
// in another, which is the property the fingerprint invariance exists for.
func TestALockMatchesTheSameDeclarationInAnotherOrder(t *testing.T) {
	env := environment.Environment{ID: cid("prod")}
	env.Plugins = []environment.ExtensionRequirement{{ID: cid("tei.multi"), CompatibleMajor: 1}}

	// The fixture's requirements must resolve, or BuildLock refuses before the
	// fingerprint is ever compared. Declare the dependencies it names.
	dependencies := []plugin.Definition{
		{ID: cid("tei.a"), Version: "2.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
		{ID: cid("tei.z"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
	}

	pinned := fingerprintFixture(t)
	catalog := plugin.Catalog{Plugins: append([]plugin.Definition{pinned}, dependencies...)}
	lock, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		t.Fatalf("BuildLock: %v", err)
	}

	// The same declaration, written with its lists in another order, still
	// satisfies the lock: the fingerprint describes the plugin, not the layout
	// of its source.
	reordered := fingerprintFixture(t)
	reverse(reordered.Requires)
	reverse(reordered.Provides)
	reverse(reordered.Provides[0].Actions)
	reorderedCatalog := plugin.Catalog{Plugins: append([]plugin.Definition{reordered}, dependencies...)}

	if _, err := reorderedCatalog.ResolveEnvironmentWithLock(env, lock); err != nil {
		t.Fatalf("a reordered declaration must satisfy its own lock: %v", err)
	}

	// And a real change to the declaration does not: this is what the lock is
	// for, so the invariance above is not just "the fingerprint ignores
	// everything".
	changed := fingerprintFixture(t)
	changed.Version = "1.0.1"
	changedCatalog := plugin.Catalog{Plugins: append([]plugin.Definition{changed}, dependencies...)}
	if _, err := changedCatalog.ResolveEnvironmentWithLock(env, lock); err == nil {
		t.Fatal("a changed declaration must not satisfy the old lock")
	}
}

// reverse reverses any slice in place.
func reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
