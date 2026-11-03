package expectedstateupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
)

// RecordSchema identifies the observation record produced by this reference.
//
// The record is the reproducible part of one run: outcomes, rejection reasons,
// the comparison byte counts, and digests of the values. Raw before/current/new
// values are not stored, because #2564 item 3 adopted server-side comparison and
// the record may outlive the request.
const RecordSchema = "tei.reference.expected-state-record/v1"

// RecordedResult is the stored form of one business result.
type RecordedResult struct {
	Key                  string      `json:"key"`
	IdempotencyKeyDigest string      `json:"idempotencyKeyDigest"`
	Outcome              string      `json:"outcome"`
	Reason               string      `json:"reason,omitempty"`
	BeforeDigest         string      `json:"beforeDigest,omitempty"`
	CurrentValueDigest   string      `json:"currentValueDigest,omitempty"`
	NewValueDigest       string      `json:"newValueDigest,omitempty"`
	ExecutionError       bool        `json:"executionError"`
	Observation          Observation `json:"observation"`
}

// Record is one reproducible run record. Digest covers Entries in order.
type Record struct {
	Schema  string           `json:"schema"`
	Entries []RecordedResult `json:"entries"`
	Digest  string           `json:"digest"`
}

func recordDigest(entries []RecordedResult) (string, error) {
	encoded, err := json.Marshal(entries)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func digestOf(value string) string {
	if value == "" {
		return ""
	}
	return valueDigest(value)
}

// RecordRun builds the observation record for one run. Requests and results must
// have the same length and order.
func RecordRun(requests []UpdateRequest, results []Result) (Record, error) {
	if len(requests) != len(results) {
		return Record{}, fmt.Errorf("expected-state record: %d requests but %d results", len(requests), len(results))
	}
	record := Record{Schema: RecordSchema, Entries: make([]RecordedResult, 0, len(results))}
	for i, result := range results {
		record.Entries = append(record.Entries, RecordedResult{
			Key:                  result.Key,
			IdempotencyKeyDigest: valueDigest(requests[i].IdempotencyKey),
			Outcome:              result.Outcome,
			Reason:               result.Reason,
			BeforeDigest:         digestOf(result.Before),
			CurrentValueDigest:   digestOf(result.CurrentValue),
			NewValueDigest:       digestOf(result.NewValue),
			ExecutionError:       result.ExecutionError,
			Observation:          result.Observation,
		})
	}
	digest, err := recordDigest(record.Entries)
	if err != nil {
		return Record{}, err
	}
	record.Digest = digest
	return record, nil
}

// EncodeRecord writes the record as canonical indented JSON plus a newline.
func EncodeRecord(w io.Writer, record Record) error {
	encoded, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(append(encoded, '\n'))
	return err
}

// DecodeRecord reads a record and refuses a schema or digest mismatch.
func DecodeRecord(r io.Reader) (Record, error) {
	var record Record
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return Record{}, err
	}
	if record.Schema != RecordSchema {
		return Record{}, fmt.Errorf("expected-state record: unexpected schema %q", record.Schema)
	}
	digest, err := recordDigest(record.Entries)
	if err != nil {
		return Record{}, err
	}
	if digest != record.Digest {
		return Record{}, fmt.Errorf("expected-state record: digest does not match the entries")
	}
	return record, nil
}

// VerifyRecord checks that a rerun reproduces the stored outcomes and counts.
// A divergence is reported with the entry index so the observation can be
// inspected without storing raw values.
func VerifyRecord(requests []UpdateRequest, results []Result, record Record) error {
	if len(requests) != len(results) || len(results) != len(record.Entries) {
		return fmt.Errorf("expected-state record: run has %d requests, %d results, %d recorded entries", len(requests), len(results), len(record.Entries))
	}
	for i := range results {
		current, err := RecordRun(requests[i:i+1], results[i:i+1])
		if err != nil {
			return err
		}
		if current.Entries[0] != record.Entries[i] {
			return fmt.Errorf("expected-state record: entry %d (key %q) does not reproduce the recorded observation", i, results[i].Key)
		}
	}
	return nil
}

// RetainRuns applies the default retention policy: keep the newest reports in
// the given order (oldest first) and drop the rest. The storage location is the
// caller's; this reference keeps one file per run.
func RetainRuns(records []Record, keep int) ([]Record, error) {
	if keep < 1 {
		return nil, fmt.Errorf("expected-state record: retention must keep at least one run, got %d", keep)
	}
	if len(records) <= keep {
		return append([]Record(nil), records...), nil
	}
	return append([]Record(nil), records[len(records)-keep:]...), nil
}
