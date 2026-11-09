// Package runlog identifies a run and decides whether it may start, resume, or
// must be refused. A run is not accepted meaning: it is a fact about execution,
// so it lives outside the semantic model.
package runlog

import (
	"strings"

	"tei/internal/jsonl"
)

// Status is how a run ended, or that it is still running.
type Status string

const (
	Running   Status = "running"
	Completed Status = "completed"
	Failed    Status = "failed"
	// Blocked means the run stopped because a precondition was not met. It is
	// not a failure: nothing was attempted.
	Blocked Status = "blocked"
)

// Record is one attempt of one run.
type Record struct {
	RunKey     string   `json:"runKey"`
	PlanID     string   `json:"planId"`
	Key        []string `json:"key,omitempty"`
	Attempt    int      `json:"attempt"`
	FenceEpoch int      `json:"fenceEpoch"`
	Status     Status   `json:"status"`
	// Steps lists the operations that finished in this attempt. A resumed attempt
	// continues after them instead of repeating work that is already done.
	Steps []string `json:"steps,omitempty"`
}

// RunKey identifies a run: the plan and the values of its key inputs. The
// separator is a unit separator so a value containing a comma cannot forge a
// different key.
func RunKey(planID string, values []string) string {
	return strings.Join(append([]string{planID}, values...), "\x1f")
}

// Action is what the caller may do with a run.
type Action string

const (
	Start  Action = "start"
	Resume Action = "resume"
	Refuse Action = "refuse"
)

// Decision is the outcome of applying the rules to the last record.
type Decision struct {
	Action     Action `json:"action"`
	Attempt    int    `json:"attempt"`
	FenceEpoch int    `json:"fenceEpoch"`
	Reason     string `json:"reason"`
}

// Decide applies the run rules: a run that never started may start; a run that
// is not finished may resume only when the plan declares checkpoints, and is
// refused otherwise; a finished run is always refused, because repeating it
// needs a different business key.
func Decide(resume string, last Record, found bool) Decision {
	if !found {
		return Decision{Action: Start, Attempt: 1, FenceEpoch: 1, Reason: "the run has not started"}
	}
	if last.Status == Completed {
		return Decision{Action: Refuse, Reason: "the run is already completed; repeating it needs a different business key"}
	}
	if resume != "checkpoint" {
		return Decision{Action: Refuse, Reason: "the run is not finished and the plan does not resume from a checkpoint"}
	}
	return Decision{Action: Resume, Attempt: last.Attempt + 1, FenceEpoch: last.FenceEpoch + 1, Reason: "the run resumes from its checkpoint"}
}

// Owns reports whether this attempt still holds the run. An attempt whose fence
// epoch is stale must not write.
func Owns(last Record, attempt, fenceEpoch int) bool {
	return last.Attempt == attempt && last.FenceEpoch == fenceEpoch
}

// File stores run records as JSON lines.
type File struct {
	Path string
}

// Append implements the journal write.
func (f File) Append(record Record) error {
	return jsonl.Append(f.Path, "run log", record)
}

// Last returns the latest record for a run key.
func (f File) Last(runKey string) (Record, bool, error) {
	records, err := f.Records()
	if err != nil {
		return Record{}, false, err
	}
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].RunKey == runKey {
			return records[i], true, nil
		}
	}
	return Record{}, false, nil
}

// Records reads every record. A missing file is empty; a damaged line is an
// error and the records before it are returned.
func (f File) Records() ([]Record, error) {
	return jsonl.Read(f.Path, "run log", jsonl.DecodeJSON[Record])
}
