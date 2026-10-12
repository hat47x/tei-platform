package customeradvice

import "testing"

func TestEvaluateSyntheticProjectRule(t *testing.T) {
	tests := []struct {
		name, code, note, kind string
	}{
		{"review", "VIP-123", "", "review_suggested"},
		{"review-trimmed", " VIP-123 ", "  ", "review_suggested"},
		{"filled", "VIP-123", "確認済み", "no_suggestion"},
		{"ordinary", "ACME-1", "", "no_suggestion"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Evaluate(Input{CustomerCode: tc.code, Note: tc.note})
			if err != nil || got.Kind != tc.kind || !got.Advisory ||
				got.Schema != "tei.reference.project-customer-advice/v0" {
				t.Fatalf("Evaluate() = %#v, %v", got, err)
			}
		})
	}
}

func TestEvaluateRejectsInvalidInputs(t *testing.T) {
	cases := []Input{
		{}, {CustomerCode: "BAD ID"}, {CustomerCode: "日本語"},
		{CustomerCode: "A", Note: string(make([]byte, 241))},
		{CustomerCode: "A", Note: string([]byte{0xff})},
	}
	for _, input := range cases {
		if _, err := Evaluate(input); err == nil {
			t.Errorf("accepted invalid input %#v", input)
		}
	}
}
