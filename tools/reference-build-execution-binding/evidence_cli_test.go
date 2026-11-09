package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os/exec"
	"testing"
)

// Exercise the public command rather than only the in-process Evidence builder.
// The CI receipt for this test therefore covers the emitted JSON contract.
func TestReferenceEvidenceCommandEmitsDeterministicJSON(t *testing.T) {
	run := func() []byte {
		t.Helper()
		command := exec.Command("go", "run", ".")
		var stderr bytes.Buffer
		command.Stderr = &stderr
		stdout, err := command.Output()
		if err != nil {
			t.Fatalf("reference Evidence command failed: %v; stderr=%s", err, stderr.String())
		}
		if stderr.Len() != 0 {
			t.Fatalf("reference Evidence command wrote unexpected stderr: %s", stderr.String())
		}
		return stdout
	}

	first := run()
	if second := run(); !bytes.Equal(first, second) {
		t.Fatal("reference Evidence command produced different JSON across invocations")
	}

	var actual referenceEvidence
	decoder := json.NewDecoder(bytes.NewReader(first))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&actual); err != nil {
		t.Fatalf("reference Evidence output is not a complete known JSON schema: %v", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("reference Evidence output contains data after the JSON object: %v", err)
	}

	expected, err := buildReferenceEvidence()
	if err != nil {
		t.Fatal(err)
	}
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actualJSON, expectedJSON) {
		t.Fatal("CLI Evidence JSON differs from the independently constructed Reference Evidence")
	}

	if actual.Schema != "tei.reference.build-execution-binding/v0" ||
		len(actual.SharedPluginBindings) != 2 ||
		len(actual.SharedPluginBindingFingerprints) != 2 ||
		actual.SharedPluginBindingFingerprints[0] == actual.SharedPluginBindingFingerprints[1] {
		t.Fatal("CLI Evidence lost the two distinct logical Plugin bindings")
	}
	if actual.SharedPluginBindings[0].BuildExecutionResultFingerprint != actual.SharedResultFingerprint ||
		actual.SharedPluginBindings[1].BuildExecutionResultFingerprint != actual.SharedResultFingerprint {
		t.Fatal("shared Plugin bindings do not refer to the same exact build result")
	}
	if actual.MultiOutputComparison.Reproducible ||
		len(actual.MultiOutputComparison.MissingFromSecond) == 0 ||
		len(actual.MultiOutputComparison.ExtraInSecond) == 0 ||
		len(actual.MultiOutputComparison.Changed) == 0 {
		t.Fatal("CLI Evidence lost the multi-output non-reproducibility details")
	}
	if len(actual.BuildGraphOrder) != 2 ||
		actual.BuildGraphOrder[0] != "stage1" ||
		actual.BuildGraphOrder[1] != "stage2" ||
		len(actual.BuildGraph.Seeds) != 1 ||
		actual.BuildGraph.Seeds[0].TrustBoundary != "external-stage0-seed" ||
		actual.RuntimeEngineBindingFingerprint == "" {
		t.Fatal("CLI Evidence lost its bootstrap or non-Plugin Runtime Engine identity")
	}

	foundLimit := false
	for _, limit := range actual.AuthorityLimits {
		if limit == "stage0-boundary-is-not-trust-acceptance" {
			foundLimit = true
		}
	}
	if !foundLimit {
		t.Fatal("CLI Evidence omitted the boundary against treating external seed provenance as authority")
	}
	if len(actual.BuildGraph.Edges) != 1 ||
		actual.BuildGraph.Edges[0].FromNode != "stage1" ||
		actual.BuildGraph.Edges[0].ToNode != "stage2" ||
		actual.BuildGraph.Edges[0].OutputRole != "builder-artifact" ||
		normalizeDigest(actual.BuildGraph.Seeds[0].ArtifactDigest) !=
			normalizeDigest(actual.BootstrapStage1Plan.Builder.ArtifactDigest) {
		t.Fatal("CLI Evidence lost the exact external seed and stage1-to-stage2 builder artifact binding")
	}
}
