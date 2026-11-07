package command

import (
	"context"
	"errors"
	"testing"

	"tei/internal/semantic"
)

type failingRecorder struct{}

func (failingRecorder) Record(context.Context, OperationRecord) error {
	return errors.New("recorder is unavailable")
}

func TestInvokerRecordsDeclaredAndReportedEffect(t *testing.T) {
	world := commandWorld()
	recorder := &MemoryRecorder{}
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      world,
		Executor:   ReferenceExecutor{World: world},
		UnitOfWork: &MemoryUnitOfWork{},
		Recorder:   recorder,
	}
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if len(recorder.Records) != 1 {
		t.Fatalf("records=%d", len(recorder.Records))
	}
	record := recorder.Records[0]
	if record.Declared == nil || record.Reported == nil ||
		record.Declared.Kind != semantic.DataEffectUpdate || record.Reported.Kind != semantic.DataEffectUpdate {
		t.Fatalf("record=%+v", record)
	}
}

func TestInvokerReportsRecorderFailure(t *testing.T) {
	world := commandWorld()
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      world,
		Executor:   ReferenceExecutor{World: world},
		UnitOfWork: &MemoryUnitOfWork{},
		Recorder:   failingRecorder{},
	}
	_, err := invoker.Invoke(context.Background(), request())
	var executionError *ExecutionError
	if !errors.As(err, &executionError) || executionError.Code != "COMMAND_RECORD_FAILED" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestInvokerDoesNotMarkEffectlessCommandUnobserved checks that a command
// which declares no effect is not reported as "the executor did not enumerate
// its operations". There is no difference to check, so the absence of Applied
// operations is not an unobserved comparison (#3068 follow-up).
func TestInvokerDoesNotMarkEffectlessCommandUnobserved(t *testing.T) {
	world := effectlessCommandWorld()
	recorder := &MemoryRecorder{}
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      world,
		Executor:   fakeExecutor{result: Result{}},
		UnitOfWork: &MemoryUnitOfWork{},
		Recorder:   recorder,
	}
	if _, err := invoker.Invoke(context.Background(), requestForEffectlessWorld()); err != nil {
		t.Fatal(err)
	}
	if len(recorder.Records) != 1 {
		t.Fatalf("records=%d", len(recorder.Records))
	}
	if recorder.Records[0].AppliedUnobserved {
		t.Fatalf("effectless command was marked unobserved: %+v", recorder.Records[0])
	}
	if recorder.Records[0].Declared != nil {
		t.Fatalf("effectless command has a declared effect: %+v", recorder.Records[0])
	}
}

// TestInvokerRecordsDeclaredEffectWithoutReport checks that the record names
// the gap between a declared effect and an executor that reported none, without
// rejecting the call (the effectcommit adapter reports only a receipt).
func TestInvokerRecordsDeclaredEffectWithoutReport(t *testing.T) {
	world := commandWorld()
	recorder := &MemoryRecorder{}
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      world,
		Executor:   fakeExecutor{result: Result{}},
		UnitOfWork: &MemoryUnitOfWork{},
		Recorder:   recorder,
	}
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if len(recorder.Records) != 1 {
		t.Fatalf("records=%d", len(recorder.Records))
	}
	record := recorder.Records[0]
	if record.Declared == nil || !record.ReportedEffectMissing || record.Reported != nil {
		t.Fatalf("record=%+v", record)
	}
	if !record.AppliedUnobserved {
		t.Fatal("declared effect with no applied list must stay unobserved")
	}
}

func TestInvokerDoesNotMarkReportedEffectMissingWithoutDeclaration(t *testing.T) {
	recorder := &MemoryRecorder{}
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      effectlessCommandWorld(),
		Executor:   fakeExecutor{result: Result{}},
		UnitOfWork: &MemoryUnitOfWork{},
		Recorder:   recorder,
	}
	if _, err := invoker.Invoke(context.Background(), requestForEffectlessWorld()); err != nil {
		t.Fatal(err)
	}
	if recorder.Records[0].ReportedEffectMissing {
		t.Fatalf("effectless command marked as missing report: %+v", recorder.Records[0])
	}
}
