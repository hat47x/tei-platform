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

func admissionFixture(t *testing.T) (plugin.Definition, plugin.BuildPlan, transform.Source, inputprofile.BuildInputSelection, []byte) {
	t.Helper()
	def, plan := transformPackageFixture()
	def.Provides = nil
	def.Requires = nil
	def.External = nil
	plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(def)
	source := referencePluginPackageTransformSource(t)
	sourceFP, err := transform.SourceFingerprint(source)
	if err != nil {
		t.Fatal(err)
	}
	planFP, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	var reordered map[string]any
	if err := json.Unmarshal(raw, &reordered); err != nil {
		t.Fatal(err)
	}
	reordered["provides"] = []any{}
	reordered["requires"] = []any{}
	reordered["externalRequirements"] = []any{}
	raw, err = json.Marshal(reordered)
	if err != nil {
		t.Fatal(err)
	}
	return def, plan, source, inputprofile.BuildInputSelection{
		Profile: inputprofile.PluginDefinitionJSONV2,
		LockedSourceFingerprint: sourceFP,
		LockedBuildPlanFingerprint: planFP,
	}, raw
}

func TestExplicitHostInputBindingProducesExactLegacyEnginePackage(t *testing.T) {
	def, plan, source, selection, raw := admissionFixture(t)
	oracle, err := plugin.MaterializePluginPackage(def, plan)
	if err != nil {
		t.Fatal(err)
	}
	// Unqualified legacy callers still see the original noncanonical JSON
	// and cannot receive automatic normalization.
	legacy, err := transform.ParseJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transform.Evaluate(source, map[string]transform.Value{
		"pluginDefinition": legacy,
		"buildPlan": valueFromGoJSON(t, plan),
	}); err == nil || !strings.Contains(err.Error(), "assert-equal failed") {
		t.Fatalf("legacy v1 evaluator accepted raw explicit arrays: %v", err)
	}
	prepared, err := inputprofile.PreparePluginPackageInputs(selection, source, plan, raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.Inputs) != 2 || prepared.Evidence.Schema != inputprofile.PackageInputBindingSchema ||
		prepared.Evidence.Profile != selection.Profile ||
		prepared.Evidence.SourceFingerprint != selection.LockedSourceFingerprint ||
		prepared.Evidence.BuildPlanFingerprint != selection.LockedBuildPlanFingerprint ||
		prepared.Evidence.PluginDefinitionFingerprint != plan.PluginDefinitionFingerprint ||
		prepared.Evidence.Plugin != plan.Plugin ||
		prepared.Evidence.OriginalInputDigest == prepared.Evidence.CanonicalInputDigest {
		t.Fatalf("Host input binding omitted exact evidence: %+v", prepared.Evidence)
	}
	result, err := transform.Evaluate(source, prepared.Inputs)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.Output.Bytes, oracle) {
		t.Fatal("explicit Host v2 ingress did not match historical Source Go output bytes")
	}
	observed, err := inputprofile.ObservePackageOutput(prepared.Evidence, result)
	if err != nil {
		t.Fatal(err)
	}
	if observed.Schema != inputprofile.PackageOutputObservationSchema ||
		observed.OutputDigest != plugin.BuildOutputDigest(oracle) ||
		observed.Steps != result.Steps ||
		observed.Input != prepared.Evidence {
		t.Fatalf("actual SB-1 output was not linked to the exact ingress: %+v", observed)
	}
	// Caller-owned raw JSON is immutable and retains the explicit-empty shape.
	if !bytes.Contains(raw, []byte(`"provides":[]`)) {
		t.Fatal("Host admission altered the original input")
	}
}

