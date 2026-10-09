package languagereference

import (
	"errors"
	"testing"
)

func TestOperatorResolutionAndComposition(t *testing.T) {
	where := OperatorContract{"std.where", "1", "where-v1", "pure", "rows", "rows"}
	expand := OperatorContract{"std.expand", "1", "expand-v1", "pure", "rows", "expanded"}
	summarize := OperatorContract{"std.summarize", "1", "summary-v1", "pure", "expanded", "summary"}
	catalog := []OperatorContract{summarize, expand, where}
	for _, pin := range []OperatorContract{where, expand, summarize} {
		got, err := ResolveOperator(pin, catalog)
		if err != nil || got != pin {
			t.Fatalf("resolution %v: %v", pin, err)
		}
	}
	out, err := ComposePure([]OperatorContract{where, expand, summarize})
	if err != nil || out != "summary" {
		t.Fatalf("composition: %q %v", out, err)
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
	if _, err := ComposePure([]OperatorContract{pure, {"std.invoke", "1", "v1", "write", "rows", "rows"}}); !errors.Is(err, ErrEffectEscalation) {
		t.Fatalf("effect escalation: %v", err)
	}
	if _, err := ComposePure([]OperatorContract{pure, {"std.summary", "1", "v1", "pure", "other", "summary"}}); !errors.Is(err, ErrOperatorInvalid) {
		t.Fatalf("type mismatch: %v", err)
	}
}
