package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"tei/internal/plugin"
)

// successfulAttemptReference distinguishes an execution occurrence from its
// content result. It is reference-local evidence, not trusted attestation.
type successfulAttemptReference struct {
	AttemptOrdinal        int    `json:"attemptOrdinal"`
	ExecutionOccurrenceID string `json:"executionOccurrenceId"`
	BuildPlanFingerprint  string `json:"buildPlanFingerprint"`
	ResultFingerprint     string `json:"resultFingerprint"`
	Outcome               string `json:"outcome"`
}

// fakeSuccessfulBuilder stands in for a builder that was actually invoked.
// A caller cannot obtain a receipt merely by copying a result fingerprint.
type fakeSuccessfulBuilder struct {
	runs int
}

func (b *fakeSuccessfulBuilder) Run(
	plan plugin.BuildPlan,
	ordinal int,
	producedBytes []byte,
) (successfulAttemptReference, error) {
	if b == nil {
		return successfulAttemptReference{}, fmt.Errorf("fake builder must be initialized")
	}
	if err := plan.Validate(); err != nil {
		return successfulAttemptReference{}, err
	}
	if ordinal <= 0 {
		return successfulAttemptReference{}, fmt.Errorf("attempt ordinal must be positive")
	}
	planFingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return successfulAttemptReference{}, err
	}
	// The occurrence is deliberately separate from the content fingerprint.
	// Random bytes only distinguish this reference's calls; they do not
	// establish independent trusted execution or grant any authority.
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return successfulAttemptReference{}, fmt.Errorf("create occurrence identity: %w", err)
	}
	b.runs++
	sum := sha256.Sum256(producedBytes)
	return successfulAttemptReference{
		AttemptOrdinal:        ordinal,
		ExecutionOccurrenceID: hex.EncodeToString(nonce),
		BuildPlanFingerprint:  planFingerprint,
		ResultFingerprint:     hex.EncodeToString(sum[:]),
		Outcome:               "succeeded",
	}, nil
}

func (s successfulAttemptReference) validate() error {
	if s.AttemptOrdinal <= 0 {
		return fmt.Errorf("successful attempt ordinal must be positive")
	}
	if err := validateFingerprint(s.ExecutionOccurrenceID); err != nil {
		return fmt.Errorf("successful attempt occurrence identity: %w", err)
	}
	if err := validateFingerprint(s.BuildPlanFingerprint); err != nil {
		return fmt.Errorf("successful attempt plan fingerprint: %w", err)
	}
	if err := validateFingerprint(s.ResultFingerprint); err != nil {
		return fmt.Errorf("successful attempt result fingerprint: %w", err)
	}
	if s.Outcome != "succeeded" {
		return fmt.Errorf("successful attempt must report succeeded")
	}
	return nil
}

func (s successfulAttemptReference) ValidateAgainst(plan plugin.BuildPlan) error {
	if err := s.validate(); err != nil {
		return err
	}
	planFingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return err
	}
	if !strings.EqualFold(s.BuildPlanFingerprint, planFingerprint) {
		return fmt.Errorf("successful attempt does not match exact BuildPlan")
	}
	return nil
}

func (h buildAttemptHistory) latestAttemptOrdinal() int {
	latest := 0
	for _, attempt := range h.FailedAttempts {
		if attempt.AttemptOrdinal > latest {
			latest = attempt.AttemptOrdinal
		}
	}
	for _, attempt := range h.SuccessfulAttempts {
		if attempt.AttemptOrdinal > latest {
			latest = attempt.AttemptOrdinal
		}
	}
	return latest
}

func (h *buildAttemptHistory) AppendSuccess(
	plan plugin.BuildPlan,
	attempt successfulAttemptReference,
) error {
	if h == nil {
		return fmt.Errorf("Build Attempt History is nil")
	}
	if err := h.Validate(); err != nil {
		return fmt.Errorf("existing Build Attempt History is invalid: %w", err)
	}
	planFingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return err
	}
	if !strings.EqualFold(h.BuildPlanFingerprint, planFingerprint) {
		return fmt.Errorf("Build Attempt History does not match exact BuildPlan")
	}
	if err := attempt.ValidateAgainst(plan); err != nil {
		return err
	}
	if attempt.AttemptOrdinal <= h.latestAttemptOrdinal() {
		return fmt.Errorf("successful attempt ordinal must increase")
	}
	for _, existing := range h.SuccessfulAttempts {
		if strings.EqualFold(existing.ExecutionOccurrenceID, attempt.ExecutionOccurrenceID) {
			return fmt.Errorf("execution occurrence cannot be reused")
		}
	}
	h.SuccessfulAttempts = append(h.SuccessfulAttempts, attempt)
	return nil
}
