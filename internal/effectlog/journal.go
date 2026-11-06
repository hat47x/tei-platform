// Package effectlog records what happened to an effect that runs outside the
// transaction. The record is not accepted meaning: it says an attempt happened
// and whether an independent observation confirmed it.
package effectlog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	if err := os.MkdirAll(filepath.Dir(f.Path), 0o755); err != nil {
		return fmt.Errorf("prepare effect journal %s: %w", f.Path, err)
	}
	file, err := os.OpenFile(f.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open effect journal %s: %w", f.Path, err)
	}
	defer file.Close()
	line, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode effect record: %w", err)
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write effect journal %s: %w", f.Path, err)
	}
	return nil
}

// Records implements Journal.
func (f File) Records() ([]Record, error) {
	file, err := os.Open(f.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open effect journal %s: %w", f.Path, err)
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
			return records, fmt.Errorf("effect journal %s line %d: %w", f.Path, line, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return records, fmt.Errorf("read effect journal %s: %w", f.Path, err)
	}
	return records, nil
}
