// Package access decides whether a capability may touch what its permissions
// declare, and records the decision. The record is not accepted meaning: it says
// that a check happened and what it found.
package access

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/core"
	"tei/internal/permission"
)

// Decision is the outcome of checking one capability.
//
// It is the same type as the record a journal keeps, on purpose. They were two
// separately declared structs with identical fields, converted by
// `Record(decision)`: adding a field to one and not the other would have
// silently dropped it at the conversion, with nothing to report the loss. One
// type cannot drift from itself, and the JSON written to the journal is the JSON
// the caller reads.
type Decision = Record

// Record is one check in the journal.
type Record struct {
	Capability string   `json:"capability"`
	Allowed    bool     `json:"allowed"`
	Missing    []string `json:"missing,omitempty"`
}

// Guard checks capabilities against the granted permissions and records every
// decision. A nil journal keeps the check but drops the record.
type Guard struct {
	Granted []permission.Spec
	Journal Journal
}

// Journal appends access records.
type Journal interface {
	Append(Record) error
}

// Check reports whether the capability's requirements are covered, and records
// the decision. A journal failure is returned because an unrecorded decision is
// not the same as no decision.
func (g Guard) Check(capabilityID core.CanonicalID, required []permission.Spec) (Decision, error) {
	missing := permission.Missing(g.Granted, required)
	decision := Decision{Capability: string(capabilityID), Allowed: len(missing) == 0}
	for _, requirement := range missing {
		decision.Missing = append(decision.Missing, string(requirement.Kind)+":"+string(requirement.Access)+" "+requirement.Scope)
	}
	if g.Journal != nil {
		if err := g.Journal.Append(decision); err != nil {
			return decision, err
		}
	}
	return decision, nil
}

// File stores access records as JSON lines.
type File struct {
	Path string
}

// Append implements Journal.
func (f File) Append(record Record) error {
	if err := os.MkdirAll(filepath.Dir(f.Path), 0o755); err != nil {
		return fmt.Errorf("prepare access journal %s: %w", f.Path, err)
	}
	file, err := os.OpenFile(f.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open access journal %s: %w", f.Path, err)
	}
	defer file.Close()
	line, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode access record: %w", err)
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write access journal %s: %w", f.Path, err)
	}
	return nil
}

// Records reads the journal. A missing file is empty; a damaged line is an error
// and the records before it are returned.
func (f File) Records() ([]Record, error) {
	file, err := os.Open(f.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open access journal %s: %w", f.Path, err)
	}
	defer file.Close()
	var records []Record
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for line := 1; scanner.Scan(); line++ {
		text := scanner.Bytes()
		if len(text) == 0 {
			continue
		}
		var record Record
		if err := json.Unmarshal(text, &record); err != nil {
			return records, fmt.Errorf("access journal %s line %d: %w", f.Path, line, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return records, fmt.Errorf("read access journal %s: %w", f.Path, err)
	}
	return records, nil
}
