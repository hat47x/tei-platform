package command

import (
	"context"
	"sync"

	"tei/internal/semantic"
)

// OperationRecord is what the boundary observed for one Command execution:
// the declared effect from the capability and the effect the executor reported.
type OperationRecord struct {
	Request  Request
	Declared *semantic.DataEffect
	Reported *semantic.DataEffect
	Outcome  *semantic.Outcome
}

// Recorder observes successful Command executions. It is optional; a recorder
// failure rolls the reference transaction back before Commit.
type Recorder interface {
	Record(ctx context.Context, record OperationRecord) error
}

// MemoryRecorder keeps operation records in memory for tests and inspection.
type MemoryRecorder struct {
	mu      sync.Mutex
	Records []OperationRecord
}

// Record appends one operation record.
func (r *MemoryRecorder) Record(_ context.Context, record OperationRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Records = append(r.Records, record)
	return nil
}
