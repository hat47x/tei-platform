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
	// OutcomeUndetermined states why no outcome was returned although the
	// capability declares some. No declared rule selects among several
	// outcomes, so an executor must not present one as the decision.
	OutcomeUndetermined string `json:"outcomeUndetermined,omitempty"`
	// Applied lists the record-level operations the executor applied, so a
	// reader can compare the declaration with what actually happened. It is
	// empty for executors that cannot enumerate their operations.
	Applied []AppliedOperation `json:"applied,omitempty"`
	// Differences keeps the applied operations the declaration does not
	// describe, so a reader sees that the declaration and the execution did
	// not fully agree. They are evidence, not a rejection.
	Differences []Difference `json:"differences,omitempty"`
	// AppliedUnobserved is true when the executor did not enumerate the
	// operations it applied. It keeps "no difference was found" distinct from
	// "the executor could not say", which an empty Applied list alone cannot.
	AppliedUnobserved bool `json:"appliedUnobserved,omitempty"`
	// ReportedEffectMissing is true when the capability declared an effect but
	// the executor returned none. The boundary does not reject this yet: doing
	// so would make every executor that reports only a commit receipt (the
	// effectcommit adapter) an unknown outcome. The field records the gap so a
	// reader does not treat "no reported effect" as "the declared effect was
	// applied" (#2996).
	ReportedEffectMissing bool `json:"reportedEffectMissing,omitempty"`
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
