package transform_test

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin/transform"
)

// This covers pure output shaping only. Contract field identity, completeness,
// source provenance, Host admission, and external disclosure are separate gates.
func TestPluginFieldProjectionKeepsOnlyExplicitConsumerFields(t *testing.T) {
	in := transform.Expr{Op: transform.OpInput, Name: "record"}
	field := func(name string) transform.Expr {
		return transform.Expr{Op: transform.OpSelectField, From: &in, Field: name}
	}
	base := transform.Source{
		FormatVersion: transform.SourceFormatVersion,
		Identity: transform.SourceIdentity{
			ID:              core.MustCanonicalID("tei.plugin.field_projection_test"),
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
	}
	whole := base
	whole.Program.Output = in
	selected := base
	selected.Program.Output = transform.Expr{
		Op: transform.OpConstructObject,
		Fields: []transform.ObjectFieldExpr{
			{Name: "orderId", Value: field("orderId")},
			{Name: "amount", Value: field("amount")},
		},
	}

	record := func(amountPresent, notePresent bool, amount transform.Value) transform.Value {
		t.Helper()
		fields := []transform.Field{{Name: "orderId", Value: transform.String("O-001")}}
		if amountPresent {
			fields = append(fields, transform.Field{Name: "amount", Value: amount})
		}
		if notePresent {
			fields = append(fields, transform.Field{Name: "internalNote", Value: transform.String("private")})
		}
		v, err := transform.Object(fields...)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	invoke := func(source transform.Source, record transform.Value) (transform.Value, error) {
		result, err := transform.Evaluate(source, map[string]transform.Value{"record": record})
		return result.Output, err
	}

	input := record(true, true, transform.Number("1200"))
	all, err := invoke(whole, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, present := all.Field("internalNote"); !present {
		t.Fatal("whole-record transfer should preserve the additional field")
	}

	projection, err := invoke(selected, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Object) != 2 {
		t.Fatalf("expected exactly two consumer fields, got %+v", projection)
	}
	if _, present := projection.Field("internalNote"); present {
		t.Fatal("field projection leaked a non-selected input field")
	}
	withoutNote, err := invoke(selected, record(true, false, transform.Number("1200")))
	if err != nil {
		t.Fatal(err)
	}
	withNoteJSON, err := projection.JSON()
	if err != nil {
		t.Fatal(err)
	}
	withoutNoteJSON, err := withoutNote.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(withNoteJSON, withoutNoteJSON) {
		t.Fatalf("extra input field changed the consumer projection: %s != %s", withNoteJSON, withoutNoteJSON)
	}

	if _, err := invoke(selected, record(false, true, transform.Null())); err == nil || !strings.Contains(err.Error(), "required field \"amount\" is missing") {
		t.Fatalf("missing required field was not rejected: %v", err)
	}
	withNull, err := invoke(selected, record(true, true, transform.Null()))
	if err != nil {
		t.Fatalf("explicit null must remain distinct from missing: %v", err)
	}
	amount, present := withNull.Field("amount")
	if !present || amount.Kind != transform.ValueNull {
		t.Fatalf("explicit null was lost: present=%t, value=%+v", present, amount)
	}
}
