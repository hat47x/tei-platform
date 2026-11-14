package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The catalog's selected Builder digest is different from a child's own
// report and from the artifact digest. In this opt-in route the selection is
// enforced before exec, without treating resolution as acceptance.
func TestSelfBuildInProcessBindsSelectedBuilderBeforeExecution(t *testing.T) {
	fixture := writeInProcessFixture(t)
	child := buildSelfBuildChild(t, fixture.root)
	childDigest, err := digestOfFile(child)
	if err != nil {
		t.Fatal(err)
	}
	// The existing fixture catalogs a placeholder builder digest. Replace
	// only that builder pin with the exact child image built for this test.
	catalogBytes, err := os.ReadFile(fixture.catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	const placeholder = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	if bytes.Count(catalogBytes, []byte(placeholder)) != 1 {
		t.Fatal("fixture catalog no longer names exactly one expected builder")
	}
	catalogBytes = bytes.Replace(catalogBytes, []byte(placeholder), []byte(childDigest), 1)
	if err := os.WriteFile(fixture.catalogPath, catalogBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	fixture.catalogSha = digestOfBytes(catalogBytes)
	derived := fixture.deriveForTest(t)
	if derived.Plan.Builder.ArtifactDigest != childDigest {
		t.Fatalf("the BuildIntent resolver did not select the real child: %q", derived.Plan.Builder.ArtifactDigest)
	}

	output := filepath.Join(fixture.root, "real-selected-builder-output")
	var raw bytes.Buffer
	code, err := runSelfBuildCommand([]string{
		"-child", child,
		"-workspace", outerWorkspace(t, fixture.root),
		"-intent-file", fixture.intentPath,
		"-build-catalog", fixture.catalogPath,
		"-build-catalog-digest", fixture.catalogSha,
		"-build-source-root", fixture.sourceRoot,
		"-expected-plan-fingerprint", derived.PlanFingerprint,
		"-bind-resolved-builder",
		"-rebuild-check=false",
		"-state", filepath.Join(fixture.root, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output, "-digest", digestOfBytes([]byte("artifact-bytes"))),
	}, &raw)
	if code != 0 || err != nil {
		t.Fatalf("resolved Builder did not invoke the selected child: code=%d err=%v receipt=%s", code, err, raw.String())
	}
	var report selfBuildOuterEvidence
	if err := json.Unmarshal(raw.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.PlanReceipt == nil || report.PlanReceipt.PlanFingerprint != derived.PlanFingerprint ||
		report.ChildRun == nil || !report.ChildRun.ChildPinChecked || !report.ChildRun.ChildPinMatched ||
		report.ChildRun.SelectedChildDigest != derived.Plan.Builder.ArtifactDigest ||
		report.ChildRun.Invoked.ExecutableDigest != childDigest || !report.ChildRun.CrossCheckOK ||
		!report.OK || report.Qualified || report.ChildRun.L3Claimed {
		t.Fatalf("resolved selected Builder was not bound to the actual child and its output: %+v", report)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("selected builder did not produce artifact: %v", err)
	}
}

func TestSelfBuildInProcessRejectsResolvedBuilderMismatchWithoutStartingChild(t *testing.T) {
	fixture := writeInProcessFixture(t) // Catalog deliberately selects a different builder.
	child := buildSelfBuildChild(t, fixture.root)
	output := filepath.Join(fixture.root, "unwritten-artifact")
	var raw bytes.Buffer
	code, err := runSelfBuildCommand([]string{
		"-child", child,
		"-workspace", outerWorkspace(t, fixture.root),
		"-intent-file", fixture.intentPath,
		"-build-catalog", fixture.catalogPath,
		"-build-catalog-digest", fixture.catalogSha,
		"-build-source-root", fixture.sourceRoot,
		"-bind-resolved-builder",
		"-rebuild-check=false",
		"-state", filepath.Join(fixture.root, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output, "-digest", digestOfBytes([]byte("artifact-bytes"))),
	}, &raw)
	if code != 1 || err != nil {
		t.Fatalf("mismatched resolved Builder was not refused: code=%d err=%v receipt=%s", code, err, raw.String())
	}
	var report selfBuildOuterEvidence
	if err := json.Unmarshal(raw.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.PlanReceipt == nil || report.PlanReceipt.State != intentResolutionResolved ||
		report.Error == nil || report.Error.Code != "SELF_BUILD_BUILDER_SELECTION_MISMATCH" ||
		!strings.Contains(report.Error.Message, "selected child builder digest mismatch") ||
		report.ChildRun != nil || report.Qualified {
		t.Fatalf("selected B / actual A was not preserved as a pre-start refusal: %+v", report)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("refused child wrote output or output is inaccessible: %v", err)
	}
}

func TestSelfBuildCannotRequestResolvedBuilderBindingWithoutInProcessCatalog(t *testing.T) {
	var out bytes.Buffer
	code, err := runSelfBuildCommand([]string{"-bind-resolved-builder"}, &out)
	if code != 2 || err == nil || !strings.Contains(err.Error(), "requires -build-catalog") {
		t.Fatalf("binding without resolution was not refused: code=%d err=%v", code, err)
	}
	if out.Len() != 0 {
		t.Fatalf("refused request emitted a misleading success report: %s", out.String())
	}
}
