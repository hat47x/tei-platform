package management

import (
	"encoding/json"
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
