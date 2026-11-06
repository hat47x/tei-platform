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
