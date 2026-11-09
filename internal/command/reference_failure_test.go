package command

import (
	"context"
	"errors"
	"testing"

	"tei/internal/semantic"
)

// TestReferenceRefusalsCarryCodes checks that each refusal a caller can expect
// from the reference store carries a code, so reporting an outcome does not rely
// on message text.
func TestReferenceRefusalsCarryCodes(t *testing.T) {
	store := NewMemoryStore()
	if err := store.Apply(&semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}, map[string]string{"order": "ORD-1"}); err != nil {
		t.Fatal(err)
	}
	err := store.Apply(&semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}, map[string]string{"order": "ORD-1"})
	var exists *ReferenceFailure
	if !errors.As(err, &exists) || exists.Code != ReferenceRecordExists {
		t.Fatalf("duplicate create err=%v", err)
	}

	err = store.Apply(&semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"}, map[string]string{"order": "ORD-2"})
	var missing *ReferenceFailure
	if !errors.As(err, &missing) || missing.Code != ReferenceRecordMissing {
		t.Fatalf("missing update err=%v", err)
	}

	executor := ReferenceDataExecutor{World: deletePolicyWorld("restrict"), Store: NewMemoryStore()}
	if err := executor.Store.Apply(&semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.line"}, map[string]string{"id": "L1"}); err != nil {
		t.Fatal(err)
	}
	if err := executor.Store.Apply(&semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}, map[string]string{"order": "ORD-1"}); err != nil {
		t.Fatal(err)
	}
	_, err = executor.Execute(context.Background(), request())
	var rule *ReferenceFailure
	if !errors.As(err, &rule) || rule.Code != ReferenceRuleRejected {
		t.Fatalf("restricted delete err=%v", err)
	}
}

// TestFailureOutcomeNamesTheBusinessResult checks that a refusal maps to a
// business outcome name, and that an execution fault does not.
func TestFailureOutcomeNamesTheBusinessResult(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{&ReferenceFailure{Code: ReferenceRecordMissing}, "not-found"},
		{&ReferenceFailure{Code: ReferenceRecordExists}, "already-exists"},
		{&ReferenceFailure{Code: ReferenceValueRejected}, "value-rejected"},
		{&ReferenceFailure{Code: ReferenceRuleRejected}, "rule-rejected"},
	}
	for _, tc := range cases {
		outcome, ok := FailureOutcome(tc.err)
		if !ok || outcome != tc.want {
			t.Fatalf("err=%v outcome=%q ok=%v", tc.err, outcome, ok)
		}
	}
	if _, ok := FailureOutcome(errors.New("plain")); ok {
		t.Fatal("an execution fault was reported as a business outcome")
	}
}
