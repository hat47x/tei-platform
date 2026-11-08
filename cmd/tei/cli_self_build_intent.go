package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"tei/internal/buildintent"
)

// This file closes two of the refusal conditions that §4.6 recorded as still
// missing: SC0-01 (the build intent must be declared) and SC0-02 (a supply that
// fails SB-0b's classification stops the attempt).
//
// The entry does not interpret the intent and does not re-implement the supply
// classifier. It records the declaration it was given and, when the caller
// names a supply checker, runs that exact executable and reads the classifier's
// own JSON. Copying the classifier into cmd/tei would create the second
// implementation §35.2 forbids.

const (
	selfBuildCodeIntentMissing = "SELF_BUILD_INTENT_MISSING"
	selfBuildCodeIntentInvalid = "SELF_BUILD_INTENT_INVALID"
	selfBuildCodeSupplyRefused = "SELF_BUILD_SUPPLY_UNVERIFIED"
	supplyCheckDelegated       = "delegated-to-child"
	supplyCheckVerified        = "verified"
	supplyCheckRefused         = "refused"
)

// IntentDeclaration is what the entry read about the declared build intent.
type IntentDeclaration struct {
	Path          string `json:"path"`
	Digest        string `json:"digest"`
	FormatVersion int    `json:"formatVersion,omitempty"`
	SubjectID     string `json:"subjectId,omitempty"`
	OutputKind    string `json:"outputKind,omitempty"`
	RecipeEntry   string `json:"recipeEntrypoint,omitempty"`
	// Fingerprint is the declaration's own identity, derived by the entry with
	// the shared reader (SB-C0 §4.13 I-1a). It is not copied from a child's
	// report, so a comparison against a child's reading has a basis the entry
	// holds itself.
	Fingerprint    string `json:"intentFingerprint,omitempty"`
	Resolved       bool   `json:"resolved"`
	ResolutionNote string `json:"resolutionNote,omitempty"`
}

// SupplyCheck records what the entry learned about the supply before starting
// the builder.
type SupplyCheck struct {
	State          string `json:"state"`
	VerifierPath   string `json:"verifierPath,omitempty"`
	VerifierDigest string `json:"verifierDigest,omitempty"`
	Verified       bool   `json:"verified"`
	Classification string `json:"classification,omitempty"`
	FailureCause   string `json:"failureCause,omitempty"`
	Note           string `json:"note,omitempty"`
}

const intentResolutionNote = "read and digested, not resolved: the entry does not derive a BuildPlan from the intent"

// readIntentDeclaration reads the declared intent as a JSON object and records
// a few top-level fields. It deliberately does not treat those fields as a
// resolved build plan; a declaration that parses is not a resolved plan.
func readIntentDeclaration(path string) (IntentDeclaration, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return IntentDeclaration{}, err
	}
	raw, err := os.ReadFile(absolute)
	if err != nil {
		return IntentDeclaration{}, fmt.Errorf("build intent is not readable: %w", err)
	}
	declaration := IntentDeclaration{
		Path:           absolute,
		Digest:         digestOfBytes(raw),
		Resolved:       false,
		ResolutionNote: intentResolutionNote,
	}
	// The declaration is read with the same strict reader the reference
	// resolver uses (SB-C0 §4.13 I-1a), so the entry and the resolver cannot
	// accept different declarations. The entry still does not derive a BuildPlan
	// from it: a declaration that parses is not a resolved plan.
	intent, err := buildintent.ParseIntent(raw)
	if err != nil {
		return IntentDeclaration{}, err
	}
	declaration.FormatVersion = intent.FormatVersion
	declaration.SubjectID = string(intent.Subject.ID)
	declaration.OutputKind = string(intent.OutputKind)
	declaration.RecipeEntry = intent.Recipe.Entrypoint
	// The declaration's own identity, derived by the entry rather than copied
	// from a child's report. This is the value a later byte-level comparison can
	// use as a basis that does not depend on another process.
	fingerprint, err := buildintent.IntentFingerprint(intent)
	if err != nil {
		return IntentDeclaration{}, fmt.Errorf("build intent fingerprint: %w", err)
	}
	declaration.Fingerprint = fingerprint
	return declaration, nil
}

// runSupplyCheck starts the caller-named supply checker and reads the classifier
// JSON it emits. A checker that exits non-zero or reports verified=false stops
// the attempt before the builder starts (SC0-02).
func runSupplyCheck(exactVerifier string, verifierArgs []string, supplyRoot string) (SupplyCheck, error) {
	digest, err := digestOfFile(exactVerifier)
	if err != nil {
		return SupplyCheck{}, fmt.Errorf("supply checker is not readable: %w", err)
	}
	check := SupplyCheck{
		State:          supplyCheckRefused,
		VerifierPath:   exactVerifier,
		VerifierDigest: digest,
	}
	args := append([]string{}, verifierArgs...)
	if strings.TrimSpace(supplyRoot) != "" {
		args = append(args, "-supply", supplyRoot)
	}
	cmd := exec.Command(exactVerifier, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	exitCode := 0
	if runErr != nil {
		exitCode = -1
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}
	if exitCode != 0 {
		check.FailureCause = "supply checker exited unsuccessfully: " + strings.TrimSpace(stderr.String())
		return check, nil
	}
	// The checker's own evidence shape (SB-0b) carries `verified`, an optional
	// `classification`, and an optional `failureCause`. Unknown or absent values
	// stay unknown rather than being read as success.
	var evidence struct {
		SchemaVersion  string `json:"schemaVersion"`
		Verified       bool   `json:"verified"`
		Classification string `json:"classification"`
		FailureCause   string `json:"failureCause"`
	}
	if err := json.Unmarshal([]byte(stdout.String()), &evidence); err != nil {
		check.FailureCause = "supply checker output is not one JSON object: " + err.Error()
		return check, nil
	}
	check.Classification = evidence.Classification
	check.FailureCause = evidence.FailureCause
	check.Verified = evidence.Verified
	if evidence.SchemaVersion == "" {
		check.FailureCause = "supply checker did not report a schemaVersion; its result is not treated as evidence"
		check.Verified = false
		return check, nil
	}
	if evidence.Verified {
		check.State = supplyCheckVerified
	}
	return check, nil
}

// supplyCheckDelegatedRecord states the default: the entry did not check the
// supply, and the child is the only place it is checked.
func supplyCheckDelegatedRecord(supplyRoot string) SupplyCheck {
	if strings.TrimSpace(supplyRoot) == "" {
		return SupplyCheck{State: "not-applicable", Note: "no supply was named, so there was nothing to check"}
	}
	return SupplyCheck{
		State: supplyCheckDelegated,
		Note:  "the entry did not run a supply check; SB-0b's classification is not applied before the child starts",
	}
}
