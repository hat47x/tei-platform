package management

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/environment"
	"tei/internal/plugin"
)

func TestDocumentReadPreflightUsesExistingResolverSelection(t *testing.T) {
	env := baseEnvironment(t, "tei.database.postgres")
	plugins := []plugin.Definition{builtinPlugin(t, "tei.database.postgres", databaseProviderCapability(t, "database.postgres", environment.ProviderPostgres)), builtinPlugin(t, "tei.unselected")}

	report, err := Derive(Input{World: world(plugins, env), Environment: env.ID})
	if err != nil {
		t.Fatal(err)
	}
	unknownLock := InspectDocumentReadPreflight(report, "tei.database.postgres")
	if unknownLock.SelectionStatus != DocumentSelectionLockUnknown || unknownLock.LockPinned != nil ||
		unknownLock.ResolvedVersion != "1.0.0" {
		t.Fatalf("selected Plugin without lock must remain lock-unknown: %+v", unknownLock)
	}
	if got := InspectDocumentReadPreflight(report, "tei.unselected"); got.SelectionStatus != DocumentSelectionNotSelected {
		t.Fatalf("declared but not selected: %+v", got)
	}
	if got := InspectDocumentReadPreflight(report, "tei.not.in.world"); got.SelectionStatus != DocumentSelectionNotInInventory {
		t.Fatalf("unknown Plugin: %+v", got)
	}

	lock, err := plugin.BuildLock(plugin.Catalog{Plugins: plugins}, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	report, err = Derive(Input{World: world(plugins, env), Environment: env.ID, Lock: &lock})
	if err != nil {
		t.Fatal(err)
	}
	got := InspectDocumentReadPreflight(report, "tei.database.postgres")
	if got.SelectionStatus != DocumentSelectionExactPinned || got.LockPinned == nil || !*got.LockPinned ||
		got.ResolvedVersion != "1.0.0" || got.Environment != "development" {
		t.Fatalf("exact selection must reuse the inventory's resolved identity: %+v", got)
	}
	if got.SourceBinding != "unbound" || got.Disclosure != "not-checked" || got.ReadEligibility != "not-assessed" {
		t.Fatalf("selection is never Document source or disclosure authority: %+v", got)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"documentBody", "documentId", "permissionGranted", "selectedProvider"} {
		if _, ok := asMap[forbidden]; ok {
			t.Fatalf("preflight fabricated %s: %s", forbidden, raw)
		}
	}
}

func TestDocumentReadPreflightFailsClosedOnUnqualifiedStates(t *testing.T) {
	trueValue, falseValue := true, false
	base := func() Report {
		return Report{
			Environment: "development",
			ScanState:   ScanComplete,
			Resolver:    ResolverOutcome{Resolved: true},
			Rows: []Row{{
				PluginID:    "tei.database.postgres",
				Version:     "1.0.0",
				Declared:    true,
				EnvSelected: true,
				Resolved:    true,
				LockPinned:  &trueValue,
			}},
		}
	}
	cases := []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{"partial-scan", func(r *Report) { r.ScanState = ScanPartial }, DocumentSelectionScanIncomplete},
		{"unrun-scan", func(r *Report) { r.ScanState = ScanNotRun }, DocumentSelectionScanIncomplete},
		{"not-declared", func(r *Report) { r.Rows[0].Declared = false }, DocumentSelectionNotDeclared},
		{"not-selected", func(r *Report) { r.Rows[0].EnvSelected = false }, DocumentSelectionNotSelected},
		{"resolver-failed", func(r *Report) { r.Resolver.Resolved = false }, DocumentSelectionResolverUnbound},
		{"row-unresolved", func(r *Report) { r.Rows[0].Resolved = false }, DocumentSelectionResolverUnbound},
		{"unparseable-version", func(r *Report) { r.Rows[0].Version = "a.b.c" }, DocumentSelectionVersionUnknown},
		{"missing-lock", func(r *Report) { r.Rows[0].LockPinned = nil }, DocumentSelectionLockUnknown},
		{"mismatched-lock", func(r *Report) { r.Rows[0].LockPinned = &falseValue }, DocumentSelectionLockMismatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := base()
			tc.mutate(&r)
			got := InspectDocumentReadPreflight(r, "tei.database.postgres")
			if got.SelectionStatus != tc.want || got.ReadEligibility != "not-assessed" ||
				got.SourceBinding != "unbound" || got.Disclosure != "not-checked" {
				t.Fatalf("preflight = %+v; want %q and no document authority", got, tc.want)
			}
		})
	}
}

