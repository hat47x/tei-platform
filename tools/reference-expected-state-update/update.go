// Package expectedstateupdate is a read-only, synthetic reference for PoC
// hypothesis H1 (#2483): detect a concurrent update by comparing the state the
// caller read with the current state, and return a business rejection instead
// of letting a later write silently win.
//
// It is not a database, a transaction manager, a version-column design, a
// distributed coordinator, or an accepted API. It never writes a Source and
// never retries. The caller supplies the current state and the updates; the
// reference returns the resulting state, the per-request outcome, and the
// observation.
package expectedstateupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Schema identifies the run record.
const Schema = "tei.reference.expected-state-update/v0"

// Comparison modes.
const (
	CompareValue  = "value"
	CompareDigest = "digest"
)

// Outcomes. A conflict, permission failure or validation failure is a business
// result, never an execution error.
const (
	OutcomeApplied          = "APPLIED"
	OutcomeConflict         = "CONFLICT"
	OutcomeIdempotentReplay = "IDEMPOTENT_REPLAY"
	OutcomeDenied           = "DENIED"
	OutcomeInvalid          = "INVALID"
	OutcomeMissing          = "MISSING"
	OutcomeRejected         = "REJECTED"
)

// Reasons are stable codes, separate from display text.
const (
	ReasonExpectedMismatch     = "expected-state-mismatch"
	ReasonPermission           = "permission"
	ReasonValidation           = "validation"
	ReasonMissingRecord        = "missing-record"
	ReasonRequestInvalid       = "request-invalid"
	ReasonIdempotencyKeyReused = "idempotency-key-reused"
)

// UpdateRequest is one caller's update. ExpectedPresent distinguishes an
// expected empty string from an expected state that was not supplied. In digest
// mode only the digest is carried, so a sensitive value is not put in the
// request.
type UpdateRequest struct {
	Key             string `json:"key"`
	ExpectedPresent bool   `json:"expectedPresent"`
	Expected        string `json:"expected,omitempty"`
	ExpectedDigest  string `json:"expectedDigest,omitempty"`
	NewValue        string `json:"newValue"`
	IdempotencyKey  string `json:"idempotencyKey"`
	Authorized      bool   `json:"authorized"`
	Valid           bool   `json:"valid"`
	// Compare is value (default) or digest.
	Compare string `json:"compare,omitempty"`
}

// AppliedUpdate is the idempotency record of one applied update.
type AppliedUpdate struct {
	Key      string `json:"key"`
	Before   string `json:"before"`
	NewValue string `json:"newValue"`
}

// State is the current records and the idempotency records already applied.
type State struct {
	Records map[string]string        `json:"records"`
	Applied map[string]AppliedUpdate `json:"applied,omitempty"`
}

// Observation is the reproducible count set. ComparisonInputBytes is the number
// of bytes actually compared; digest mode compares only the digest.
type Observation struct {
	Comparisons              int `json:"comparisons"`
	Applied                  int `json:"applied"`
	ConflictRejections       int `json:"conflictRejections"`
	IdempotentReplays        int `json:"idempotentReplays"`
	PermissionRejections     int `json:"permissionRejections"`
	ValidationRejections     int `json:"validationRejections"`
	MissingRecords           int `json:"missingRecords"`
	ComparisonInputBytes     int `json:"comparisonInputBytes"`
	LostUpdatesPrevented     int `json:"lostUpdatesPrevented"`
	IdempotencyKeyRejections int `json:"idempotencyKeyRejections"`
}

// Result is one request's business result.
type Result struct {
	Schema         string      `json:"schema"`
	Key            string      `json:"key"`
	Outcome        string      `json:"outcome"`
	Reason         string      `json:"reason,omitempty"`
	Before         string      `json:"before,omitempty"`
	NewValue       string      `json:"newValue,omitempty"`
	CurrentValue   string      `json:"currentValue,omitempty"`
	ExecutionError bool        `json:"executionError"`
	Observation    Observation `json:"observation"`
}

// RunResult is the deterministic run record.
type RunResult struct {
	Schema      string      `json:"schema"`
	Results     []Result    `json:"results"`
	State       State       `json:"state"`
	Observation Observation `json:"observation"`
	Digest      string      `json:"digest"`
}

func validateState(state State) error {
	for key := range state.Records {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("record key is empty")
		}
	}
	for key := range state.Applied {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("idempotency key is empty")
		}
	}
	return nil
}

func copyState(state State) State {
	out := State{Records: map[string]string{}, Applied: map[string]AppliedUpdate{}}
	for key, value := range state.Records {
		out.Records[key] = value
	}
	for key, applied := range state.Applied {
		out.Applied[key] = applied
	}
	return out
}

func valueDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func compareMode(req UpdateRequest) (string, error) {
	switch strings.TrimSpace(req.Compare) {
	case "", CompareValue:
		return CompareValue, nil
	case CompareDigest:
		return CompareDigest, nil
	default:
		return "", fmt.Errorf("unknown compare mode %q", req.Compare)
	}
}

