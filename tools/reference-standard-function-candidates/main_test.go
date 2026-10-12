package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

func firstWorkspace(t *testing.T) ir.World {
	t.Helper()
	loaded, err := workspace.Load("../../examples/first-workspace")
	if err != nil {
		t.Fatal(err)
	}
	return loaded.World
}

func TestFirstWorkspaceProposalsCannotExecute(t *testing.T) {
	w := firstWorkspace(t)
	report, err := Project(w, "request_log")
	if err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != proposalSchema || report.Authority != "proposal-only" || len(report.Candidates) != 4 {
		t.Fatalf("unexpected report: %+v", report)
	}
	for i, want := range []string{"browse", "record", "create", "update"} {
		c := report.Candidates[i]
		if c.Intent != want || c.Data != "intake.request" || c.Executable || len(c.Missing) == 0 {
			t.Fatalf("candidate %d invented readiness: %+v", i, c)
		}
		if c.DerivedID != "derived.request_log.intake.request."+want {
			t.Fatalf("candidate %d has unstable secondary ID: %s", i, c.DerivedID)
		}
		if len(c.Fields) != 3 {
			t.Fatalf("candidate %d did not preserve source attributes: %+v", i, c.Fields)
		}
	}
	if report.Candidates[0].Kind != application.PatternBrowse ||
		report.Candidates[1].Kind != application.PatternRecord ||
		report.Candidates[2].Kind != application.PatternCommand {
		t.Fatalf("proposal did not reuse existing Pattern types: %+v", report.Candidates)
	}
}

func TestRepeatAndSourceAttributeChange(t *testing.T) {
	w := firstWorkspace(t)
	before, err := Project(w, "request_log")
	if err != nil {
		t.Fatal(err)
	}
	again, err := Project(w, "request_log")
	if err != nil || !reflect.DeepEqual(before, again) {
		t.Fatalf("projection is not deterministic: %+v, %v", again, err)
	}

	for i := range w.Semantics {
		if w.Semantics[i].ID == "intake" {
			w.Semantics[i].Data[0].Attributes = append(w.Semantics[i].Data[0].Attributes,
				semantic.Attribute{ID: "intake.request.priority", Symbol: "priority", TypeRef: "std.code"})
		}
	}
	after, err := Project(w, "request_log")
	if err != nil {
		t.Fatal(err)
	}
	for i, old := range before.Candidates {
		now := after.Candidates[i]
		if old.DerivedID != now.DerivedID || len(now.Fields) != 4 || now.Executable {
			t.Fatalf("attribute change changed IDs or invented execution: %+v", now)
		}
	}
	if len(w.Applications[0].Patterns) != 0 {
		t.Fatal("a proposal modified canonical Application Patterns")
	}
}

func TestManualRecordOverrideAndCollision(t *testing.T) {
	w := firstWorkspace(t)
	app := &w.Applications[0]
	app.Patterns = []application.Pattern{{ID: "request_log.request_record", Kind: application.PatternRecord, Data: "intake.request"}}
	report, err := Project(w, "request_log")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Candidates) != 3 {
		t.Fatalf("an explicit record must not be duplicated: %+v", report.Candidates)
	}
	for _, c := range report.Candidates {
		if c.Intent == "record" {
			t.Fatal("explicit record was replaced")
		}
	}

	app.Patterns = []application.Pattern{{ID: "derived.request_log.intake.request.browse", Kind: application.PatternRecord, Data: "intake.request"}}
	_, err = Project(w, "request_log")
	if err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("expected explicit ID collision refusal, got %v", err)
	}
}

func TestMissingOrAmbiguousAuthorityFailsClosed(t *testing.T) {
	w := firstWorkspace(t)
	if _, err := Project(w, "unknown_app"); err == nil {
		t.Fatal("missing Application must be rejected")
	}
	w.Applications = append(w.Applications, w.Applications[0])
	if _, err := Project(w, "request_log"); err == nil {
		t.Fatal("duplicate Application must be rejected")
	}
	w = firstWorkspace(t)
	w.Applications[0].SemanticUses = []core.CanonicalID{"missing_namespace"}
	if _, err := Project(w, "request_log"); err == nil {
		t.Fatal("missing namespace must not be inferred")
	}
	w = firstWorkspace(t)
	for _, ns := range w.Semantics {
		if ns.ID == "intake" {
			w.Semantics = append(w.Semantics, ns)
			break
		}
	}
	if _, err := Project(w, "request_log"); err == nil {
		t.Fatal("duplicate semantic namespace must be rejected")
	}
}

func TestCLIProducesReadOnlyProjection(t *testing.T) {
	args := []string{"-workspace", "../../examples/first-workspace", "-application", "request_log"}
	var a, b bytes.Buffer
	if err := run(args, &a); err != nil {
		t.Fatal(err)
	}
	if err := run(args, &b); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("same input produced different bytes")
	}
	var report Report
	if err := json.Unmarshal(a.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Candidates) != 4 || report.Authority != "proposal-only" {
		t.Fatalf("CLI output unexpectedly claims runtime authority: %+v", report)
	}
	if err := run([]string{"-workspace", "../../examples/first-workspace", "-application", "invalid!"}, &b); err == nil {
		t.Fatal("bad Application identity must not be accepted")
	}
}
