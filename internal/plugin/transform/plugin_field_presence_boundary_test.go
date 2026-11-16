package transform_test

import (
	"fmt"
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin/transform"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// The Value projection contains values, not the read state or semantic Attribute ID.
// This reader records the independently selected Attribute requested by the Rule.
type presenceBoundarySource struct {
	reading validationeval.NumberReading
	reads   []core.CanonicalID
}

func (s *presenceBoundarySource) ReadNumber(id core.CanonicalID) (validationeval.NumberReading, error) {
	s.reads = append(s.reads, id)
	return s.reading, nil
}

func (s *presenceBoundarySource) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, fmt.Errorf("unexpected relation read %s/%s", relation, attribute)
}

// PC-1b boundary: optional value projection deliberately collapses a missing
// field and an explicit JSON null. The typed reader must not derive the Rule's
// read state, Attribute identity, or a delivery grant from that projected JSON.
// This is a local negative witness, not a Plugin binding or Provider call.
func TestPluginOptionalProjectionCannotStandInForAStatefulTypedRead(t *testing.T) {
	recordExpr := transform.Expr{Op: transform.OpInput, Name: "record"}
	amountExpr := transform.Expr{
		Op: transform.OpSelectField, From: &recordExpr, Field: "amount", Optional: true,
	}
	source := transform.Source{
		FormatVersion: transform.SourceFormatVersion,
		Identity: transform.SourceIdentity{
			ID:              core.MustCanonicalID("tei.plugin.optional_presence_boundary_test"),
			Version:         "1.0.0",
			ContractVersion: "tei.plugin-transform/v1",
		},
		Profile: transform.ProfileV1,
		Limits: transform.Limits{
			MaxSteps:        200,
			MaxCollection:   16,
			MaxInputNodes:   128,
			MaxInputBytes:   4096,
			MaxOutputBytes:  4096,
			MaxNestingDepth: 16,
		},
		Inputs: []transform.InputSpec{{Name: "record", Kind: transform.InputObject}},
		Program: transform.Program{Output: transform.Expr{
			Op: transform.OpConstructObject,
			Fields: []transform.ObjectFieldExpr{{Name: "amount", Value: amountExpr}},
		}},
	}
	project := func(record transform.Value) transform.Value {
		t.Helper()
		got, err := transform.Evaluate(source, map[string]transform.Value{"record": record})
		if err != nil {
			t.Fatal(err)
		}
		return got.Output
	}
	missing, err := transform.Object(transform.Field{Name: "orderId", Value: transform.String("O-1")})
	if err != nil {
		t.Fatal(err)
	}
	explicitNull, err := transform.Object(
		transform.Field{Name: "orderId", Value: transform.String("O-1")},
		transform.Field{Name: "amount", Value: transform.Null()},
	)
	if err != nil {
		t.Fatal(err)
	}
	missingOutput := project(missing)
	nullOutput := project(explicitNull)
	if !transform.Equal(missingOutput, nullOutput) {
		t.Fatal("expected optional value projection to lose the missing/null distinction")
	}
	amount, ok := missingOutput.Field("amount")
	if !ok || amount.Kind != transform.ValueNull {
		t.Fatalf("expected projected null, got present=%t value=%+v", ok, amount)
	}

	rule := semantic.Validation{
		ID:        core.MustCanonicalID("sales.order_line.amount_positive"),
		Symbol:    "amount_positive",
		Kind:      semantic.ValidationPositive,
		Attribute: core.MustCanonicalID("sales.order_line.amount"),
	}
	for _, tc := range []struct {
		name    string
		reading validationeval.NumberReading
		reason  validationeval.UndecidedReason
	}{
		{"explicit null", validationeval.NumberReading{State: validationeval.ValueNull}, validationeval.UndecidedValueNull},
		{"not fetched", validationeval.NumberReading{State: validationeval.ValueNotFetched}, validationeval.UndecidedValueNotFetched},
		{"positive value", validationeval.NumberReading{State: validationeval.ValuePresent, Value: big.NewRat(2, 1)}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &presenceBoundarySource{reading: tc.reading}
			decision, err := validationeval.Decide(rule, reader, validationeval.Operators{})
			if err != nil {
				t.Fatal(err)
			}
			if len(reader.reads) != 1 || reader.reads[0] != rule.Attribute {
				t.Fatalf("Rule read the wrong semantic Attribute: %v", reader.reads)
			}
			if tc.reason == "" {
				if decision.Result != validationeval.ResultSatisfied || decision.Undecided != nil {
					t.Fatalf("expected satisfied, got %+v", decision)
				}
			} else if decision.Result != "" || decision.Undecided == nil ||
				decision.Undecided.Reason != tc.reason || decision.Undecided.Attribute != rule.Attribute {
				t.Fatalf("expected undecided(%s) for %s, got %+v", tc.reason, rule.Attribute, decision)
			}
		})
	}
}
