package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/studiohost"
)

// This calls the actual product CLI from a separate OS process. A selected
// Markdown Rule is readable, but it is not a published Function Entry.
// Neither CLI inspection nor Source Index publication is an execution grant.
func TestProductInspectCLISelectedMarkdownRuleIsNotCapability(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build actual product CLI: %v\n%s", err, output)
	}

	root := filepath.Join("..", "..", "internal", "workspace", "testdata", "tei-md-rule")
	const ruleID = "sales.order_line.quantity_positive"
	const sourcePath = "semantic/sales/order-line-rules.tei.md"
	invoke := func(kind, id string, expectedExit int) cliInspectionReport {
		t.Helper()
		cmd := exec.Command(binary, "inspect", "-workspace", root, "-subject-kind", kind, "-subject-id", id)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		exit := 0
		if err != nil {
			var status *exec.ExitError
			if !errors.As(err, &status) {
				t.Fatalf("start actual inspect: %v", err)
			}
			exit = status.ExitCode()
		}
		if exit != expectedExit || stderr.Len() != 0 {
			t.Fatalf("inspect %s:%s exit=%d stderr=%q, want exit=%d", kind, id, exit, stderr.String(), expectedExit)
		}
		var report cliInspectionReport
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatalf("inspect %s:%s returned non-JSON: %v; stdout=%q", kind, id, err, stdout.String())
		}
		if report.SchemaVersion != "tei.cli.inspect/v0" {
			t.Fatalf("unexpected actual CLI inspect contract: %+v", report)
		}
		return report
	}

	rule := invoke("semantic-validation", ruleID, 0)
	if !rule.OK || rule.Result == nil || rule.Error != nil {
		t.Fatalf("selected Markdown Rule was not inspected: %+v", rule)
	}
	subject := sourceindex.Subject{Kind: "semantic-validation", ID: ruleID}
	seen := rule.Result
	// Fix the direct Application Operation baseline before introducing a
	// child Backend transport. The OS-process CLI must preserve the entire
	// read-only projection, including Source and authority limitations.
	direct, err := operations.InspectSubject(root, subject)
	if err != nil {
		t.Fatalf("direct inspect Operation: %v", err)
	}
	if !reflect.DeepEqual(*seen, direct) {
		t.Fatalf("actual CLI process diverged from direct inspect Operation:\nCLI: %+v\nDirect: %+v", *seen, direct)
	}
	// The Wails-independent Studio Host must project the same meaning and
	// Source as the real CLI. This checks its typed Host boundary, not a
	// WebView or HTTP transport, and grants no disclosure or write authority.
	studio, err := studiohost.NewReadOnly([]string{root})
	if err != nil {
		t.Fatalf("create read-only Studio Host: %v", err)
	}
	studioRead := studio.InspectSubject("workspace-1", subject.Kind, subject.ID)
	if !studioRead.OK || studioRead.Value == nil || studioRead.Failure != nil {
		t.Fatalf("Studio Host could not inspect selected Rule: %+v", studioRead)
	}
	if !reflect.DeepEqual(*seen, *studioRead.Value) {
		t.Fatalf("CLI and Studio Host returned different inspection projections:\nCLI: %+v\nStudio: %+v", *seen, *studioRead.Value)
	}
	if seen.Subject != subject || seen.Effect != "read-only" ||
		seen.SourceStatus != "indexed" || len(seen.AuthorityLimits) == 0 {
		t.Fatalf("CLI Rule inspection lost subject/source/authority: %+v", seen)
	}
	var canonical struct {
		ID        string `json:"id"`
		Attribute string `json:"attribute"`
	}
	if err := json.Unmarshal(seen.Canonical, &canonical); err != nil {
		t.Fatal(err)
	}
	if canonical.ID != ruleID || canonical.Attribute != "sales.order_line.quantity" {
		t.Fatalf("unexpected canonical Rule meaning: %+v", canonical)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	if len(seen.Sources.References) != 1 || len(seen.Sources.Artifacts) != 1 {
		t.Fatalf("Rule has no unique original Source: %+v", seen.Sources)
	}
	ref := seen.Sources.References[0]
	artifact := seen.Sources.Artifacts[0]
	if ref.Subject != subject || ref.ArtifactPath != sourcePath ||
		ref.Role != sourceindex.RoleDefinition ||
		artifact.Path != sourcePath || artifact.Fingerprint != sourceindex.Fingerprint(raw) {
		t.Fatalf("CLI Rule Source did not match actual Markdown bytes: %+v", seen.Sources)
	}
	if err := ref.Range.Validate(raw); err != nil {
		t.Fatalf("invalid Markdown Rule source range: %v", err)
	}
	start, end := ref.Range.Start.Offset, ref.Range.End.Offset
	if string(raw[start:end]) != ruleID {
		t.Fatalf("CLI Rule source anchor=%q, want %q", raw[start:end], ruleID)
	}

	// Same ID under a different Subject kind cannot be inferred from the
	// success of the first, separate-process read-only inspection.
	for _, tc := range []struct {
		kind, id string
	}{
		{"semantic-capability", ruleID},
		{"semantic-validation", "sales.order_line.unit_price_positive"}, // text fence only
	} {
		missing := invoke(tc.kind, tc.id, 1)
		if missing.OK || missing.Result != nil || missing.Error == nil ||
			missing.Error.Code != operations.InspectionNotFound {
			t.Fatalf("inspect %s:%s must fail closed: %+v", tc.kind, tc.id, missing)
		}
		studioFailure := studio.InspectSubject("workspace-1", tc.kind, tc.id)
		if studioFailure.OK || studioFailure.Value != nil || studioFailure.Failure == nil ||
			studioFailure.Failure.Code != missing.Error.Code ||
			studioFailure.Failure.Kind != "business_rejection" ||
			studioFailure.Failure.Retryable != "not_retryable" ||
			studioFailure.Failure.EffectUnknown {
			t.Fatalf("Studio Host must preserve CLI refusal class without claiming an effect: %+v; CLI=%+v", studioFailure, missing)
		}
		_, directErr := operations.InspectSubject(root, sourceindex.Subject{Kind: tc.kind, ID: tc.id})
		var classified *operations.InspectionError
		if !errors.As(directErr, &classified) || classified.Code != missing.Error.Code {
			t.Fatalf("inspect %s:%s direct/CLI failure mismatch: direct=%v CLI=%+v", tc.kind, tc.id, directErr, missing.Error)
		}
	}
	// A browser-supplied workspace ID does not become an authorized root,
	// even when the Subject exists in a different allowed Workspace.
	denied := studio.InspectSubject("workspace-2", subject.Kind, subject.ID)
	if denied.OK || denied.Value != nil || denied.Failure == nil ||
		denied.Failure.Code != "WORKSPACE_NOT_ALLOWED" ||
		denied.Failure.Kind != "business_rejection" || denied.Failure.EffectUnknown {
		t.Fatalf("unlisted Studio Workspace must fail before inspection: %+v", denied)
	}

}
