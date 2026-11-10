package main

import "testing"

// §4.36 makes the builder refuse a definition the plan does not name. That is the
// builder's own behaviour, so a builder that skips it would pass. These tests pin
// the entry-side comparison that closes the same gap without trusting the builder
// to enforce anything.

func TestBindDefinitionToPlan(t *testing.T) {
	receipt := func(digest string) *PlanReceipt {
		body := `{"plan":{"sources":[{"kind":"plugin-definition","id":"definition","locator":"plugin.tei","digest":"` + digest + `"}]}}`
		return &PlanReceipt{ReceiptJSON: body}
	}

	cases := []struct {
		name     string
		receipt  *PlanReceipt
		reported string
		want     string
	}{
		{
			name:     "the builder read the planned definition",
			receipt:  receipt("sha256:aaaa"),
			reported: "sha256:aaaa",
			want:     definitionBindingBound,
		},
		{
			name:     "the builder read a different definition",
			receipt:  receipt("sha256:aaaa"),
			reported: "sha256:bbbb",
			want:     definitionBindingFailed,
		},
		{
			name:     "the builder reported nothing",
			receipt:  receipt("sha256:aaaa"),
			reported: "",
			want:     definitionBindingNotRun,
		},
		{
			name:     "there is no plan",
			receipt:  nil,
			reported: "sha256:aaaa",
			want:     definitionBindingNotRun,
		},
		{
			name:     "the plan records no definition digest",
			receipt:  receipt(""),
			reported: "sha256:aaaa",
			want:     definitionBindingNotRun,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := bindDefinitionToPlan(tc.receipt, tc.reported)
			if record.State != tc.want {
				t.Fatalf("state = %q, want %q (%s%s)", record.State, tc.want, record.Reason, record.UnperformedReason)
			}
			if record.Boundary == "" {
				t.Fatal("the record has no boundary")
			}
			switch record.State {
			case definitionBindingNotRun:
				if record.UnperformedReason == "" {
					t.Fatal("an unperformed record must say why")
				}
			case definitionBindingFailed:
				if record.Reason == "" {
					t.Fatal("a mismatch must say what disagreed")
				}
			}
		})
	}
}

// A missing value is not agreement: the record must not read as bound when either
// side is absent.
func TestBindDefinitionToPlanDoesNotTreatSilenceAsAgreement(t *testing.T) {
	body := `{"plan":{"sources":[{"kind":"plugin-definition","id":"definition","locator":"plugin.tei","digest":""}]}}`
	record := bindDefinitionToPlan(&PlanReceipt{ReceiptJSON: body}, "")
	if record.State == definitionBindingBound || record.Matched {
		t.Fatalf("empty values were bound: %+v", record)
	}
}
