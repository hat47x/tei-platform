// Package effectlog records what happened to an effect that runs outside the
// transaction. The record is not accepted meaning: it says an attempt happened
// and whether an independent observation confirmed it.
package effectlog

import (
	"tei/internal/jsonl"
)

// Record is one attempt at an effect.
type Record struct {
	RuleID     string `json:"ruleId"`
	Stage      string `json:"stage"`
	Capability string `json:"capability"`
	Succeeded  bool   `json:"succeeded"`
	// Observed is true only when an independent read confirmed the effect. A
	// success code from the capability is not an observation.
	Observed bool   `json:"observed"`
	Detail   string `json:"detail,omitempty"`
}

// Journal appends records and reads them back.
type Journal interface {
	Append(Record) error
	Records() ([]Record, error)
}

// File stores records as JSON lines. Each append is a separate line, so a
// partially written file still yields the records before the damaged line as an
// error is reported rather than silently dropped.
type File struct {
	Path string
}

// Append implements Journal.
func (f File) Append(record Record) error {
	return jsonl.Append(f.Path, "effect journal", record)
}

// Records implements Journal.
func (f File) Records() ([]Record, error) {
	return jsonl.Read(f.Path, "effect journal", jsonl.DecodeJSON[Record])
}
