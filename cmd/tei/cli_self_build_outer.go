package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"tei/internal/validation"
	"tei/internal/workspace"
)

// This file adds the §4 outer shape that §4.6 recorded as still missing: the
// workspace/intent/supply/state/output form, its refusal conditions (SC0-01〜
// SC0-05), and a `qualified` field that is always false.
//
// The existing -child/-artifact form stays the primitive. Both forms launch a
// separately named child, neither builds in-process, and neither is in the
// read-only discovery catalog (#3414 §4.5, D-3).

// selfBuildOuterEvidence adds the §4 fields around the primitive record. Every
// field is an observation; `qualified` is written explicitly as false so a
// reader cannot infer qualification from an artifact's existence.
type selfBuildOuterEvidence struct {
	SchemaVersion string                  `json:"schemaVersion"`
	OK            bool                    `json:"ok"`
	Nature        string                  `json:"nature"`
	Workspace     string                  `json:"workspace"`
	Intent        string                  `json:"intent,omitempty"`
	SupplyRoot    string                  `json:"supplyRoot,omitempty"`
	StateDir      string                  `json:"buildState"`
	Output        string                  `json:"output"`
	Qualified     bool                    `json:"qualified"`
	IntentRecord  *IntentDeclaration      `json:"intentDeclaration,omitempty"`
	SupplyRecord  *SupplyCheck            `json:"supplyCheck,omitempty"`
	Boundary      string                  `json:"claimBoundary"`
	ChildRun      *selfBuildEntryEvidence `json:"childRun,omitempty"`
	Error         *cliFailure             `json:"error,omitempty"`
}

const selfBuildOuterBoundary = "one child build launched by this TEI binary from a validated workspace; qualified is always false: an artifact is not acceptance, L3, or runtime authority"

type selfBuildOuterArgs struct {
	Workspace string
	Intent    string
	Supply    string
	State     string
	Output    string
	Child     string
	// ChildArgs is the extra argv the caller passes to the child as JSON. The
	// outer shape does not invent those arguments, but it must forward them.
	ChildArgs string
	// IntentFile is the declared build intent. It is required by SC0-01.
	IntentFile string
	// SupplyVerifier is an exact executable that runs SB-0b's classification.
	// When named, its result is required before the builder starts (SC0-02).
	SupplyVerifier string
	// SupplyVerifierArg is extra argv for the verifier as a JSON array.
	SupplyVerifierArg string
}

// looksLikeOuterShape reports whether the caller used any §4 flag. A caller that
// passes none of them keeps the primitive -child/-artifact behaviour.
func looksLikeOuterShape(args []string) bool {
	for _, arg := range args {
		name := arg
		if index := strings.IndexByte(arg, '='); index >= 0 {
			name = arg[:index]
		}
		switch name {
		case "-workspace", "-intent", "-supply", "-state", "-output":
			return true
		}
	}
	return false
}

func parseSelfBuildOuterArgs(args []string) (selfBuildOuterArgs, error) {
	var parsed selfBuildOuterArgs
	fs := flag.NewFlagSet("self-build", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&parsed.Workspace, "workspace", "", "TEI workspace that declares the build intent")
	fs.StringVar(&parsed.Intent, "intent", "", "build intent ID; recorded, not resolved by this entry")
	fs.StringVar(&parsed.Supply, "supply", "", "offline module supply passed to the child")
	fs.StringVar(&parsed.State, "state", "", "isolated state directory for this attempt (required)")
	fs.StringVar(&parsed.Output, "output", "", "artifact path written by the child (no-clobber)")
	fs.StringVar(&parsed.Child, "child", "", "exact child executable (required; never resolved from PATH)")
	fs.StringVar(&parsed.ChildArgs, "child-arg", "", "extra child arguments as a JSON array of strings")
	fs.StringVar(&parsed.IntentFile, "intent-file", "", "declared build intent JSON (required)")
	fs.StringVar(&parsed.SupplyVerifier, "supply-verifier", "", "exact executable that classifies the supply (SB-0b)")
	fs.StringVar(&parsed.SupplyVerifierArg, "supply-verifier-arg", "", "extra arguments for the supply verifier as a JSON array")
	if err := fs.Parse(args); err != nil {
		return selfBuildOuterArgs{}, err
	}
	if fs.NArg() != 0 {
		return selfBuildOuterArgs{}, errors.New("self-build does not accept positional arguments")
	}
	return parsed, nil
}

