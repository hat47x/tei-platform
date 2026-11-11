package saveunit

import (
	"context"
	"errors"
	"testing"

	"tei/internal/core"
	"tei/internal/failure"
)

// The save boundary reports three different situations: the unit itself is
// refused, the store refuses one change, and the storage boundary fails. They
// must not be reported the same way, because only the last one may be retried
// with a new identity after a lookup.
func TestSaveErrorFailureClassification(t *testing.T) {
	cases := []struct {
		name        string
		err         *SaveError
		wantKind    failure.Kind
		wantRetry   failure.RetryPolicy
		wantUnknown bool
	}{
		{
			name:     "empty unit is a business rejection",
			err:      &SaveError{Code: "SAVE_UNIT_EMPTY", Message: "save unit has no changes"},
			wantKind: failure.BusinessRejection, wantRetry: failure.NotRetryable,
		},
		{
			name:     "dependency cycle is a business rejection",
			err:      &SaveError{Code: "SAVE_CYCLE", Message: "cycle"},
			wantKind: failure.BusinessRejection, wantRetry: failure.NotRetryable,
		},
		{
			name:     "apply wrapping a store refusal keeps the refusal",
			err:      &SaveError{Code: "SAVE_APPLY_FAILED", Message: "apply failed", Cause: &ReferenceFailure{Code: ReferenceRecordExists, Message: "record exists"}},
			wantKind: failure.BusinessRejection, wantRetry: failure.NotRetryable,
		},
		{
			name:     "apply wrapping a host fault is technical",
			err:      &SaveError{Code: "SAVE_APPLY_FAILED", Message: "apply failed", Cause: errors.New("disk error")},
			wantKind: failure.TechnicalFailure, wantRetry: failure.NotRetryable,
		},
		{
			name:     "transient provider failure may be resent",
			err:      &SaveError{Code: "SAVE_PROVIDER_TRANSIENT", Message: "transient"},
			wantKind: failure.TechnicalFailure, wantRetry: failure.Retryable,
		},
		{
			// The effect may have been applied, so a new identity would
			// duplicate it: the caller must look the result up first.
			name:        "commit failure requires a lookup before any retry",
			err:         &SaveError{Code: "SAVE_TRANSACTION_COMMIT_FAILED", Message: "commit failed", EffectUnknown: true},
			wantKind:    failure.TechnicalFailure,
			wantRetry:   failure.LookupFirst,
			wantUnknown: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unified, ok := failure.AsFailure(tc.err)
			if !ok {
				t.Fatal("AsFailure did not recognize SaveError")
			}
			if unified.Kind() != tc.wantKind {
				t.Errorf("Kind() = %v, want %v", unified.Kind(), tc.wantKind)
			}
			if unified.Retryable() != tc.wantRetry {
				t.Errorf("Retryable() = %v, want %v", unified.Retryable(), tc.wantRetry)
			}
			if unified.HasEffectUnknown() != tc.wantUnknown {
				t.Errorf("HasEffectUnknown() = %v, want %v", unified.HasEffectUnknown(), tc.wantUnknown)
			}
			if unified.FailureCode() != tc.err.Code {
				t.Errorf("FailureCode() = %q, want %q", unified.FailureCode(), tc.err.Code)
			}
			if unified.FailureMessage() != tc.err.Message {
				t.Errorf("FailureMessage() = %q, want %q", unified.FailureMessage(), tc.err.Message)
			}
		})
	}
}

func TestReferenceStoreRefusalIsBusinessRejection(t *testing.T) {
	for _, code := range []string{ReferenceRecordExists, ReferenceRecordMissing} {
		unified, ok := failure.AsFailure(&ReferenceFailure{Code: code, Message: "refused"})
		if !ok {
			t.Fatalf("%s: AsFailure did not recognize ReferenceFailure", code)
		}
		if unified.Kind() != failure.BusinessRejection {
			t.Errorf("%s: Kind() = %v, want %v", code, unified.Kind(), failure.BusinessRejection)
		}
		if unified.Retryable() != failure.NotRetryable {
			t.Errorf("%s: Retryable() = %v, want %v", code, unified.Retryable(), failure.NotRetryable)
		}
		if unified.HasEffectUnknown() {
			t.Errorf("%s: a refused change was reported as having an unknown effect", code)
		}
	}
}

func TestHookFailureIsBusinessRejection(t *testing.T) {
	unified, ok := failure.AsFailure(&HookFailure{Code: "SAVE_HOOK_DENIED", Message: "denied by rule"})
	if !ok {
		t.Fatal("AsFailure did not recognize HookFailure")
	}
	if unified.Kind() != failure.BusinessRejection {
		t.Fatalf("Kind() = %v, want %v", unified.Kind(), failure.BusinessRejection)
	}
	if unified.Retryable() != failure.NotRetryable {
		t.Fatalf("Retryable() = %v, want %v", unified.Retryable(), failure.NotRetryable)
	}
	if unified.HasEffectUnknown() {
		t.Fatal("a denied save was reported as having an unknown effect")
	}
}

// TestSaveAppliesRecordExistsRefusal checks the whole path: the memory adapter
// refuses a second create of the same key, and the refusal reaches the caller
// classified as a business rejection with no effect, not as a host fault.
func TestSaveAppliesRecordExistsRefusal(t *testing.T) {
	store := &testStore{data: map[core.CanonicalID]map[string]map[string]string{}}
	adapter := &MemoryStoreAdapter{Store: store}
	unit := &SaveUnit{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		World:       orderLineWorld(),
		Changes: []EntityChange{
			{Data: "sales.order", Kind: ChangeKindCreate, Key: "order=ORD-1", Fields: PatchSet{"customer": {Value: strp("C1")}}},
		},
	}
	saver := NewSaver(WithStore(adapter), WithUnitOfWork(InMemoryUnitOfWork(adapter)), WithWorld(orderLineWorld()))
	if _, err := saver.Save(context.Background(), unit); err != nil {
		t.Fatalf("first save: %v", err)
	}
	_, err := saver.Save(context.Background(), unit)
	if err == nil {
		t.Fatal("a second create of the same key was accepted")
	}
	if !failure.IsBusinessRejection(err) {
		t.Fatalf("kind = %v, want business rejection (%v)", failureKindOf(err), err)
	}
	if failure.EffectMayBeUnknown(err) {
		t.Fatal("a refused create was reported as having an unknown effect")
	}
}

func failureKindOf(err error) failure.Kind {
	if f, ok := failure.AsFailure(err); ok {
		return f.Kind()
	}
	return ""
}
