package main

import (
	"os"
	"path/filepath"
	"testing"
)

// §4.36 closes "the builder built from a definition the plan does not name" only
// for a builder that implements the refusal. This test makes the builder say it
// read something else and checks that the attempt fails anyway, because this side
// compared the two values.

func TestEntryFailsWhenTheBuilderSaysItReadTheWrongDefinition(t *testing.T) {
	fixture := writeInProcessFixture(t)
	child := buildSelfBuildChild(t, fixture.root)
	output := filepath.Join(fixture.root, "artifact")
	derived := fixture.deriveForTest(t)

	report, code, err := runInProcessShape(t,
		"-child", child,
		"-workspace", outerWorkspace(t, fixture.root),
		"-intent", "tei.plugin.inprocess_probe",
		"-intent-file", fixture.intentPath,
		"-build-catalog", fixture.catalogPath,
		"-build-catalog-digest", fixture.catalogSha,
		"-build-source-root", fixture.sourceRoot,
		"-expected-plan-fingerprint", derived.PlanFingerprint,
		"-state", filepath.Join(fixture.root, "state"),
		"-output", output,
		"-child-arg", childArgs(t,
			"-artifact", output,
			"-digest", digestOfBytes([]byte("artifact-bytes")),
			"-report-definition", "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		),
	)
	if err == nil && code == 0 {
		t.Fatalf("a builder that reported a different definition was accepted: %+v", report)
	}
	if code == 0 {
		t.Fatal("the attempt did not fail")
	}
	if report.DefinitionBinding == nil {
		t.Fatalf("no definition binding was recorded: %+v", report)
	}
	if report.DefinitionBinding.State != definitionBindingFailed {
		t.Fatalf("state = %q, want %q (reason %q)", report.DefinitionBinding.State, definitionBindingFailed, report.DefinitionBinding.Reason)
	}
	// The plan's own value must be in the record, so a reader can see both sides.
	if report.DefinitionBinding.PlannedDigest == "" {
		t.Fatal("the record does not name the planned digest")
	}
	if report.DefinitionBinding.PlannedDigest == report.DefinitionBinding.ReportedDigest {
		t.Fatal("the two sides are equal, so this attempt did not exercise the comparison")
	}
}

// The same attempt with an agreeing report passes, so the check above is not
// failing everything.
func TestEntryAcceptsABuilderThatNamesThePlannedDefinition(t *testing.T) {
	fixture := writeInProcessFixture(t)
	child := buildSelfBuildChild(t, fixture.root)
	output := filepath.Join(fixture.root, "artifact")
	derived := fixture.deriveForTest(t)

	// The plan's digest for the definition source, read the same way the entry
	// reads it: from the declaration's resolved source.
	intentBytes, err := os.ReadFile(fixture.intentPath)
	if err != nil {
		t.Fatal(err)
	}
	_ = intentBytes

	report, code, err := runInProcessShape(t,
		"-child", child,
		"-workspace", outerWorkspace(t, fixture.root),
		"-intent", "tei.plugin.inprocess_probe",
		"-intent-file", fixture.intentPath,
		"-build-catalog", fixture.catalogPath,
		"-build-catalog-digest", fixture.catalogSha,
		"-build-source-root", fixture.sourceRoot,
		"-expected-plan-fingerprint", derived.PlanFingerprint,
		"-state", filepath.Join(fixture.root, "state"),
		"-output", output,
		"-child-arg", childArgs(t,
			"-artifact", output,
			"-digest", digestOfBytes([]byte("artifact-bytes")),
			"-report-definition", plannedDefinitionDigestForTest(t, fixture),
		),
	)
	if code != 0 || err != nil {
		t.Fatalf("an agreeing builder was refused: code=%d err=%v report=%+v", code, err, report)
	}
	if report.DefinitionBinding == nil || report.DefinitionBinding.State != definitionBindingBound {
		t.Fatalf("definition binding = %+v, want bound", report.DefinitionBinding)
	}
}

// plannedDefinitionDigestForTest derives the digest the entry's plan records, using
// the same API the entry uses.
func plannedDefinitionDigestForTest(t *testing.T, fixture inProcessFixture) string {
	t.Helper()
	receipt := fixture.deriveForTest(t)
	for _, source := range receipt.Plan.Sources {
		if source.Kind == "plugin-definition" {
			return source.Digest
		}
	}
	t.Fatal("the derived plan records no definition source")
	return ""
}
