package transform_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/plugin"
	"tei/internal/plugin/inputprofile"
	"tei/internal/plugin/transform"
)

// The v2 adapter may make two different ingress byte streams yield identical
// SB-1 bytes. Their input evidence must still record which byte stream was
// admitted; output equality is not evidence of identical Host decisions.
func TestV2HostIngressEquivalentOutputKeepsDistinctOriginalEvidence(t *testing.T) {
	def, plan, source, selection, explicitArrays := admissionFixture(t)
	canonicalRaw, err := json.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(canonicalRaw, explicitArrays) || !bytes.Contains(explicitArrays, []byte(`"provides":[]`)) {
		t.Fatal("fixture does not distinguish Go-canonical and explicit-array ingress")
	}
	first, err := inputprofile.PreparePluginPackageInputs(selection, source, plan, canonicalRaw)
	if err != nil {
		t.Fatalf("canonical v2 ingress: %v", err)
	}
	second, err := inputprofile.PreparePluginPackageInputs(selection, source, plan, explicitArrays)
	if err != nil {
		t.Fatalf("explicit-array v2 ingress: %v", err)
	}
	if first.Evidence.OriginalInputDigest == second.Evidence.OriginalInputDigest ||
		first.Evidence.CanonicalInputDigest != second.Evidence.CanonicalInputDigest ||
		first.Evidence.SourceFingerprint != second.Evidence.SourceFingerprint ||
		first.Evidence.BuildPlanFingerprint != second.Evidence.BuildPlanFingerprint ||
		first.Evidence.PluginDefinitionFingerprint != second.Evidence.PluginDefinitionFingerprint {
		t.Fatalf("distinct raw ingress must have distinct original evidence but same canonical binding: first=%+v second=%+v", first.Evidence, second.Evidence)
	}
	oracle, err := plugin.MaterializePluginPackage(def, plan)
	if err != nil {
		t.Fatal(err)
	}
	var observations []inputprofile.PackageOutputObservation
	for _, prepared := range []inputprofile.PreparedPackageInputs{first, second} {
		result, err := transform.Evaluate(source, prepared.Inputs)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(result.Output.Bytes, oracle) {
			t.Fatal("v2 ingress did not reproduce exact Go package bytes")
		}
		observed, err := inputprofile.ObservePackageOutput(prepared.Evidence, result)
		if err != nil {
			t.Fatal(err)
		}
		observations = append(observations, observed)
	}
	if observations[0].OutputDigest != observations[1].OutputDigest ||
		observations[0].Input.OriginalInputDigest == observations[1].Input.OriginalInputDigest ||
		observations[0].Input.CanonicalInputDigest != observations[1].Input.CanonicalInputDigest {
		t.Fatalf("output equality incorrectly erased distinct ingress provenance: %+v", observations)
	}
	// The shipped v1 Source must still refuse the explicit-array form. A
	// successful v2 admission must not alter unqualified legacy evaluation.
	legacyInput, err := transform.ParseJSON(explicitArrays)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transform.Evaluate(source, map[string]transform.Value{
		"buildPlan":        first.Inputs["buildPlan"],
		"pluginDefinition": legacyInput,
	}); err == nil || !strings.Contains(err.Error(), "assert-equal failed") {
		t.Fatalf("legacy evaluator silently normalized an explicit array: %v", err)
	}
}
