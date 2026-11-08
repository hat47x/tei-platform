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
	Path           string `json:"path"`
	Digest         string `json:"digest"`
	FormatVersion  int    `json:"formatVersion,omitempty"`
	SubjectID      string `json:"subjectId,omitempty"`
	OutputKind     string `json:"outputKind,omitempty"`
	RecipeEntry    string `json:"recipeEntrypoint,omitempty"`
	Resolved       bool   `json:"resolved"`
	ResolutionNote string `json:"resolutionNote,omitempty"`
	// IntentFingerprint is the entry's OWN derivation from the declaration it
	// read (SB-C0 §4.13, I-1a). Before this, the only intent fingerprint in the
	// record came from a child's report, so the entry could not compare a
	// reading it held against one it was told.
	IntentFingerprint string `json:"intentFingerprint,omitempty"`
	// StrictlyRead reports whether the declaration satisfied the same reader
	// the reference resolver uses. A declaration that only looks like JSON is
	// recorded with false and the reason, not silently accepted.
	StrictlyRead         bool   `json:"strictlyRead"`
	StrictReadFailedNote string `json:"strictReadFailure,omitempty"`
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

// readIntentDeclaration reads the declaration with the same strict reader the
// reference resolver uses (SB-C0 §4.13, I-1a), then records the entry's own
// intent fingerprint beside the declaration's byte digest.
//
// The two are different readings and the record keeps them apart:
//   - Digest is the bytes this entry held.
//   - IntentFingerprint is what those bytes mean as a declaration.
//
// A declaration that the shared reader refuses is still recorded, with
// StrictlyRead=false and the reason, so a caller learns why the entry's
// fingerprint is absent instead of reading silence as agreement.
func readIntentDeclaration(path string) (IntentDeclaration, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return IntentDeclaration{}, err
	}
	raw, err := os.ReadFile(absolute)
	if err != nil {
		return IntentDeclaration{}, fmt.Errorf("build intent is not readable: %w", err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return IntentDeclaration{}, fmt.Errorf("build intent is not a JSON object: %w", err)
	}
	if top == nil {
		return IntentDeclaration{}, errors.New("build intent is not a JSON object")
	}
	declaration := IntentDeclaration{
		Path:           absolute,
		Digest:         digestOfBytes(raw),
		Resolved:       false,
		ResolutionNote: intentResolutionNote,
	}
	if rawVersion, ok := top["formatVersion"]; ok {
		var version int
		if err := json.Unmarshal(rawVersion, &version); err != nil {
			return IntentDeclaration{}, fmt.Errorf("build intent formatVersion is not an integer: %w", err)
		}
		declaration.FormatVersion = version
	} else {
		return IntentDeclaration{}, errors.New("build intent has no formatVersion")
	}
	var subject struct {
		ID string `json:"id"`
	}
	if rawSubject, ok := top["subject"]; ok {
		if err := json.Unmarshal(rawSubject, &subject); err != nil {
			return IntentDeclaration{}, fmt.Errorf("build intent subject is malformed: %w", err)
		}
	}
	if strings.TrimSpace(subject.ID) == "" {
		return IntentDeclaration{}, errors.New("build intent subject has no id")
	}
	declaration.SubjectID = subject.ID
	var outputKind string
	if rawKind, ok := top["outputKind"]; ok {
		if err := json.Unmarshal(rawKind, &outputKind); err != nil {
			return IntentDeclaration{}, fmt.Errorf("build intent outputKind is malformed: %w", err)
		}
	}
	declaration.OutputKind = outputKind
	var recipe struct {
		Entrypoint string `json:"entrypoint"`
	}
	if rawRecipe, ok := top["recipe"]; ok {
		if err := json.Unmarshal(rawRecipe, &recipe); err != nil {
			return IntentDeclaration{}, fmt.Errorf("build intent recipe is malformed: %w", err)
		}
	}
	declaration.RecipeEntry = recipe.Entrypoint

	// The entry's own reading, through the package the reference resolver now
	// shares. This is not resolution: no BuildPlan is derived.
	intent, readErr := buildintent.ParseIntent(raw)
	if readErr != nil {
		declaration.StrictlyRead = false
		declaration.StrictReadFailedNote = readErr.Error()
		return declaration, nil
	}
	declaration.StrictlyRead = true
	if fingerprint, err := buildintent.IntentFingerprint(intent); err == nil {
		declaration.IntentFingerprint = fingerprint
	} else {
		declaration.StrictReadFailedNote = "intent fingerprint could not be derived: " + err.Error()
	}
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