// runSelfBuildOuterShape enforces the §4 refusal conditions before anything is
// started, then delegates to the primitive driver with the mapped arguments.
func runSelfBuildOuterShape(args []string, out io.Writer) (int, error) {
	parsed, err := parseSelfBuildOuterArgs(args)
	if err != nil {
		return 2, fmt.Errorf("INVALID_ARGUMENT: %w", err)
	}
	report := selfBuildOuterEvidence{
		SchemaVersion: selfBuildEntrySchema,
		Nature:        "thin CLI entry: one child build launched from a validated workspace",
		Workspace:     parsed.Workspace,
		Intent:        parsed.Intent,
		SupplyRoot:    parsed.Supply,
		StateDir:      parsed.State,
		Output:        parsed.Output,
		Qualified:     false,
		Boundary:      selfBuildOuterBoundary,
	}

	// SC0-09: the builder is an argument of this command, not a property of the
	// workspace, so a missing one is settled first. A bare name is refused later
	// by the primitive driver, which owns the digest.
	if strings.TrimSpace(parsed.Child) == "" {
		return selfBuildOuterPrecondition(out, report, "-child is required: the entry never resolves a builder from PATH")
	}
	// SC0-01/SC0-05: the workspace is required and must not have validation
	// errors. A build never starts on an unvalidated workspace.
	if parsed.Workspace == "" {
		return selfBuildOuterPrecondition(out, report, "-workspace is required")
	}
	loaded, err := workspace.Load(parsed.Workspace)
	if err != nil {
		return selfBuildOuterPrecondition(out, report, "workspace load failed: "+err.Error())
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			return selfBuildOuterPrecondition(out, report, "workspace has validation errors at "+diagnostic.Code+": "+diagnostic.Message)
		}
	}
	// SC0-01: the build intent must be declared. A path is required, and the
	// declaration must be a JSON object with a formatVersion and a subject id.
	// The entry records the declaration; it does not derive a BuildPlan from it.
	if strings.TrimSpace(parsed.IntentFile) == "" {
		return selfBuildOuterPrecondition(out, report, "-intent-file is required: a build attempt needs a declared build intent")
	}
	declaration, err := readIntentDeclaration(parsed.IntentFile)
	if err != nil {
		report.Error = &cliFailure{Code: selfBuildCodeIntentInvalid, Message: err.Error()}
		if encodeErr := encodeIndented(out, report); encodeErr != nil {
			return 2, encodeErr
		}
		return 2, nil
	}
	report.IntentRecord = &declaration

	// SC0-02: when a verifier is named, its classification must pass before the
	// builder starts. Without one, the record says the check was delegated and
	// does not read that as a pass.
	if strings.TrimSpace(parsed.SupplyVerifier) != "" {
		verifierArgs := []string{}
		if strings.TrimSpace(parsed.SupplyVerifierArg) != "" {
			if err := json.Unmarshal([]byte(parsed.SupplyVerifierArg), &verifierArgs); err != nil {
				return 2, fmt.Errorf("INVALID_ARGUMENT: -supply-verifier-arg is not a JSON array of strings: %w", err)
			}
		}
		check, err := runSupplyCheck(parsed.SupplyVerifier, verifierArgs, parsed.Supply)
		if err != nil {
			report.Error = &cliFailure{Code: selfBuildCodeSupplyRefused, Message: err.Error()}
			if encodeErr := encodeIndented(out, report); encodeErr != nil {
				return 2, encodeErr
			}
			return 2, nil
		}
		report.SupplyRecord = &check
		if !check.Verified {
			report.Error = &cliFailure{Code: selfBuildCodeSupplyRefused, Message: "the supply did not pass SB-0b classification: " + check.FailureCause}
			if encodeErr := encodeIndented(out, report); encodeErr != nil {
				return 2, encodeErr
			}
			return 2, nil
		}
	} else {
		delegated := supplyCheckDelegatedRecord(parsed.Supply)
		report.SupplyRecord = &delegated
	}

	// SC0-03: the state directory is required and has no default, so two
	// attempts cannot silently share one.
	if parsed.State == "" {
		return selfBuildOuterPrecondition(out, report, "-state is required: a shared state directory would make two attempts indistinguishable")
	}
	// SC0-04: no-clobber, decided before the child starts.
	if parsed.Output == "" {
		return selfBuildOuterPrecondition(out, report, "-output is required")
	}
	if _, err := os.Stat(parsed.Output); err == nil {
		return selfBuildOuterPrecondition(out, report, "output already exists; self-build does not overwrite: "+parsed.Output)
	} else if !os.IsNotExist(err) {
		return selfBuildOuterPrecondition(out, report, "cannot inspect the output path: "+err.Error())
	}

	childArgs := []string{}
	if strings.TrimSpace(parsed.ChildArgs) != "" {
		if err := json.Unmarshal([]byte(parsed.ChildArgs), &childArgs); err != nil {
			return 2, fmt.Errorf("INVALID_ARGUMENT: -child-arg is not a JSON array of strings: %w", err)
		}
	}
	if parsed.Supply != "" {
		childArgs = append(childArgs, "-supply", parsed.Supply)
	}
	childArgs = append(childArgs, "-artifact-out", parsed.Output, "-state", parsed.State)
	evidence, err := driveSelfBuild(parsed.Child, childArgs, parsed.Output, "")
	if err != nil {
		return 1, err
	}
	report.ChildRun = &evidence
	report.OK = evidence.CrossCheckOK && evidence.FailureCause == ""
	if !report.OK {
		report.Error = &cliFailure{Code: "SELF_BUILD_CHILD_FAILED", Message: evidence.FailureCause}
	}
	if err := encodeIndented(out, report); err != nil {
		return 1, err
	}
	if !report.OK {
		return 1, nil
	}
	return 0, nil
}

func selfBuildOuterPrecondition(out io.Writer, report selfBuildOuterEvidence, message string) (int, error) {
	report.Error = &cliFailure{Code: "SELF_BUILD_PRECONDITION_FAILED", Message: message}
	if err := encodeIndented(out, report); err != nil {
		return 2, err
	}
	return 2, nil
}

func encodeIndented(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
