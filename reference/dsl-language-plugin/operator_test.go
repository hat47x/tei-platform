package languagereference

import (
	"errors"
	"testing"
)

func TestOperatorResolutionAndPinnedComposition(t *testing.T) {
	where := OperatorContract{ID: "std.where", Version: "1", Fingerprint: "where-v1", Effect: "pure", InputType: "rows", OutputType: "rows"}
	expand := OperatorContract{ID: "std.expand", Version: "1", Fingerprint: "expand-v1", Effect: "pure", InputType: "rows", OutputType: "expanded"}
	summarize := OperatorContract{ID: "std.summarize", Version: "1", Fingerprint: "summary-v1", Effect: "pure", InputType: "expanded", OutputType: "summary"}
	calls := []PinnedCall{{Contract: where, InputType: "rows"}, {Contract: expand, InputType: "rows"}, {Contract: summarize, InputType: "expanded"}}
	for _, catalog := range [][]OperatorContract{{summarize, expand, where}, {where, summarize, expand}} {
		for _, pin := range []OperatorContract{where, expand, summarize} {
			got, err := ResolveOperator(pin, catalog)
			if err != nil || got != pin {
				t.Fatalf("resolution %v: %v", pin, err)
			}
		}
		out, err := ComposePinned("rows", calls, catalog)
		if err != nil || out != "summary" {
			t.Fatalf("composition: %q %v", out, err)
		}
	}
}

func TestOperatorFailures(t *testing.T) {
	pure := OperatorContract{"std.where", "1", "v1", "pure", "rows", "rows"}
	cases := []struct {
		name    string
		pin     OperatorContract
		catalog []OperatorContract
		want    error
	}{
		{"missing", pure, nil, ErrOperatorMissing},
		{"duplicate", pure, []OperatorContract{pure, pure}, ErrOperatorAmbiguous},
		{"same ID and version, different fingerprints", pure, []OperatorContract{pure, {"std.where", "1", "v2", "pure", "rows", "rows"}}, ErrOperatorAmbiguous},
		{"invalid", OperatorContract{}, nil, ErrOperatorInvalid},
		{"effectful contract", OperatorContract{"std.invoke", "1", "v1", "invoke", "rows", "rows"}, []OperatorContract{{"std.invoke", "1", "v1", "invoke", "rows", "rows"}}, ErrEffectEscalation},
		{"signature drift", pure, []OperatorContract{{"std.where", "1", "v1", "pure", "different", "rows"}}, ErrOperatorInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveOperator(tc.pin, tc.catalog)
			if !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

func TestPinnedCompositionCannotBypassCatalog(t *testing.T) {
	pure := OperatorContract{"std.where", "1", "v1", "pure", "rows", "rows"}
	effectful := OperatorContract{"std.invoke", "1", "v1", "invoke", "rows", "rows"}
	badType := OperatorContract{"std.summary", "1", "v1", "pure", "other", "summary"}
	cases := []struct {
		name    string
		calls   []PinnedCall
		catalog []OperatorContract
		want    error
	}{
		{"missing accepted catalog", []PinnedCall{{Contract: pure, InputType: "rows"}}, nil, ErrOperatorMissing},
		{"effect escalation", []PinnedCall{{Contract: effectful, InputType: "rows"}}, []OperatorContract{effectful}, ErrEffectEscalation},
		{"type discontinuity", []PinnedCall{{Contract: pure, InputType: "rows"}, {Contract: badType, InputType: "other"}}, []OperatorContract{pure, badType}, ErrOperatorInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ComposePinned("rows", tc.calls, tc.catalog)
			if !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}
