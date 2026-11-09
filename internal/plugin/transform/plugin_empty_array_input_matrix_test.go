package transform_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

// This file pins the input-acceptance contract of the shipped SB-1 package
// Source (plugin-package-v1.transform.json, deterministic-transform-v1) for
// every omitempty array of the Plugin definition (#1739).
//
// The Go builder canonicalizes nil and empty slices to the same absent field.
// The shipped Source hashes the exact input JSON shape (member presence and
// member order) to check the BuildPlan fingerprint, so a raw TEI caller that
// writes an explicit empty array, or reorders members, supplies a different
// noncanonical input. The tests record the accept/reject matrix as it is; they
// do not normalize anything. A future Profile- or Engine-version-bound
// normalization must change this matrix deliberately.

type jsonPair struct {
	key   string
	value json.RawMessage
}

func decodePairs(t *testing.T, raw []byte) []jsonPair {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		t.Fatalf("not a JSON object: %v", err)
	}
	var pairs []jsonPair
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			t.Fatal(err)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			t.Fatal(err)
		}
		pairs = append(pairs, jsonPair{key: keyToken.(string), value: value})
	}
	return pairs
}

func encodePairs(pairs []jsonPair) []byte {
	var out bytes.Buffer
	out.WriteByte('{')
	for i, pair := range pairs {
		if i > 0 {
			out.WriteByte(',')
		}
		key, _ := json.Marshal(pair.key)
		out.Write(key)
		out.WriteByte(':')
		out.Write(pair.value)
	}
	out.WriteByte('}')
	return out.Bytes()
}

// editObject applies edit to the object found by walking path (a string selects
// an object member, an int selects an array element) and preserves every other
// byte and the member order of the surrounding objects.
func editObject(t *testing.T, raw []byte, path []any, edit func([]jsonPair) []jsonPair) []byte {
	t.Helper()
	if len(path) == 0 {
		return encodePairs(edit(decodePairs(t, raw)))
	}
	switch step := path[0].(type) {
	case string:
		pairs := decodePairs(t, raw)
		for i := range pairs {
			if pairs[i].key == step {
				pairs[i].value = editObject(t, pairs[i].value, path[1:], edit)
				return encodePairs(pairs)
			}
		}
	case int:
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			t.Fatal(err)
		}
		items[step] = editObject(t, items[step], path[1:], edit)
		out, err := json.Marshal(items)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	t.Fatalf("path %v not found", path)
	return nil
}

// insertEmptyArray adds key with an explicit empty array at the position the Go
// struct field order gives it, so that the member order stays the one Go emits.
func insertEmptyArray(key string, order []string) func([]jsonPair) []jsonPair {
	rank := func(k string) int {
		for i, name := range order {
			if name == k {
				return i
			}
		}
		return len(order)
	}
	return func(pairs []jsonPair) []jsonPair {
		at := len(pairs)
		for i, pair := range pairs {
			if rank(pair.key) > rank(key) {
				at = i
				break
			}
		}
		out := append([]jsonPair(nil), pairs[:at]...)
		out = append(out, jsonPair{key: key, value: json.RawMessage(`[]`)})
		return append(out, pairs[at:]...)
	}
}

var (
	definitionMemberOrder = []string{"id", "version", "runtime", "artifact", "provides", "requires", "externalRequirements"}
	runtimeMemberOrder    = []string{"kind", "command", "args", "sandbox"}
	capabilityMemberOrder = []string{"kind", "id", "actions", "observations", "evidence", "databaseProviders", "migrationTools", "features", "permissions"}
)

type emptyArrayInputCase struct {
	name string
	// strip removes the (otherwise populated) field from the Go definition so
	// that the explicit empty array is the only representation difference.
	strip func(*plugin.Definition)
	path  []any
	key   string
	order []string
}

func emptyArrayInputMatrix() []emptyArrayInputCase {
	return []emptyArrayInputCase{
		{name: "provides", strip: func(d *plugin.Definition) { d.Provides = nil }, key: "provides", order: definitionMemberOrder},
		{name: "requires", strip: func(d *plugin.Definition) { d.Requires = nil }, key: "requires", order: definitionMemberOrder},
		{name: "externalRequirements", strip: func(d *plugin.Definition) { d.External = nil }, key: "externalRequirements", order: definitionMemberOrder},
		{name: "runtime.args", strip: func(d *plugin.Definition) { d.Runtime.Args = nil }, path: []any{"runtime"}, key: "args", order: runtimeMemberOrder},
		{name: "capability.actions", strip: func(d *plugin.Definition) { d.Provides[0].Actions = nil }, path: []any{"provides", 0}, key: "actions", order: capabilityMemberOrder},
		{name: "capability.observations", strip: func(d *plugin.Definition) { d.Provides[1].Observations = nil }, path: []any{"provides", 1}, key: "observations", order: capabilityMemberOrder},
		{name: "capability.evidence", strip: func(d *plugin.Definition) { d.Provides[1].Evidence = nil }, path: []any{"provides", 1}, key: "evidence", order: capabilityMemberOrder},
		{name: "capability.databaseProviders", strip: func(d *plugin.Definition) { d.Provides[1].DatabaseProviders = nil }, path: []any{"provides", 1}, key: "databaseProviders", order: capabilityMemberOrder},
		{name: "capability.migrationTools", strip: func(d *plugin.Definition) { d.Provides[1].MigrationTools = nil }, path: []any{"provides", 1}, key: "migrationTools", order: capabilityMemberOrder},
		{name: "capability.features", strip: func(d *plugin.Definition) { d.Provides[1].Features = nil }, path: []any{"provides", 1}, key: "features", order: capabilityMemberOrder},
		{name: "capability.permissions", strip: func(d *plugin.Definition) { d.Provides[0].Permissions = nil }, path: []any{"provides", 0}, key: "permissions", order: capabilityMemberOrder},
	}
}

