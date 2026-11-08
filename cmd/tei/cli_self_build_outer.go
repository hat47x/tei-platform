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
	Boundary      string                  `json:"claimBoundary"`
	ChildRun      *selfBuildEntryEvidence `json:"childRun,omitempty"`
	// IntentBinding is §4.8: the bytes this entry held, the fingerprints the
	// child reported, and whether the two readings agree. It is present only
	// when -intent-file is given.
	IntentBinding *selfBuildIntentBinding `json:"intentBinding,omitempty"`
	Error         *cliFailure             `json:"error,omitempty"`
}

// selfBuildIntentBinding separates "which declaration this entry held" from
// "which declaration the child read". Recording only the child's fingerprint
// would not detect a child that read a different declaration (§4.8, SC0-13).
type selfBuildIntentBinding struct {
	// IntentPath and IntentBytesDigest describe the bytes this entry held.
	IntentPath        string `json:"intentPath"`
	IntentBytesDigest string `json:"intentBytesDigest"`
	ChildIntentDigest string `json:"childIntentBytesDigest,omitempty"`
	ChildIntentFP     string `json:"childIntentFingerprint,omitempty"`
	ChildPlanFP       string `json:"childPlanFingerprint,omitempty"`
	CheckPerformed    bool   `json:"intentByteCheckPerformed"`
	CheckMatched      bool   `json:"intentByteCheckMatched"`
	L2BClaimed        bool   `json:"l2bClaimed"`
	BoundaryStatement string `json:"boundary"`
	UnperformedReason string `json:"byteCheckNotPerformedReason,omitempty"`
}

const selfBuildIntentBindingBoundary = "the entry held the declaration and the child reported its own readings; l2bClaimed is always false: agreement of digests is not proof that the child derived the same plan from the same meaning"

const selfBuildOuterBoundary = "one child build launched by this TEI binary from a validated workspace; qualified is always false: an artifact is not acceptance, L3, or runtime authority"

type selfBuildOuterArgs struct {
	Workspace string
	Intent    string
	// IntentFile is the declaration this entry holds and hands to the child.
	// -intent stays the logical ID; the file is the byte-level input (§4.8).
	IntentFile string
	Supply     string
	State      string
	Output     string
	Child      string
	// ChildArgs is the extra argv the caller passes to the child as JSON. The
	// outer shape does not invent those arguments, but it must forward them.
	ChildArgs string
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
		case "-workspace", "-intent", "-intent-file", "-supply", "-state", "-output":
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
	fs.StringVar(&parsed.IntentFile, "intent-file", "", "build intent JSON this entry holds and hands to the child")
	fs.StringVar(&parsed.Supply, "supply", "", "offline module supply passed to the child")
	fs.StringVar(&parsed.State, "state", "", "isolated state directory for this attempt (required)")
	fs.StringVar(&parsed.Output, "output", "", "artifact path written by the child (no-clobber)")
	fs.StringVar(&parsed.Child, "child", "", "exact child executable (required; never resolved from PATH)")
	fs.StringVar(&parsed.ChildArgs, "child-arg", "", "extra child arguments as a JSON array of strings")
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
	// §4.8 L2-a: this entry records the declaration bytes it holds. The bytes
	// are hashed before anything starts, so a later change to the file cannot
	// be mistaken for the declaration that was handed over.
	var intentBinding *selfBuildIntentBinding
	if strings.TrimSpace(parsed.IntentFile) != "" {
		rawIntent, readErr := os.ReadFile(parsed.IntentFile)
		if readErr != nil {
			return selfBuildOuterPrecondition(out, report, "cannot read -intent-file: "+readErr.Error())
		}
		intentBinding = &selfBuildIntentBinding{
			IntentPath:        parsed.IntentFile,
			IntentBytesDigest: digestOfBytes(rawIntent),
			L2BClaimed:        false,
			BoundaryStatement: selfBuildIntentBindingBoundary,
		}
		// The declaration is forwarded to the child so both sides read the same
		// bytes. The child reports its own digest and fingerprints back.
		childArgs = append(childArgs, "-intent", parsed.IntentFile)
	} else if strings.TrimSpace(parsed.Intent) != "" {
		// A logical ID without a declaration cannot be byte-checked. Record why
		// rather than leaving the absence unexplained (SC0-14).
		intentBinding = &selfBuildIntentBinding{
			L2BClaimed:        false,
			BoundaryStatement: selfBuildIntentBindingBoundary,
			UnperformedReason: "-intent names a logical ID only; -intent-file is required to bind the declaration bytes",
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
	// §4.8: read the child's own report of what it read. Only a matching
	// declaration digest plus matching fingerprints make the check performed;
	// the result still does not claim L2-b.
	if intentBinding != nil && intentBinding.IntentBytesDigest != "" {
		intentBinding.ChildIntentDigest = evidence.ChildIntentBytesDigest
		intentBinding.ChildIntentFP = evidence.ChildIntentFingerprint
		intentBinding.ChildPlanFP = evidence.ChildPlanFingerprint
		if evidence.ChildIntentBytesDigest != "" {
			intentBinding.CheckPerformed = true
			intentBinding.CheckMatched = evidence.ChildIntentBytesDigest == intentBinding.IntentBytesDigest
			if !intentBinding.CheckMatched {
				appendSelfBuildFailure(&evidence, "child reported a different declaration byte digest than the one this entry held")
			}
		} else {
			intentBinding.UnperformedReason = "child did not report the declaration byte digest it read"
		}
	}
	report.IntentBinding = intentBinding
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
