package crossprojectextraction_test

import (
	"os"
	"strings"
	"testing"

	"tei/internal/plugin/transform"
)

// The serialized Source is the single Reference for this projection, not a
// separately generated program hidden in the test. It is not accepted meaning.
func projectionSource(t *testing.T) transform.Source {
	t.Helper()
	raw, err := os.ReadFile("required-fields-projection.transform.json")
	if err != nil {
		t.Fatal(err)
	}
	source, err := transform.ParseSourceJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func projectionOutput(t *testing.T, source transform.Source, rules string) string {
	t.Helper()
	input, err := transform.ParseJSON([]byte(rules))
	if err != nil {
		t.Fatal(err)
	}
	result, err := transform.Evaluate(source, map[string]transform.Value{"rules": input})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := result.Output.JSON()
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestExistingMapArrayProjectsTwoIndependentAcceptedRuleSets(t *testing.T) {
	source := projectionSource(t)
	if err := source.Validate(); err != nil {
		t.Fatalf("reference Source is invalid: %v", err)
	}

	for _, tt := range []struct {
		name, acceptedRuleDescriptors, expected string
	}{
		{
			name: "order",
			acceptedRuleDescriptors: `[{"op":"required-text","field":"orderId"},{"op":"nonnegative-safe-integer","field":"totalYen"}]`,
			expected: `{"format":"tei.reference.validation-plan/v0","rules":[{"op":"required-text","field":"orderId"},{"op":"nonnegative-safe-integer","field":"totalYen"}]}`,
		},
		{
			name: "invoice",
			acceptedRuleDescriptors: `[{"op":"required-text","field":"invoiceId"},{"op":"nonnegative-safe-integer","field":"amountYen"}]`,
			expected: `{"format":"tei.reference.validation-plan/v0","rules":[{"op":"required-text","field":"invoiceId"},{"op":"nonnegative-safe-integer","field":"amountYen"}]}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			first := projectionOutput(t, source, tt.acceptedRuleDescriptors)
			second := projectionOutput(t, source, tt.acceptedRuleDescriptors)
			if first != tt.expected || second != tt.expected {
				t.Fatalf("projection differs from independent golden: first=%s second=%s", first, second)
			}
		})
	}
}

func TestProjectionSourceRejectsUnknownTransformOperator(t *testing.T) {
	raw, err := os.ReadFile("required-fields-projection.transform.json")
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(raw), `"op": "map-array"`, `"op": "unrecognized-transform"`, 1)
	if changed == string(raw) {
		t.Fatal("test fixture did not change the transform program")
	}
	if _, err := transform.ParseSourceJSON([]byte(changed)); err == nil {
		t.Fatal("unsupported transform operation was not rejected")
	}
}

func TestProjectionFailsClosedForMissingRuleField(t *testing.T) {
	source := projectionSource(t)
	input, err := transform.ParseJSON([]byte(`[{"op":"required-text"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transform.Evaluate(source, map[string]transform.Value{"rules": input}); err == nil {
		t.Fatal("missing mandatory field was silently accepted")
	}
}

func TestProjectionDoesNotValidateDomainRuleSemantics(t *testing.T) {
	// A mapping operation can copy descriptors, not decide whether their
	// operation is part of the accepted business meaning. The authoring
	// layer must enforce the semantic contract before this transform runs.
	source := projectionSource(t)
	unaccepted := projectionOutput(t, source, `[{"op":"unknown-business-rule","field":"orderId"}]`)
	if !strings.Contains(unaccepted, "unknown-business-rule") {
		t.Fatal("fixture must show that the transform alone cannot authorize semantic rules")
	}
}