func evaluatePackageFromRawDefinition(t *testing.T, raw []byte, plan plugin.BuildPlan) ([]byte, error) {
	t.Helper()
	value, err := transform.ParseJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, evalErr := transform.Evaluate(referencePluginPackageTransformSource(t), map[string]transform.Value{
		"buildPlan":        valueFromGoJSON(t, plan),
		"pluginDefinition": value,
	})
	if evalErr != nil {
		return nil, evalErr
	}
	if got.Output.Kind != transform.ValueBytes {
		t.Fatalf("TEI materializer returned kind %s", got.Output.Kind)
	}
	return got.Output.Bytes, nil
}

// TestPluginDefinitionExplicitEmptyArrayInputMatrix records, for each omitempty
// array, whether an explicitly written empty array (a shape Go cannot emit) is
// accepted with bytes identical to omission or rejected by the shipped Source.
func TestPluginDefinitionExplicitEmptyArrayInputMatrix(t *testing.T) {
	for _, tc := range emptyArrayInputMatrix() {
		t.Run(tc.name, func(t *testing.T) {
			definition, plan := transformPackageFixture()
			tc.strip(&definition)
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(definition)
			goBytes, err := plugin.MaterializePluginPackage(definition, plan)
			if err != nil {
				t.Fatalf("Go builder rejected the omitted representation: %v", err)
			}
			if bytes.Contains(goBytes, []byte(`[]`)) {
				t.Fatal("Go package must never carry an explicit empty array")
			}
			omitted, err := json.Marshal(definition)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := evaluatePackageFromRawDefinition(t, omitted, plan); err != nil || !bytes.Equal(got, goBytes) {
				t.Fatalf("omitted representation must match Go bytes: %v", err)
			}

			explicit := editObject(t, omitted, tc.path, insertEmptyArray(tc.key, tc.order))
			if !bytes.Contains(explicit, []byte(`"`+tc.key+`":[]`)) {
				t.Fatal("explicit empty array was not injected")
			}
			if _, teiErr := evaluatePackageFromRawDefinition(t, explicit, plan); teiErr == nil || !strings.Contains(teiErr.Error(), "assert-equal failed") {
				t.Fatalf("explicit empty %s must be rejected as a fingerprint mismatch until a versioned normalization exists: %v", tc.name, teiErr)
			}
		})
	}
}

// TestUnknownOmitEmptyOperationIsRejectedByPinnedEngine records why a generic
// omit-empty-array-field operation cannot be added to the shipped Source: the
// pinned deterministic-transform-v1 engine rejects every operation it does not
// know, so such a Source would not run on the historical Runtime Engine.
func TestUnknownOmitEmptyOperationIsRejectedByPinnedEngine(t *testing.T) {
	source := referencePluginPackageTransformSource(t)
	source.Program.Output = transform.Expr{Op: transform.Op("omit-empty-array-field")}
	definition, plan := transformPackageFixture()
	_, err := transform.Evaluate(source, map[string]transform.Value{
		"buildPlan":        valueFromGoJSON(t, plan),
		"pluginDefinition": valueFromGoJSON(t, definition),
	})
	if err == nil || !strings.Contains(err.Error(), `unsupported transform operation "omit-empty-array-field"`) {
		t.Fatalf("unknown operation must be rejected: %v", err)
	}
}

// TestPluginDefinitionInputMemberOrderIsBoundByShippedSource separates the two
// causes that can make a raw input fingerprint-mismatch. Moving a member of an
// otherwise canonical definition is rejected exactly like an explicit empty
// array, so a caller must emit Go's member order as well as omit empty arrays.
func TestPluginDefinitionInputMemberOrderIsBoundByShippedSource(t *testing.T) {
	definition, plan := transformPackageFixture()
	raw, err := json.Marshal(definition)
	if err != nil {
		t.Fatal(err)
	}
	reordered := editObject(t, raw, nil, func(pairs []jsonPair) []jsonPair {
		return append([]jsonPair{pairs[len(pairs)-1]}, pairs[:len(pairs)-1]...)
	})
	if bytes.Equal(raw, reordered) {
		t.Fatal("reordering did not change the input")
	}
	if _, err := evaluatePackageFromRawDefinition(t, raw, plan); err != nil {
		t.Fatalf("canonical member order rejected: %v", err)
	}
	if _, err := evaluatePackageFromRawDefinition(t, reordered, plan); err == nil || !strings.Contains(err.Error(), "assert-equal failed") {
		t.Fatalf("reordered members must fail the fingerprint check: %v", err)
	}
}
