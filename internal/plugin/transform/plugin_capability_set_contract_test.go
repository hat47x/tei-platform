package transform_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

func materializeCapabilitySetPair(t *testing.T, definition plugin.Definition, explicitProvides string) ([]byte, []byte, error, error) {
	t.Helper()
	_, plan := transformPackageFixture()
	plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(definition)
	goBytes, goErr := plugin.MaterializePluginPackage(definition, plan)

	raw, err := json.Marshal(definition)
	if err != nil {
		t.Fatal(err)
	}
	if explicitProvides != "" {
		// json.Marshal omits nil and empty Provides. Test the same semantic
		// value when an external TEI caller supplies the field explicitly.
		raw = append([]byte(`{"provides":`+explicitProvides+","), raw[1:]...)
	}
	value, err := transform.ParseJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, teiErr := transform.Evaluate(referencePluginPackageTransformSource(t), map[string]transform.Value{
		"buildPlan":        valueFromGoJSON(t, plan),
		"pluginDefinition": value,
	})
	if teiErr != nil {
		return goBytes, nil, goErr, teiErr
	}
	if got.Output.Kind != transform.ValueBytes {
		t.Fatalf("TEI materializer returned kind %s", got.Output.Kind)
	}
	return goBytes, got.Output.Bytes, goErr, nil
}

func TestPluginCapabilitySetPermutationPreservesFingerprintAndBytes(t *testing.T) {
	base, _ := transformPackageFixture()
	reversed := base
	reversed.Provides = append([]plugin.Capability(nil), base.Provides...)
	reversed.Provides[0], reversed.Provides[1] = reversed.Provides[1], reversed.Provides[0]
	if plugin.BuildDefinitionFingerprint(base) != plugin.BuildDefinitionFingerprint(reversed) {
		t.Fatal("Capability permutation changed definition fingerprint")
	}
	aGo, aTEI, aErr, aTEIErr := materializeCapabilitySetPair(t, base, "")
	bGo, bTEI, bErr, bTEIErr := materializeCapabilitySetPair(t, reversed, "")
	if aErr != nil || aTEIErr != nil || bErr != nil || bTEIErr != nil {
		t.Fatalf("valid Capability order rejected: Go=%v TEI=%v reversed Go=%v TEI=%v", aErr, aTEIErr, bErr, bTEIErr)
	}
	if !bytes.Equal(aGo, bGo) || !bytes.Equal(aGo, aTEI) || !bytes.Equal(bGo, bTEI) {
		t.Fatal("permuting valid Capability order changed Go/TEI package bytes")
	}
}

func TestPluginCapabilitySameIDDifferentKindIsValid(t *testing.T) {
	definition, _ := transformPackageFixture()
	definition.Provides[1].ID = definition.Provides[0].ID
	gotGo, gotTEI, goErr, teiErr := materializeCapabilitySetPair(t, definition, "")
	if goErr != nil || teiErr != nil {
		t.Fatalf("different kinds must be allowed to share a Capability ID: Go=%v TEI=%v", goErr, teiErr)
	}
	if !bytes.Equal(gotGo, gotTEI) {
		t.Fatal("Go and TEI builders differ when Capability IDs are shared across kinds")
	}
}

func TestPluginCapabilityAbsentAndEmptyProvidesPreservePackageBytes(t *testing.T) {
	base, _ := transformPackageFixture()
	base.Provides = nil
	baselineFingerprint := plugin.BuildDefinitionFingerprint(base)
	baseGo, baseTEI, goErr, teiErr := materializeCapabilitySetPair(t, base, "")
	if goErr != nil || teiErr != nil || !bytes.Equal(baseGo, baseTEI) {
		t.Fatalf("absent Provides not equivalent: Go=%v TEI=%v", goErr, teiErr)
	}
	for _, tc := range []struct {
		name     string
		provides []plugin.Capability
		json     string
	}{
		{name: "nil omitted"},
		{name: "empty omitted", provides: []plugin.Capability{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition := base
			definition.Provides = tc.provides
			if plugin.BuildDefinitionFingerprint(definition) != baselineFingerprint {
				t.Fatal("nil/empty Provides changed definition fingerprint")
			}
			goBytes, teiBytes, goErr, teiErr := materializeCapabilitySetPair(t, definition, tc.json)
			if goErr != nil || teiErr != nil {
				t.Fatalf("nil/empty Provides rejected: Go=%v TEI=%v", goErr, teiErr)
			}
			if !bytes.Equal(baseGo, goBytes) || !bytes.Equal(goBytes, teiBytes) {
				t.Fatal("absent/empty Provides changed Go/TEI package bytes")
			}
		})
	}
	// A raw JSON caller can express the empty array explicitly, while Go's
	// omitempty cannot. Existing deterministic-transform-v1 Source binds its
	// fingerprint to the exact input shape; historical SB-3 engines do not
	// support conditional empty-array omission. Do not silently claim that
	// this noncanonical input is equivalent to the omitted representation.
	// This input also puts "provides" first, so it differs in member order as
	// well; plugin_empty_array_input_matrix_test.go isolates emptiness from
	// order for every omitempty array.
	explicitDefinition := base
	explicitDefinition.Provides = []plugin.Capability{}
	explicitGo, _, explicitGoErr, explicitTEIErr := materializeCapabilitySetPair(t, explicitDefinition, "[]")
	if explicitGoErr != nil || !bytes.Equal(explicitGo, baseGo) {
		t.Fatalf("Go empty Provides changed canonical bytes: %v", explicitGoErr)
	}
	if explicitTEIErr == nil || !strings.Contains(explicitTEIErr.Error(), "assert-equal failed") {
		t.Fatalf("TEI Source must fail closed on explicit empty Provides until compatible normalization is versioned: %v", explicitTEIErr)
	}
}

func TestPluginCapabilityDuplicatePairRejectedWithoutOrderDependence(t *testing.T) {
	definition, _ := transformPackageFixture()
	duplicate := definition.Provides[0]
	duplicate.Features = nil // Same (Kind, ID) with different legal details.
	definition.Provides = append(definition.Provides, duplicate)
	reversed := definition
	reversed.Provides = append([]plugin.Capability(nil), definition.Provides...)
	reversed.Provides[0], reversed.Provides[2] = reversed.Provides[2], reversed.Provides[0]

	var priorGoError string
	for _, tc := range []struct {
		name       string
		definition plugin.Definition
	}{
		{name: "original order", definition: definition},
		{name: "reversed order", definition: reversed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, goErr, teiErr := materializeCapabilitySetPair(t, tc.definition, "")
			if goErr == nil || !strings.Contains(goErr.Error(), "more than once") {
				t.Fatalf("Go accepted repeated Capability (Kind, ID): %v", goErr)
			}
			if teiErr == nil || !strings.Contains(teiErr.Error(), "duplicate key") {
				t.Fatalf("TEI accepted repeated Capability (Kind, ID): %v", teiErr)
			}
			if priorGoError != "" && priorGoError != goErr.Error() {
				t.Fatalf("Go diagnostic depends on Capability order: %q versus %q", priorGoError, goErr)
			}
			priorGoError = goErr.Error()
		})
	}
}
