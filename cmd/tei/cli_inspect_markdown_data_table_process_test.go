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
)

// This is the actual tei inspect OS process, not a parser or a direct
// reference Reader. The same marked Markdown Data document must be inspected
// through both the CLI transport and the existing Application Operation.
// Source references are read-only evidence, never edit or execution grants.
func TestProductInspectCLIMarkedDataTableMatchesDirectOperation(t *testing.T) {
	fixture := filepath.Join("..", "..", "internal", "workspace", "testdata", "tei-md-data-table")
	const sourcePath = "semantic/organization/transfer-request.tei.md"
	root := t.TempDir()
	copySource := func(relative string) []byte {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		return raw
	}
	copySource("tei.workspace.yaml")
	original := copySource(sourcePath)

	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build product CLI: %v\n%s", err, output)
	}
	invoke := func(subject sourceindex.Subject, wantExit int) cliInspectionReport {
		t.Helper()
		cmd := exec.Command(binary, "inspect", "-workspace", root,
			"-subject-kind", subject.Kind, "-subject-id", subject.ID)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		runErr := cmd.Run()
		exit := 0
		if runErr != nil {
			var status *exec.ExitError
			if !errors.As(runErr, &status) {
				t.Fatalf("start product inspect %v: %v", subject, runErr)
			}
			exit = status.ExitCode()
		}
		if exit != wantExit || stderr.Len() != 0 {
			t.Fatalf("inspect %v exit=%d stderr=%q, want exit=%d", subject, exit, stderr.String(), wantExit)
		}
		var report cliInspectionReport
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatalf("non-JSON product inspect %v: %v; stdout=%q", subject, err, stdout.String())
		}
		if report.SchemaVersion != "tei.cli.inspect/v0" {
			t.Fatalf("unexpected inspect schema for %v: %+v", subject, report)
		}
		return report
	}

	cases := []struct {
		subject sourceindex.Subject
		token   string
	}{
		{sourceindex.Subject{Kind: "semantic-data", ID: "organization.transfer_request"}, "TransferRequest"},
		{sourceindex.Subject{Kind: "semantic-attribute", ID: "organization.transfer_request.effective_to"}, "effectiveTo"},
		{sourceindex.Subject{Kind: "semantic-validation", ID: "organization.transfer_request.allocation_positive"}, "organization.transfer_request.allocation_positive"},
	}
	for _, tc := range cases {
		t.Run(tc.subject.Kind, func(t *testing.T) {
			got := invoke(tc.subject, 0)
			if !got.OK || got.Result == nil || got.Error != nil {
				t.Fatalf("CLI did not inspect the declared subject: %+v", got)
			}
			direct, err := operations.InspectSubject(root, tc.subject)
			if err != nil {
				t.Fatalf("direct Application Operation: %v", err)
			}
			if !reflect.DeepEqual(*got.Result, direct) {
				t.Fatalf("actual CLI and direct Operation diverged:\nCLI: %+v\nDirect: %+v", *got.Result, direct)
			}
			if err := operations.ValidateInspectionProjection(tc.subject, *got.Result); err != nil {
				t.Fatalf("CLI projection invalid: %v", err)
			}
			if got.Result.Effect != "read-only" || got.Result.SourceStatus != "indexed" ||
				!reflect.DeepEqual(got.Result.AuthorityLimits, operations.InspectionAuthorityLimits()) {
				t.Fatalf("CLI escalated or lost inspection authority: %+v", got.Result)
			}
			var canonical struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(got.Result.Canonical, &canonical); err != nil || canonical.ID != tc.subject.ID {
				t.Fatalf("CLI returned the wrong typed Subject: id=%q err=%v", canonical.ID, err)
			}
			if len(got.Result.Sources.References) != 1 || len(got.Result.Sources.Artifacts) != 1 {
				t.Fatalf("expected one declaration source: %+v", got.Result.Sources)
			}
			ref, artifact := got.Result.Sources.References[0], got.Result.Sources.Artifacts[0]
			if ref.Subject != tc.subject || ref.Role != sourceindex.RoleDefinition ||
				ref.ArtifactPath != sourcePath || artifact.Path != sourcePath ||
				artifact.Fingerprint != sourceindex.Fingerprint(original) {
				t.Fatalf("CLI lost marked Markdown Source identity: ref=%+v artifact=%+v", ref, artifact)
			}
			if err := ref.Range.Validate(original); err != nil {
				t.Fatalf("stale original byte anchor: %v", err)
			}
			if bytes.Count(original, []byte(tc.token)) != 1 ||
				ref.Range.Start.Offset != bytes.Index(original, []byte(tc.token)) {
				t.Fatalf("CLI returned wrong original Data/Rule anchor for %v: %+v", tc.subject, ref.Range)
			}
		})
	}

	// An unmarked example table does not create Data, and the Rule never
	// becomes a public Capability merely because it is inspectable.
	for _, subject := range []sourceindex.Subject{
		{Kind: "semantic-data", ID: "organization.example_record"},
		{Kind: "semantic-capability", ID: "organization.transfer_request.allocation_positive"},
	} {
		report := invoke(subject, 1)
		if report.OK || report.Result != nil || report.Error == nil ||
			report.Error.Code != operations.InspectionNotFound {
			t.Fatalf("wrong rejection of nondefinition %v: %+v", subject, report)
		}
		_, directErr := operations.InspectSubject(root, subject)
		var classified *operations.InspectionError
		if !errors.As(directErr, &classified) || classified.Code != report.Error.Code {
			t.Fatalf("direct and CLI rejection differ for %v: %v / %+v", subject, directErr, report.Error)
		}
	}

	// The CLI must re-read the current Source rather than continuing to serve
	// the earlier descriptive Index after a prose-only revision.
	subject := cases[0].subject
	before := invoke(subject, 0)
	edited := append(append([]byte(nil), original...), []byte("\n説明だけを追記しました。\n")...)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(sourcePath)), edited, 0o644); err != nil {
		t.Fatal(err)
	}
	after := invoke(subject, 0)
	if !after.OK || after.Result == nil || before.Result == nil {
		t.Fatalf("prose revision broke product inspect: before=%+v after=%+v", before, after)
	}
	if !bytes.Equal(before.Result.Canonical, after.Result.Canonical) ||
		before.Result.WorldFingerprint != after.Result.WorldFingerprint {
		t.Fatal("prose-only edit changed typed meaning")
	}
	if before.Result.SourceIndexFingerprint == after.Result.SourceIndexFingerprint ||
		reflect.DeepEqual(before.Result.Sources.Artifacts, after.Result.Sources.Artifacts) {
		t.Fatal("CLI served stale Source evidence after prose-only revision")
	}
	if !reflect.DeepEqual(before.Result.Sources.References, after.Result.Sources.References) {
		t.Fatal("trailing prose unexpectedly moved Data definition Source")
	}
}
