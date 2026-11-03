package studioplugininventory

import (
	"reflect"
	"strings"
	"testing"
)

func ptr(v int) *int { return &v }

// fixture covers every status the acceptance names, plus one denied permission.
func fixture() Input {
	return Input{
		Scan: Scan{State: ScanComplete, ObservedAt: "2026-11-01T10:00:00Z", SettingsVersion: "settings-7",
			AllowedTargets: []string{"enabled-plugin", "multi", "unreadable", "providerless", "version-old"}},
		Plugins: []Plugin{
			{ID: "enabled-plugin", Version: "1", Installed: true, Enabled: true, Permission: "allowed",
				SupportedVersions: []string{"1"}, Namespaces: []string{"alpha"}, ReadState: "readable", Files: 5},
			{ID: "disabled-plugin", Version: "1", Installed: true, Enabled: false, Permission: "allowed",
				SupportedVersions: []string{"2"}, Namespaces: []string{"beta"}, ReadState: "readable", Files: 2, PastFiles: ptr(4)},
			{ID: "not-installed", Version: "1", Installed: false, Enabled: false, Permission: "allowed",
				SupportedVersions: []string{"1"}, Namespaces: []string{"theta"}, ReadState: "not-read"},
			{ID: "version-old", Version: "1", Installed: true, Enabled: true, Permission: "allowed",
				SupportedVersions: []string{"1", "2"}, Namespaces: []string{"gamma"}, ReadState: "readable", Files: 1},
			{ID: "multi", Version: "1", Installed: true, Enabled: true, Permission: "allowed",
				SupportedVersions: []string{"1"}, Namespaces: []string{"epsilon"}, ReadState: "readable", Files: 3},
			{ID: "unreadable", Version: "1", Installed: true, Enabled: true, Permission: "allowed",
				SupportedVersions: []string{"1"}, Namespaces: []string{"zeta"}, ReadState: "unreadable", Files: 7},
			{ID: "providerless", Version: "1", Installed: true, Enabled: true, Permission: "allowed",
				SupportedVersions: []string{"1"}, Namespaces: []string{"delta"}, ReadState: "readable"},
			{ID: "denied", Version: "1", Installed: true, Enabled: true, Permission: "denied",
				SupportedVersions: []string{"1"}, Namespaces: []string{"iota"}, ReadState: "readable", Files: 9},
		},
		Areas: []Area{
			{Key: "area-a", FormatID: "fmt", Namespace: "alpha", Version: "1", Providers: []string{"enabled-plugin"}, ReadState: "readable"},
			{Key: "area-b", FormatID: "fmt", Namespace: "beta", Version: "2", Providers: []string{"disabled-plugin"}, ReadState: "readable"},
			{Key: "area-c", FormatID: "fmt", Namespace: "gamma", Version: "3", Providers: []string{"version-old"}, ReadState: "readable"},
			{Key: "area-d", FormatID: "fmt", Namespace: "delta", Version: "1", ReadState: "readable"},
			{Key: "area-e", FormatID: "fmt", Namespace: "epsilon", Version: "1", Providers: []string{"multi", "other"}, ReadState: "readable"},
			{Key: "area-f", FormatID: "fmt", Namespace: "zeta", Version: "1", Providers: []string{"unreadable"}, ReadState: "readable"},
			{Key: "area-g", FormatID: "fmt", Namespace: "eta", Version: "1", Providers: []string{"ghost"}, ReadState: "readable"},
		},
	}
}

func rowsByID(report Report) map[string]Row {
	out := map[string]Row{}
	for _, row := range report.Rows {
		out[row.PluginID+"|"+row.AreaKey] = row
	}
	return out
}

func TestStatusesAreDistinct(t *testing.T) {
	report, err := Diagnose(fixture(), nil)
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(report)
	want := map[string]string{
		"enabled-plugin|":  StatusEnabled,
		"disabled-plugin|": StatusDisabled,
		"not-installed|":   StatusNotInstalled,
		"version-old|":     StatusVersionUnsupported,
		"providerless|":    StatusProviderUnknown,
		"multi|":           StatusMultipleProviders,
		"unreadable|":      StatusUnreadable,
		"denied|":          StatusPermissionDenied,
	}
	for id, status := range want {
		row, ok := rows[id]
		if !ok {
			t.Fatalf("row %q is missing", id)
		}
		if row.Status != status {
			t.Errorf("row %q status=%q want %q", id, row.Status, status)
		}
	}
	if rows["not-installed|"].Kind != RowReference {
		t.Fatal("a not-installed plugin was not a reference row")
	}
}