// TestDocumentReadPreflightReportsOnlyDirectUpstream checks an actual
// Catalog -> Resolver -> Lock -> Report chain. The lower Plugin C may
// discover B as its direct requirement, but not silently treat A as a
// direct or publishable document source.
func TestDocumentReadPreflightReportsOnlyDirectUpstream(t *testing.T) {
	a := builtinPlugin(t, "fixture.graph.base")
	a.Version = "1.4.2"
	b := builtinPlugin(t, "fixture.graph.extension")
	b.Requires = []plugin.Requirement{{Plugin: a.ID, CompatibleMajor: 1}}
	c := builtinPlugin(t, "fixture.graph.consumer")
	c.Requires = []plugin.Requirement{{Plugin: b.ID, CompatibleMajor: 1}}
	plugins := []plugin.Definition{a, b, c}
	env := baseEnvironment(t, string(c.ID))
	lock, err := plugin.BuildLock(plugin.Catalog{Plugins: plugins}, []environment.Environment{env})
	if err != nil {
		t.Fatalf("build actual dependency lock: %v", err)
	}

	report, err := Derive(Input{World: world(plugins, env), Environment: env.ID, Lock: &lock})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		pluginID    string
		upstreamID  string
		upstreamVer string
	}{
		{string(c.ID), string(b.ID), b.Version},
		{string(b.ID), string(a.ID), a.Version},
	} {
		got := InspectDocumentReadPreflight(report, tc.pluginID)
		if got.SelectionStatus != DocumentSelectionExactPinned || len(got.Upstream) != 1 {
			t.Fatalf("direct upstream for %s: %+v", tc.pluginID, got)
		}
		up := got.Upstream[0]
		if up.PluginID != tc.upstreamID || up.CompatibleMajor != 1 ||
			up.SelectionStatus != DocumentSelectionExactPinned || up.ResolvedVersion != tc.upstreamVer ||
			up.LockPinned == nil || !*up.LockPinned {
			t.Fatalf("upstream for %s: %+v", tc.pluginID, up)
		}
		if got.SourceBinding != "unbound" || got.Disclosure != "not-checked" ||
			got.ReadEligibility != "not-assessed" {
			t.Fatalf("upstream selection must not authorize a Document read: %+v", got)
		}
		raw, err := json.Marshal(got)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{`"documentBody"`, `"documentId"`, `"sourcePath"`, `"audience"`, `"permissionGranted"`} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatalf("unexpected manual disclosure: %s", raw)
			}
		}
	}
	if got := InspectDocumentReadPreflight(report, string(a.ID)); len(got.Upstream) != 0 {
		t.Fatalf("A has no direct Requires; cannot invent one: %+v", got.Upstream)
	}

	unlocked, err := Derive(Input{World: world(plugins, env), Environment: env.ID})
	if err != nil {
		t.Fatal(err)
	}
	got := InspectDocumentReadPreflight(unlocked, string(c.ID))
	if got.SelectionStatus != DocumentSelectionLockUnknown ||
		len(got.Upstream) != 1 || got.Upstream[0].SelectionStatus != DocumentSelectionLockUnknown ||
		got.Upstream[0].ResolvedVersion != b.Version || got.Upstream[0].LockPinned != nil {
		t.Fatalf("compatible major must not invent a locked exact selection: %+v", got)
	}
}

func TestDocumentReadPreflightUpstreamFailureStates(t *testing.T) {
	yes, no := true, false
	parent := Row{
		PluginID: "fixture.graph.extension", Version: "1.0.0", Declared: true,
		EnvSelected: true, Resolved: true, LockPinned: &yes,
		Requires: []Requirement{{Plugin: "fixture.graph.base", CompatibleMajor: 1}},
	}
	base := Row{
		PluginID: "fixture.graph.base", Version: "1.4.2", Declared: true,
		EnvSelected: true, Resolved: true, LockPinned: &yes,
	}
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{"missing", func(r *Report) { r.Rows = r.Rows[:1] }, DocumentSelectionNotInInventory},
		{"undeclared", func(r *Report) { r.Rows[1].Declared = false }, DocumentSelectionNotDeclared},
		{"unselected", func(r *Report) { r.Rows[1].EnvSelected = false }, DocumentSelectionNotSelected},
		{"unresolved", func(r *Report) { r.Rows[1].Resolved = false }, DocumentSelectionResolverUnbound},
		{"invalid-version", func(r *Report) { r.Rows[1].Version = "invalid" }, DocumentSelectionVersionUnknown},
		{"major-mismatch", func(r *Report) { r.Rows[1].Version = "2.0.0" }, DocumentSelectionVersionMismatch},
		{"lock-unknown", func(r *Report) { r.Rows[1].LockPinned = nil }, DocumentSelectionLockUnknown},
		{"lock-mismatch", func(r *Report) { r.Rows[1].LockPinned = &no }, DocumentSelectionLockMismatch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := Report{
				Environment: "development", ScanState: ScanComplete,
				Resolver: ResolverOutcome{Resolved: true}, Rows: []Row{parent, base},
			}
			tc.mutate(&report)
			got := InspectDocumentReadPreflight(report, parent.PluginID)
			if got.SelectionStatus != DocumentSelectionExactPinned ||
				len(got.Upstream) != 1 || got.Upstream[0].SelectionStatus != tc.want ||
				got.ReadEligibility != "not-assessed" {
				t.Fatalf("dependent Plugin must not be treated as Document source: %+v", got)
			}
			if tc.want == DocumentSelectionVersionMismatch && got.Upstream[0].ResolvedVersion != "" {
				t.Fatalf("an incompatible dependency may not report an accepted exact version: %+v", got.Upstream[0])
			}
		})
	}

	// A failed parent selection, incomplete scan, or absent Plugin must not
	// expose upstream identity as if it were a qualified selection.
	report := Report{
		Environment: "development", ScanState: ScanPartial,
		Resolver: ResolverOutcome{Resolved: true}, Rows: []Row{parent, base},
	}
	if got := InspectDocumentReadPreflight(report, parent.PluginID); len(got.Upstream) != 0 {
		t.Fatalf("partial inventory must not publish upstream claims: %+v", got)
	}
	report.ScanState = ScanComplete
	report.Rows[0].Resolved = false
	if got := InspectDocumentReadPreflight(report, parent.PluginID); len(got.Upstream) != 0 {
		t.Fatalf("unresolved parent must not publish upstream claims: %+v", got)
	}
}
