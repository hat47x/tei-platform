// Package jsonl reads and appends the JSON Lines files the runtime decisions
// are recorded in.
//
// Four places used to carry their own copy of this loop: the access journal, the
// issued-grant log, the run log, and the effect journal. Each copy had to be kept
// identical by hand — the same owner-only mode, the same missing-file rule, the
// same "a damaged line is an error and the earlier lines are returned", the same
// scanner buffer. A fix to one (the permission mode, for instance) did not reach
// the others unless someone remembered.
//
// The rules live here once. What differs between callers — the name used in the
// error and the record type — is passed in.
package jsonl

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Append writes one value as a line, creating the file if needed.
//
// The file is owner-only. A record that other users can rewrite is a record that
// can be made to say something else, and every caller here records a decision.
func Append(path, label string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("prepare %s %s: %w", label, path, err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open %s %s: %w", label, path, err)
	}
	defer file.Close()
	line, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode %s record: %w", label, err)
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write %s %s: %w", label, path, err)
	}
	return nil
}

// Read decodes every line into the given slice of records.
//
// A missing file is empty, because a journal that was never written has no
// records — refusing that would make the first append impossible to check. A
// damaged line is an error, and the records before it are returned: dropping the
// rest silently would turn corruption into "nothing after this happened".
func Read[T any](path, label string, decode func([]byte) (T, error)) ([]T, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open %s %s: %w", label, path, err)
	}
	defer file.Close()
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read %s %s: %w", label, path, err)
	}
	return ReadBytes(raw, label, decode)
}

// ReadBytes is Read over bytes the caller already holds. It exists because a
// received file is not always read from disk by this package: the external
// analysis intake fingerprints the bytes as they arrived and then classifies
// them, and a second copy of the line rules there would be the drift this
// package was created to stop. The rules are the same ones Read documents; the
// label stands in for the path, which the caller already knows.
func ReadBytes[T any](raw []byte, label string, decode func([]byte) (T, error)) ([]T, error) {
	var records []T
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for line := 1; scanner.Scan(); line++ {
		text := scanner.Bytes()
		if len(text) == 0 {
			continue
		}
		record, err := decode(text)
		if err != nil {
			return records, fmt.Errorf("%s %s line %d: %w", label, "bytes", line, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return records, fmt.Errorf("read %s %s: %w", label, "bytes", err)
	}
	return records, nil
}

// DecodeJSON is the decoder most callers need. It is a function rather than an
// implicit default so a caller that needs a stricter or looser rule can supply
// its own.
func DecodeJSON[T any](raw []byte) (T, error) {
	var value T
	err := json.Unmarshal(raw, &value)
	return value, err
}