func TestMultiProviderAreaIsCountedOnce(t *testing.T) {
	report, err := Diagnose(fixture(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Totals.AreasTotal == nil || *report.Totals.AreasTotal != 7 {
		t.Fatalf("areas total = %v, want 7 (the shared area counted once)", report.Totals.AreasTotal)
	}
	multi := rowsByID(report)["multi|"]
	if multi.Status != StatusMultipleProviders || !contains(multi.ReasonCodes, "multiple-providers:area-e") {
		t.Fatalf("multi=%+v", multi)
	}
	if !contains(multi.AdditionalTargets, "other") {
		t.Fatalf("additional targets=%v", multi.AdditionalTargets)
	}
	// The shared area must not produce an aggregate row of its own.
	aggregates := 0
	for _, row := range report.Rows {
		if row.Kind == RowAggregate && row.AreaKey == "area-e" {
			aggregates++
		}
	}
	if aggregates != 0 {
		t.Fatalf("area-e aggregate rows = %d, want 0 (the area is covered by a plugin row)", aggregates)
	}
	if !containsAreaEvidence(multi, "area-e") {
		t.Fatalf("multi row does not reference area-e: %+v", multi.Evidence)
	}
}

func TestProviderUnknownAggregateIsDistinctFromPluginRow(t *testing.T) {
	report, err := Diagnose(fixture(), nil)
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(report)
	aggregate, ok := rows["|area-g"]
	if !ok {
		t.Fatal("unknown-provider area has no aggregate row")
	}
	if aggregate.Kind != RowAggregate || aggregate.Status != StatusProviderUnknown {
		t.Fatalf("aggregate=%+v", aggregate)
	}
	if !contains(aggregate.AdditionalTargets, "ghost") {
		t.Fatalf("unknown provider %q was not named: %v", "ghost", aggregate.AdditionalTargets)
	}
	if _, duplicate := rows["|area-d"]; duplicate {
		t.Fatal("a covered area produced a duplicate aggregate row")
	}
	// A disabled plugin's area is represented by its plugin row, not by an
	// unknown-provider aggregate.
	if _, wrong := rows["|area-b"]; wrong {
		t.Fatal("a disabled plugin's area was reported as provider-unknown")
	}
}

func TestTotalsCountOnlyReadableProviderAreas(t *testing.T) {
	report, err := Diagnose(fixture(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Totals.AreasTotal == nil || *report.Totals.AreasTotal != 7 {
		t.Fatalf("areas total=%v", report.Totals.AreasTotal)
	}
	if report.Totals.AreasKnown != 2 {
		t.Fatalf("areas known=%d, want 2 (alpha and epsilon only)", report.Totals.AreasKnown)
	}
	if report.Totals.FilesKnown == nil || *report.Totals.FilesKnown != 8 {
		t.Fatalf("files known=%v, want 8", report.Totals.FilesKnown)
	}
	if report.Totals.FilesTotal != nil || report.Confirmed {
		t.Fatalf("ambiguous/incompatible data was confirmed: %+v", report.Totals)
	}
}

func TestPartialScanNeverReportsZeroOrConfirmed(t *testing.T) {
	in := fixture()
	in.Scan.State = ScanPartial
	report, err := Diagnose(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Confirmed {
		t.Fatal("a partial scan reported confirmed")
	}
	if !contains(report.ScanReasons, "scan-incomplete") {
		t.Fatalf("scan reasons=%v", report.ScanReasons)
	}
	if report.Totals.AreasTotal != nil || report.Totals.FilesTotal != nil {
		t.Fatalf("a partial scan produced totals: %+v", report.Totals)
	}
	if report.Totals.AreasKnown == 0 {
		t.Fatal("known areas were dropped instead of marked partial")
	}
	enabled := rowsByID(report)["enabled-plugin|"]
	if enabled.Counts.FilesTotal != nil {
		t.Fatalf("partial scan file total=%v, want unknown", *enabled.Counts.FilesTotal)
	}
	if enabled.Counts.FilesKnown == nil {
		t.Fatal("observed files were dropped")
	}
}

func TestNotRunScanIsNotZero(t *testing.T) {
	in := fixture()
	in.Scan.State = ScanNotRun
	report, err := Diagnose(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Confirmed || report.Totals.FilesTotal != nil || report.Totals.AreasTotal != nil {
		t.Fatalf("not-run scan produced a result: %+v", report.Totals)
	}
	if !contains(report.ScanReasons, "scan-not-run") {
		t.Fatalf("scan reasons=%v", report.ScanReasons)
	}
}

func TestPastCountIsSeparateFromCurrentCount(t *testing.T) {
	report, err := Diagnose(fixture(), nil)
	if err != nil {
		t.Fatal(err)
	}
	disabled := rowsByID(report)["disabled-plugin|"]
	if disabled.Counts.FilesKnown != nil || disabled.Counts.FilesTotal != nil {
		t.Fatalf("a disabled plugin has a current file count: %+v", disabled.Counts)
	}
	if disabled.PastFiles == nil || *disabled.PastFiles != 4 {
		t.Fatalf("past count lost: %v", disabled.PastFiles)
	}
}

func TestUnreadableDoesNotBecomeZero(t *testing.T) {
	report, err := Diagnose(fixture(), nil)
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(report)["unreadable|"]
	if row.Counts.FilesKnown != nil || row.Counts.FilesTotal != nil {
		t.Fatalf("unreadable counts=%+v", row.Counts)
	}
	if !contains(row.ReasonCodes, "counts-unknown") {
		t.Fatalf("reason codes=%v", row.ReasonCodes)
	}
}

func TestDiagnosisIsDeterministicAndTextIndependent(t *testing.T) {
	first, err := Diagnose(fixture(), nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Diagnose(fixture(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("the same input produced different reports")
	}
	catalog := Catalog{}
	for code := range first.Messages {
		catalog[code] = "別の表示文言:" + code
	}
	translated, err := Diagnose(fixture(), catalog)
	if err != nil {
		t.Fatal(err)
	}
	for i := range first.Rows {
		a, b := first.Rows[i], translated.Rows[i]
		if a.Status != b.Status || !reflect.DeepEqual(a.ReasonCodes, b.ReasonCodes) {
			t.Fatalf("display text changed the diagnosis at row %d: %+v vs %+v", i, a, b)
		}
	}
	if reflect.DeepEqual(first.Messages, translated.Messages) {
		t.Fatal("the catalog did not change the display text")
	}
}

func TestObserveSkipsDisabledAndOutOfScope(t *testing.T) {
	in := fixture()
	in.Scan.AllowedTargets = []string{"enabled-plugin"}
	var called []string
	observation, err := Observe(in, func(id string) error {
		called = append(called, id)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(called, []string{"enabled-plugin"}) {
		t.Fatalf("called=%v, want only the allowed enabled plugin", called)
	}
	wantSkip := map[string]string{
		"disabled-plugin": "disabled",
		"not-installed":   "not-installed",
		"denied":          "permission-not-granted",
		"multi":           "out-of-scope",
	}
	for _, skip := range observation.Skipped {
		if want, ok := wantSkip[skip.PluginID]; ok && skip.Reason == want {
			delete(wantSkip, skip.PluginID)
		}
	}
	if len(wantSkip) != 0 {
		t.Fatalf("missing skip reasons: %v (all: %+v)", wantSkip, observation.Skipped)
	}
}

func TestEmptyAllowListAllowsNothing(t *testing.T) {
	in := fixture()
	in.Scan.AllowedTargets = nil
	var called []string
	observation, err := Observe(in, func(id string) error {
		called = append(called, id)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(called) != 0 || len(observation.Observed) != 0 {
		t.Fatalf("an empty allow list invoked %v", called)
	}
}

func TestUnknownScanStateIsRejected(t *testing.T) {
	in := fixture()
	in.Scan.State = "finished"
	if _, err := Diagnose(in, nil); err == nil {
		t.Fatal("an unknown scan state was accepted")
	}
}

func TestDuplicatePluginIsRejected(t *testing.T) {
	in := fixture()
	in.Plugins = append(in.Plugins, in.Plugins[0])
	if _, err := Diagnose(in, nil); err == nil || !strings.Contains(err.Error(), "twice") {
		t.Fatalf("err=%v", err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsAreaEvidence(row Row, key string) bool {
	for _, evidence := range row.Evidence {
		if evidence.Ref == key {
			return true
		}
	}
	return false
}
