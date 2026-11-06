package command

import (
	"context"
	"testing"

	"tei/internal/semantic"
)

// TestReferenceDataExecutorReportsAppliedOperation checks that a reference
// execution names the record it applied, so a reader can compare the declared
// effect with what happened without reading the store.
func TestReferenceDataExecutorReportsAppliedOperation(t *testing.T) {
	store := NewMemoryStore()
	seedDeleteStore(t, store)
	executor := ReferenceDataExecutor{World: deletePolicyWorld("cascade"), Store: store}
	result, err := executor.Execute(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Applied) != 2 {
		t.Fatalf("applied=%+v", result.Applied)
	}
	declared := semantic.DataEffect{Kind: semantic.DataEffectDelete, Data: "sales.order"}
	found := false
	for _, operation := range result.Applied {
		if operation.Kind != semantic.DataEffectDelete {
			t.Fatalf("operation=%+v", operation)
		}
		if operation.Key == "" {
			t.Fatalf("operation key is empty: %+v", operation)
		}
		if operation.MatchesDeclared(&declared) {
			found = true
		}
	}
	if !found {
		t.Fatalf("declared target is missing from applied=%+v", result.Applied)
	}
}

// TestReferenceDataExecutorCascadeNamesEachRecord checks that a cascade reports
// the related record it removed, not only the declared target.
func TestReferenceDataExecutorCascadeNamesEachRecord(t *testing.T) {
	store := NewMemoryStore()
	seedDeleteStore(t, store)
	executor := ReferenceDataExecutor{World: deletePolicyWorld("cascade"), Store: store}
	result, err := executor.Execute(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	var line bool
	for _, operation := range result.Applied {
		if operation.Data == "sales.line" {
			line = true
		}
	}
	if !line {
		t.Fatalf("applied=%+v", result.Applied)
	}
}

// TestInvokerRejectsAppliedOperationOutsideDeclaredKind checks that an executor
// reporting an operation of another kind is rejected before Commit.
func TestInvokerRejectsAppliedOperationOutsideDeclaredKind(t *testing.T) {
	unitOfWork := &MemoryUnitOfWork{}
	invoker := Invoker{
		World: commandWorld(),
		Executor: fakeExecutor{result: Result{
			Effect:  &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
			Applied: []AppliedOperation{{Kind: semantic.DataEffectDelete, Data: "sales.order"}},
		}},
		UnitOfWork: unitOfWork,
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "UNDECLARED_COMMAND_EFFECT" {
		t.Fatalf("code=%s", code)
	}
	if commits, _ := unitOfWork.Counts(); commits != 0 {
		t.Fatalf("commits=%d", commits)
	}
}

// TestInvokerReportsMissingAppliedOperation checks that a declared effect with
// no matching applied operation is a difference, not a silent success.
func TestInvokerReportsMissingAppliedOperation(t *testing.T) {
	invoker := Invoker{
		World:      commandWorld(),
		Executor:   fakeExecutor{result: Result{Applied: []AppliedOperation{{Kind: semantic.DataEffectUpdate, Data: "sales.order_line"}}}},
		UnitOfWork: &MemoryUnitOfWork{},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "MISSING_COMMAND_EFFECT" {
		t.Fatalf("code=%s", code)
	}
}

// TestInvokerRecordsAppliedOperations checks that the recorder keeps the
// applied operations beside the declared and reported effects.
func TestInvokerRecordsAppliedOperations(t *testing.T) {
	recorder := &MemoryRecorder{}
	invoker := Invoker{
		World:      commandWorld(),
		Executor:   ReferenceDataExecutor{World: commandWorld(), Store: seededStore(t)},
		UnitOfWork: &MemoryUnitOfWork{},
		Recorder:   recorder,
	}
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if len(recorder.Records) != 1 || len(recorder.Records[0].Applied) != 1 {
		t.Fatalf("records=%+v", recorder.Records)
	}
}

func seededStore(t *testing.T) *MemoryStore {
	t.Helper()
	store := NewMemoryStore()
	if err := store.Apply(&semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}, map[string]string{"order": "ORD-1"}); err != nil {
		t.Fatal(err)
	}
	return store
}
