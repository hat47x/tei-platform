package plugin_test

import (
	"strings"
	"testing"

	"tei/internal/environment"
	"tei/internal/plugin"
)

// A catalog entry whose version does not parse is silently unusable. The
// resolver filters candidates with CompatibleMajor, which answers false on a
// parse error, so a malformed declaration becomes indistinguishable from a
// missing one and the caller is told to install a plugin that is already there.
//
// The management layer already separates these two situations
// (DocumentSelectionVersionUnknown vs DocumentSelectionVersionMismatch), so the
// information exists at one boundary and is dropped at this one. This test pins
// the resolver's current answer so a change to it is deliberate.
func TestAMalformedPluginVersionIsReportedAsNoCompatiblePlugin(t *testing.T) {
	c := plugin.Catalog{Plugins: []plugin.Definition{
		{ID: cid("tei.a"), Version: "1.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
	}}
	env := environment.Environment{
		ID:      cid("prod"),
		Plugins: []environment.ExtensionRequirement{{ID: cid("tei.a"), CompatibleMajor: 1}},
	}
	_, err := c.ResolveEnvironment(env)
	if err == nil {
		t.Fatal("expected a refusal")
	}
	if got := plugin.ResolveCode(err); got != plugin.DiagnosticNoCompatiblePlugin {
		t.Fatalf("ResolveCode = %q, want %q", got, plugin.DiagnosticNoCompatiblePlugin)
	}
	// The message blames the selection ("compatible with major 1") rather than
	// the declaration ("version 1.0 does not parse"). A reader following it
	// would widen the requirement, which cannot help.
	if !strings.Contains(err.Error(), "no plugin tei.a compatible with major 1") {
		t.Fatalf("message changed: %v", err)
	}
	if strings.Contains(err.Error(), "must be major.minor.patch") {
		t.Fatalf("the parse error is now surfaced; update this test: %v", err)
	}
}

// CompareVersion discards the parse error and compares zero values, so every
// unparseable version is 0.0.0. The consequence a caller can observe is that a
// malformed entry sorts below any valid one, so it is never selected over a
// usable candidate.
func TestCompareVersionTreatsUnparseableVersionsAsZero(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want int
	}{
		{name: "equal versions", a: "1.0.0", b: "1.0.0", want: 0},
		{name: "patch differs", a: "1.0.0", b: "1.0.1", want: -1},
		{name: "major dominates", a: "2.0.0", b: "1.9.9", want: 1},
		{name: "both unparseable compare equal", a: "bad", b: "worse", want: 0},
		{name: "unparseable sorts below a valid version", a: "bad", b: "0.0.1", want: -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := plugin.CompareVersion(tc.a, tc.b); got != tc.want {
				t.Fatalf("CompareVersion(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// A malformed entry does not shadow a usable one: the resolver keeps the valid
// version even though the malformed entry is in the same catalog.
func TestAMalformedVersionDoesNotShadowAUsableOne(t *testing.T) {
	c := plugin.Catalog{Plugins: []plugin.Definition{
		{ID: cid("tei.a"), Version: "1.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
		{ID: cid("tei.a"), Version: "1.2.3", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
	}}
	env := environment.Environment{
		ID:      cid("prod"),
		Plugins: []environment.ExtensionRequirement{{ID: cid("tei.a"), CompatibleMajor: 1}},
	}
	set, err := c.ResolveEnvironment(env)
	if err != nil {
		t.Fatalf("a usable entry should resolve: %v", err)
	}
	if len(set.Plugins) != 1 || set.Plugins[0].Version != "1.2.3" {
		t.Fatalf("resolved %+v, want tei.a 1.2.3", set.Plugins)
	}
}

// CompatibleMajor is the acceptance gate for a catalog entry, so its answer on
// a malformed version decides whether a declared plugin can ever be selected.
func TestCompatibleMajorRefusesAMalformedVersion(t *testing.T) {
	cases := []struct {
		version string
		major   int
		want    bool
	}{
		{version: "1.0.0", major: 1, want: true},
		{version: "2.0.0", major: 1, want: false},
		{version: "1.0", major: 1, want: false},
		{version: "1.0.0.0", major: 1, want: false},
		{version: "", major: 1, want: false},
		{version: "1.x.0", major: 1, want: false},
		// A negative component does not parse, so no major can match it. This is
		// the case a caller is most likely to get wrong: the requirement says
		// major 1, the declaration says "-1.0.0", and the two are unrelated.
		{version: "-1.0.0", major: -1, want: false},
		{version: "-1.0.0", major: 1, want: false},
	}
	for _, tc := range cases {
		if got := plugin.CompatibleMajor(tc.version, tc.major); got != tc.want {
			t.Errorf("CompatibleMajor(%q, %d) = %v, want %v", tc.version, tc.major, got, tc.want)
		}
	}
}