func TestHostBuildInputBindingRejectsWrongLocksPlansAndProfiles(t *testing.T) {
	_, plan, source, selection, raw := admissionFixture(t)
	cases := []struct {
		name string
		edit func(*inputprofile.BuildInputSelection, *plugin.BuildPlan, *[]byte)
		want string
	}{
		{name: "no explicit opt-in", edit: func(s *inputprofile.BuildInputSelection, _ *plugin.BuildPlan, _ *[]byte) { s.Profile = "" }, want: "unsupported package input profile"},
		{name: "unsupported opt-in", edit: func(s *inputprofile.BuildInputSelection, _ *plugin.BuildPlan, _ *[]byte) { s.Profile = "tei.plugin-definition-json/v3" }, want: "unsupported package input profile"},
		{name: "unapproved Source", edit: func(s *inputprofile.BuildInputSelection, _ *plugin.BuildPlan, _ *[]byte) { s.LockedSourceFingerprint = "source-mismatch" }, want: "exact locked SB-1 Source fingerprint"},
		{name: "missing locked plan", edit: func(s *inputprofile.BuildInputSelection, _ *plugin.BuildPlan, _ *[]byte) { s.LockedBuildPlanFingerprint = "" }, want: "Host-approved BuildPlan fingerprint"},
		{name: "wrong locked plan", edit: func(s *inputprofile.BuildInputSelection, _ *plugin.BuildPlan, _ *[]byte) { s.LockedBuildPlanFingerprint = "plan-mismatch" }, want: "Host-approved BuildPlan fingerprint"},
		{name: "plan source changed after lock", edit: func(_ *inputprofile.BuildInputSelection, p *plugin.BuildPlan, _ *[]byte) { p.Sources[0].Locator = "other.tei" }, want: "Host-approved BuildPlan fingerprint"},
		{name: "unsupported output kind", edit: func(s *inputprofile.BuildInputSelection, p *plugin.BuildPlan, _ *[]byte) {
			p.OutputKind = plugin.BuildOutputKind("other")
			s.LockedBuildPlanFingerprint, _ = plugin.BuildPlanFingerprint(*p)
		}, want: "requires plugin-package output kind"},
		{name: "definition different from plan", edit: func(_ *inputprofile.BuildInputSelection, _ *plugin.BuildPlan, raw *[]byte) {
			*raw = bytes.Replace(*raw, []byte(`"id":"tei.transform.package_test"`), []byte(`"id":"tei.other_package"`), 1)
		}, want: "canonical Plugin identity does not match"},
		{name: "definition content differs from plan", edit: func(_ *inputprofile.BuildInputSelection, _ *plugin.BuildPlan, raw *[]byte) {
			*raw = bytes.Replace(*raw, []byte(`"version":"1.0.0"`), []byte(`"version":"1.0.1"`), 1)
		}, want: "canonical Plugin identity does not match"},
		{name: "malformed JSON before execution", edit: func(_ *inputprofile.BuildInputSelection, _ *plugin.BuildPlan, raw *[]byte) { *raw = []byte(`{"provides":[],"provides":[]}`) }, want: "declared more than once"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sel := selection
			currentPlan := plan
			currentPlan.Sources = append([]plugin.BuildSource(nil), plan.Sources...)
			currentRaw := append([]byte(nil), raw...)
			tc.edit(&sel, &currentPlan, &currentRaw)
			_, err := inputprofile.PreparePluginPackageInputs(sel, source, currentPlan, currentRaw)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Host admitted invalid input: %v (expected %q)", err, tc.want)
			}
		})
	}
}

func TestHostBuildOutputObservationRejectsUnboundOrAlteredResult(t *testing.T) {
	def, plan, source, selection, raw := admissionFixture(t)
	prepared, err := inputprofile.PreparePluginPackageInputs(selection, source, plan, raw)
	if err != nil {
		t.Fatal(err)
	}
	result, err := transform.Evaluate(source, prepared.Inputs)
	if err != nil {
		t.Fatal(err)
	}
	oracle, err := plugin.MaterializePluginPackage(def, plan)
	if err != nil || !bytes.Equal(oracle, result.Output.Bytes) {
		t.Fatal("fixture Go and TEI results disagree")
	}
	t.Run("not an execution result", func(t *testing.T) {
		modified := result
		modified.Steps = 0
		if _, err := inputprofile.ObservePackageOutput(prepared.Evidence, modified); err == nil {
			t.Fatal("zero-step output was observed as an execution")
		}
	})
	t.Run("wrong output kind", func(t *testing.T) {
		modified := result
		modified.Output = transform.String("not a package")
		if _, err := inputprofile.ObservePackageOutput(prepared.Evidence, modified); err == nil {
			t.Fatal("non-byte output was accepted")
		}
	})
	t.Run("wrong input fingerprint", func(t *testing.T) {
		modified := prepared.Evidence
		modified.BuildPlanFingerprint = strings.Repeat("a", 64)
		if _, err := inputprofile.ObservePackageOutput(modified, result); err == nil {
			t.Fatal("output was incorrectly linked to a different BuildPlan")
		}
	})
	t.Run("mutated output definition", func(t *testing.T) {
		var manifest plugin.PluginPackageManifest
		if err := json.Unmarshal(oracle, &manifest); err != nil {
			t.Fatal(err)
		}
		manifest.Definition.Version = "2.0.0"
		altered, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		modified := result
		modified.Output = transform.Bytes(altered)
		if _, err := inputprofile.ObservePackageOutput(prepared.Evidence, modified); err == nil {
			t.Fatal("tampered manifest was accepted as an observed build result")
		}
	})
	t.Run("missing input evidence", func(t *testing.T) {
		if _, err := inputprofile.ObservePackageOutput(inputprofile.BuildInputEvidence{}, result); err == nil {
			t.Fatal("unbound build output was accepted")
		}
	})
}
