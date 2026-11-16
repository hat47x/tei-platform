package management_test

import (
	"strings"
	"testing"

	"tei/internal/plugin/management"
)

// Detail rows are looked up by plugin id from a report the CLI has just built.
// FindRow returns the row and an "ok" flag, and the CLI refuses the request when
// ok is false, so the difference between "absent" and "present" has to be
// exactly the id.
func TestFindRowFindsTheRowForItsID(t *testing.T) {
	report := management.Report{
		Environment: "development",
		Rows: []management.Row{
			{PluginID: "tei.a", Version: "1.0.0", Declared: true},
			{PluginID: "tei.b", Version: "2.0.0", Declared: false},
		},
	}

	row, ok := management.FindRow(report, "tei.a")
	if !ok {
		t.Fatal("tei.a is in the report")
	}
	if row.PluginID != "tei.a" || row.Version != "1.0.0" || !row.Declared {
		t.Fatalf("FindRow returned another row: %+v", row)
	}

	// The second row is found by its own id, so the search is not returning the
	// first row regardless.
	row, ok = management.FindRow(report, "tei.b")
	if !ok || row.PluginID != "tei.b" || row.Version != "2.0.0" || row.Declared {
		t.Fatalf("FindRow returned the wrong row for tei.b: %+v ok=%v", row, ok)
	}
}

// An id that is not in the report is reported as absent, and the returned row is
// the zero value rather than an unrelated row. The CLI turns this into an
// invalid-argument refusal, so returning a stale row would describe a plugin the
// caller did not ask about.
func TestFindRowReportsAbsence(t *testing.T) {
	report := management.Report{Rows: []management.Row{{PluginID: "tei.a"}}}

	row, ok := management.FindRow(report, "tei.absent")
	if ok {
		t.Fatalf("an absent id must not be found: %+v", row)
	}
	if row.PluginID != "" || row.Version != "" || row.Declared || row.EnvSelected || row.Resolved {
		t.Fatalf("an absent lookup must return the zero row: %+v", row)
	}

	// An empty report behaves the same, so the caller does not have to special
	// case "no scan ran".
	if _, ok := management.FindRow(management.Report{}, "tei.a"); ok {
		t.Fatal("an empty report contains no rows")
	}
}

// The match is on the exact id. A prefix or a case-folded match would return a
// row for a plugin the caller did not name, which is the failure the CLI's
// refusal exists to prevent.
func TestFindRowMatchesTheExactID(t *testing.T) {
	report := management.Report{Rows: []management.Row{{PluginID: "tei.a"}}}
	for _, near := range []string{"tei", "tei.a.extra", "TEI.A", " tei.a", "tei.a "} {
		if row, ok := management.FindRow(report, near); ok {
			t.Fatalf("%q must not match tei.a: %+v", near, row)
		}
	}
}

// A resolved row says so in one sentence, and a row with reasons lists them.
// DetailMessage is the only prose a caller reads, so it must never claim a
// permission state: a reason code names why a fact is unknown, it is not a
// finding that the permission was granted or refused.
func TestDetailMessageDescribesTheRowWithoutClaimingAPermission(t *testing.T) {
	clear := management.Row{PluginID: "tei.a", Declared: true, EnvSelected: true, Resolved: true}
	message := management.DetailMessage(clear)
	if !strings.Contains(message, "tei.a") {
		t.Fatalf("the message must name the plugin: %q", message)
	}
	if !strings.Contains(message, "揃っています") {
		t.Fatalf("a row with no reasons is described as complete: %q", message)
	}

	blocked := management.Row{PluginID: "tei.b"}
	blocked.ReasonCodes = []string{
		management.ReasonPluginNotDeclared,
		management.ReasonEnvNotSelected,
	}
	message = management.DetailMessage(blocked)
	if !strings.Contains(message, "tei.b") {
		t.Fatalf("the message must name the plugin: %q", message)
	}
	for _, code := range blocked.ReasonCodes {
		if !strings.Contains(message, code) {
			t.Errorf("the message must list %q: %q", code, message)
		}
	}
	// The codes are listed in the order the row holds them, joined readably.
	if !strings.Contains(message, management.ReasonPluginNotDeclared+", "+management.ReasonEnvNotSelected) {
		t.Fatalf("the codes must be joined in order: %q", message)
	}

	// No permission claim appears, in either case.
	for _, claim := range []string{"許可されています", "拒否されました", "granted", "denied"} {
		if strings.Contains(message, claim) {
			t.Fatalf("the detail message must not claim a permission state (%q): %q", claim, message)
		}
	}
}

// A single code is listed without a separator, so the message does not read as
// if a second reason was elided.
func TestDetailMessageWithOneCodeHasNoSeparator(t *testing.T) {
	row := management.Row{PluginID: "tei.a"}
	row.ReasonCodes = []string{management.ReasonScanNotRun}
	message := management.DetailMessage(row)
	if strings.Contains(message, ", ") {
		t.Fatalf("one code must not be followed by a separator: %q", message)
	}
	if !strings.Contains(message, management.ReasonScanNotRun) {
		t.Fatalf("the one code must appear: %q", message)
	}
}
