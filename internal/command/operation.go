package command

import (
	"context"
	"sync"

	"tei/internal/semantic"
)

// OperationRecord is what the boundary observed for one Command execution:
// the declared effect from the capability, the effect the executor reported,
// and the record-level operations the executor said it applied.
type OperationRecord struct {
	Request  Request              `json:"request"`
	Declared *semantic.DataEffect `json:"declared,omitempty"`
	Reported *semantic.DataEffect `json:"reported,omitempty"`
	Outcome  *semantic.Outcome    `json:"outcome,omitempty"`
	// Applied lists the record-level operations the executor applied, so a
	// reader can compare the declaration with what actually happened. It is
	// empty for executors that cannot enumerate their operations.
	Applied []AppliedOperation `json:"applied,omitempty"`
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