// Apply evaluates one request against the state and returns the resulting
// state. The input state is never mutated. A conflict is a business rejection
// (ExecutionError=false), not an execution failure.
func Apply(state State, req UpdateRequest) (State, Result) {
	if err := validateState(state); err != nil {
		return state, Result{Schema: Schema, Key: req.Key, Outcome: OutcomeInvalid, Reason: err.Error()}
	}
	out := copyState(state)
	result := Result{Schema: Schema, Key: req.Key}

	if strings.TrimSpace(req.Key) == "" || strings.TrimSpace(req.IdempotencyKey) == "" {
		result.Outcome, result.Reason = OutcomeInvalid, ReasonRequestInvalid
		return out, result
	}
	mode, err := compareMode(req)
	if err != nil {
		result.Outcome, result.Reason = OutcomeInvalid, ReasonRequestInvalid
		return out, result
	}
	if mode == CompareValue && !req.ExpectedPresent {
		result.Outcome, result.Reason = OutcomeInvalid, ReasonRequestInvalid
		return out, result
	}
	if mode == CompareDigest && strings.TrimSpace(req.ExpectedDigest) == "" {
		result.Outcome, result.Reason = OutcomeInvalid, ReasonRequestInvalid
		return out, result
	}

	// A retry of the same request is not applied again. Reusing the key for a
	// different update is rejected rather than silently applied.
	if applied, ok := out.Applied[req.IdempotencyKey]; ok {
		if applied.Key == req.Key && applied.NewValue == req.NewValue {
			result.Outcome = OutcomeIdempotentReplay
			result.Before = applied.Before
			result.NewValue = applied.NewValue
			result.Observation.IdempotentReplays = 1
			return out, result
		}
		result.Outcome, result.Reason = OutcomeRejected, ReasonIdempotencyKeyReused
		result.Observation.IdempotencyKeyRejections = 1
		return out, result
	}

	if !req.Authorized {
		result.Outcome, result.Reason = OutcomeDenied, ReasonPermission
		result.Observation.PermissionRejections = 1
		return out, result
	}
	if !req.Valid {
		result.Outcome, result.Reason = OutcomeInvalid, ReasonValidation
		result.Observation.ValidationRejections = 1
		return out, result
	}

	current, exists := out.Records[req.Key]
	if !exists {
		result.Outcome, result.Reason = OutcomeMissing, ReasonMissingRecord
		result.Observation.MissingRecords = 1
		return out, result
	}

	result.Observation.Comparisons = 1
	matched := false
	switch mode {
	case CompareValue:
		result.Observation.ComparisonInputBytes = len(req.Expected) + len(current)
		matched = current == req.Expected
		result.CurrentValue = current
	case CompareDigest:
		result.Observation.ComparisonInputBytes = len(req.ExpectedDigest)
		matched = valueDigest(current) == strings.TrimSpace(req.ExpectedDigest)
		// The current value is not exposed in digest mode.
	}
	if !matched {
		result.Outcome, result.Reason = OutcomeConflict, ReasonExpectedMismatch
		result.Observation.ConflictRejections = 1
		result.Observation.LostUpdatesPrevented = 1
		return out, result
	}

	out.Records[req.Key] = req.NewValue
	out.Applied[req.IdempotencyKey] = AppliedUpdate{Key: req.Key, Before: current, NewValue: req.NewValue}
	result.Outcome = OutcomeApplied
	result.Before = current
	result.NewValue = req.NewValue
	result.Observation.Applied = 1
	return out, result
}

// Run applies the requests in order to a copy of the state and aggregates the
// observation. The same input yields the same state, results and digest.
func Run(state State, requests []UpdateRequest) (RunResult, error) {
	if err := validateState(state); err != nil {
		return RunResult{}, err
	}
	current := copyState(state)
	run := RunResult{Schema: Schema, Results: make([]Result, 0, len(requests))}
	for _, req := range requests {
		next, result := Apply(current, req)
		current = next
		run.Results = append(run.Results, result)
		addObservation(&run.Observation, result.Observation)
	}
	run.State = current
	raw, err := json.Marshal(struct {
		State       State       `json:"state"`
		Observation Observation `json:"observation"`
	}{current, run.Observation})
	if err != nil {
		return RunResult{}, err
	}
	sum := sha256.Sum256(raw)
	run.Digest = "sha256:" + hex.EncodeToString(sum[:])
	return run, nil
}

func addObservation(total *Observation, delta Observation) {
	total.Comparisons += delta.Comparisons
	total.Applied += delta.Applied
	total.ConflictRejections += delta.ConflictRejections
	total.IdempotentReplays += delta.IdempotentReplays
	total.PermissionRejections += delta.PermissionRejections
	total.ValidationRejections += delta.ValidationRejections
	total.MissingRecords += delta.MissingRecords
	total.ComparisonInputBytes += delta.ComparisonInputBytes
	total.LostUpdatesPrevented += delta.LostUpdatesPrevented
	total.IdempotencyKeyRejections += delta.IdempotencyKeyRejections
}

// SortedRecordKeys is a deterministic key list for report rendering.
func SortedRecordKeys(state State) []string {
	keys := make([]string, 0, len(state.Records))
	for key := range state.Records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
