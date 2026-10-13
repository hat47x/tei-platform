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

func v2RawDefinition(t *testing.T, def plugin.Definition, edit func(map[string]any)) []byte {
	t.Helper()
	raw, err := json.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	if edit != nil {
		edit(object)
	}
	raw, err = json.Marshal(object) // Intentionally uses map order, not Go struct member order.
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestV2InputProfileNormalizesExplicitEmptyArraysToGoTEIPackageBytes(t *testing.T) {
	source := referencePluginPackageTransformSource(t)
	before, err := transform.SourceFingerprint(source)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name  string
		strip func(*plugin.Definition)
		edit  func(map[string]any)
	}{
		{name: "missing arrays and reordered members"},
		{name: "provides explicitly empty", strip: func(d *plugin.Definition) { d.Provides = nil }, edit: func(v map[string]any) { v["provides"] = []any{} }},
		{name: "requires explicitly empty", strip: func(d *plugin.Definition) { d.Requires = nil }, edit: func(v map[string]any) { v["requires"] = []any{} }},
		{name: "external requirements explicitly empty", strip: func(d *plugin.Definition) { d.External = nil }, edit: func(v map[string]any) { v["externalRequirements"] = []any{} }},
		{name: "runtime args explicitly empty", strip: func(d *plugin.Definition) { d.Runtime.Args = nil }, edit: func(v map[string]any) {
			v["runtime"].(map[string]any)["args"] = []any{}
		}},
		{name: "capability actions explicitly empty", strip: func(d *plugin.Definition) { d.Provides[1].Actions = nil }, edit: func(v map[string]any) {
			v["provides"].([]any)[1].(map[string]any)["actions"] = []any{}
		}},
		{name: "capability permissions explicitly empty", strip: func(d *plugin.Definition) { d.Provides[1].Permissions = nil }, edit: func(v map[string]any) {
			v["provides"].([]any)[1].(map[string]any)["permissions"] = []any{}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			def, plan := transformPackageFixture()
			if tc.strip != nil {
				tc.strip(&def)
			}
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(def)
			expected, err := plugin.MaterializePluginPackage(def, plan)
			if err != nil {
				t.Fatal(err)
			}
			raw := v2RawDefinition(t, def, tc.edit)
			normalized, evidence, err := inputprofile.CanonicalizePluginDefinition(inputprofile.PluginDefinitionJSONV2, source, raw)
			if err != nil {
				t.Fatalf("profile-v2 rejected valid normalization input: %v", err)
			}
			if evidence.Profile != inputprofile.PluginDefinitionJSONV2 || evidence.SourceFingerprint != before {
				t.Fatalf("profile v2 did not bind Source identity: %#v", evidence)
			}
			if evidence.OriginalInputDigest == "" || evidence.CanonicalInputDigest == "" {
				t.Fatal("normalization evidence omitted input digests")
			}
			actual, err := transform.Evaluate(source, map[string]transform.Value{
				"pluginDefinition": normalized,
				"buildPlan":        valueFromGoJSON(t, plan),
			})
			if err != nil {
				t.Fatalf("legacy engine-compatible Source rejected normalized input: %v", err)
			}
			if actual.Output.Kind != transform.ValueBytes || !bytes.Equal(actual.Output.Bytes, expected) {
				t.Fatal("opt-in profile v2 did not reproduce existing Go SB-1 package bytes")
			}
		})
	}
	after, err := transform.SourceFingerprint(source)
	if err != nil || before != after {
		t.Fatalf("input admission changed shipped Source identity: %v", err)
	}
}

func TestV2InputProfileRejectsUnqualifiedOrAmbiguousRequests(t *testing.T) {
	source := referencePluginPackageTransformSource(t)
	def, _ := transformPackageFixture()
	valid, err := json.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	bad := []struct {
		name    string
		profile string
		source  transform.Source
		raw     []byte
		want    string
	}{
		{name: "no opt in", source: source, raw: valid, want: "unsupported plugin definition input profile"},
		{name: "unknown profile", profile: "tei.plugin-definition-json/v3", source: source, raw: valid, want: "unsupported plugin definition input profile"},
		{name: "duplicate JSON key", profile: inputprofile.PluginDefinitionJSONV2, source: source, raw: []byte(`{"id":"tei.x","id":"tei.y"}`), want: "declared more than once"},
		{name: "unknown schema field", profile: inputprofile.PluginDefinitionJSONV2, source: source, raw: []byte(`{"unknown":true}`), want: "unknown field"},
		{name: "wrong property type", profile: inputprofile.PluginDefinitionJSONV2, source: source, raw: []byte(`{"provides":99}`), want: "unmarshal"},
		{name: "trailing input", profile: inputprofile.PluginDefinitionJSONV2, source: source, raw: []byte(`{}{}`), want: "multiple values"},
	}
	other := source
	other.Identity.Version = "2.0.0"
	bad = append(bad, struct {
		name    string
		profile string
		source  transform.Source
		raw     []byte
		want    string
	}{name: "unqualified Source identity", profile: inputprofile.PluginDefinitionJSONV2, source: other, raw: valid, want: "requires exact SB-1 package Source identity"})
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := inputprofile.CanonicalizePluginDefinition(tc.profile, tc.source, tc.raw)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("profile-v2 admitted invalid input: %v (want %q)", err, tc.want)
			}
		})
	}
}
