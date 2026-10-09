package crossprojectextraction_test

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin/transform"
)

// E6: when the accepted semantic input already carries ordered typed rule
// descriptors, the existing map-array operation can materialize the plan.
// This fixture does not decide which rules are accepted.
func projectionSource() transform.Source {
	rule := transform.Expr{Op: transform.OpInput, Name: "rule"}
	ruleField := func(name string) transform.Expr {
		from := rule
		return transform.Expr{Op: transform.OpSelectField, From: &from, Field: name}
	}
	arrayInput := transform.Expr{Op: transform.OpInput, Name: "rules"}
	body := transform.Expr{
		Op: transform.OpConstructObject,
		Fields: []transform.ObjectFieldExpr{
			{Name: "op", Value: ruleField("op")},
			{Name: "field", Value: ruleField("field")},
		},
	}
	mappedRules := transform.Expr{Op: transform.OpMapArray, From: &arrayInput, Item: "rule", Body: &body}
	return transform.Source{
		FormatVersion: transform.SourceFormatVersion,
		Identity: transform.SourceIdentity{
			ID: core.MustCanonicalID("tei.reference.required-fields-projection"),
			Version: "1.0.0",
			ContractVersion: "tei.reference.validation-plan/v0",
		},
		Profile: transform.ProfileV1,
		Limits: transform.Limits{
			MaxSteps: 200, MaxCollection: 8, MaxInputNodes: 100,
			MaxInputBytes: 4096, MaxOutputBytes: 4096, MaxNestingDepth: 16,
		},
		Inputs: []transform.InputSpec{{Name: "rules", Kind: transform.InputArray}},
		Program: transform.Program{Output: transform.Expr{
			Op: transform.OpConstructObject,
			Fields: []transform.ObjectFieldExpr{
				{Name: "format", Value: transform.Expr{Op: transform.OpLiteral,
					Literal: json.RawMessage(`"tei.reference.validation-plan/v0"`)}},
				{Name: "rules", Value: mappedRules},
			},
		}},
	}
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
	source := projectionSource()
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

func TestProjectionFailsClosedForMissingRuleField(t *testing.T) {
	source := projectionSource()
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
	source := projectionSource()
	unaccepted := projectionOutput(t, source, `[{"op":"unknown-business-rule","field":"orderId"}]`)
	if !strings.Contains(unaccepted, "unknown-business-rule") {
		t.Fatal("fixture must show that the transform alone cannot authorize semantic rules")
	}
}
