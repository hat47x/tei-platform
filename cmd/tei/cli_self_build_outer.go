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
	// IntentBinding is §4.8: the bytes this entry held, the fingerprints the
	// child reported, and whether the two readings agree. It is present only
	// when -intent-file is given.
	IntentBinding *selfBuildIntentBinding `json:"intentBinding,omitempty"`
	// PlanReceipt is §4.8 L2-b's material as recorded by the resolver child.
	// Present only when the caller named a resolver. It never makes the entry
	// claim L2: the artifact came from a different child.
	PlanReceipt *PlanReceipt `json:"planReceipt,omitempty"`
	// PlanCrossCheck is §4.8's third observation: the resolver's plan and the
	// builder's reported plan, compared. Agreement of two self-reports still
	// does not establish L2-b.
	PlanCrossCheck *planCrossCheck `json:"planCrossCheck,omitempty"`
	// PlanRecompute is the entry's own derivation of the receipt's plan
	// fingerprint. It checks the resolver's arithmetic, not the builder's work.
	PlanRecompute *planRecompute `json:"planRecompute,omitempty"`
	// PlanArtifactBinding is §4.8(c): one validated BuildArtifactLock tying the
	// reported plan fingerprint to the artifact digest, with the digest
	// independently read back from disk.
	PlanArtifactBinding *planArtifactBinding `json:"planArtifactBinding,omitempty"`
	// DefinitionBinding is the entry-side reading of §4.36: the builder reports
	// the definition it read, and this compares that statement with the digest the
	// plan names. The builder refuses such a build itself; this is the check that
	// does not depend on the builder doing so.
	DefinitionBinding *selfBuildDefinitionBinding `json:"definitionBinding,omitempty"`
	// ArtifactReproducibility is §4.21: a second, independent build of the same
	// plan produced the same artifact bytes. Together with planArtifactBinding it
	// is the observation §4.8(c) asks for; it is never present with state
	// "reproduced" when the caller did not ask for it.
	ArtifactReproducibility *selfBuildReproducibility `json:"artifactReproducibility,omitempty"`
	// L2B is the combined claim (§4.23). The per-observation flags above each
	// still say false, because no single observation is the claim; this record
	// answers whether all of them hold together and names every one that did not.
	L2B   *selfBuildL2BClaim `json:"l2b,omitempty"`
	Error *cliFailure        `json:"error,omitempty"`
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
	// EntryIntentFingerprint is what the entry itself derived from the bytes it
	// held (SB-C0 §4.13, I-1a). It is a second, independent basis for comparing
	// readings: the byte check above depends on the child reporting the bytes it
	// read, while this depends only on the entry's own reading.
	EntryIntentFingerprint string `json:"entryIntentFingerprint,omitempty"`
	// FingerprintCheckPerformed is true only when both sides produced a
	// fingerprint. An absent side leaves it false with the reason, never true.
	FingerprintCheckPerformed bool `json:"intentFingerprintCheckPerformed"`
	// FingerprintMatched compares the child's reported intent fingerprint with
	// the entry's own. It is a different question from the byte check: two
	// different byte sequences cannot produce the same declaration fingerprint
	// unless they normalise to the same declaration, and the same bytes always
	// do. Disagreement means the child read a different declaration.
	FingerprintMatched bool   `json:"intentFingerprintMatched"`
	L2BClaimed         bool   `json:"l2bClaimed"`
	BoundaryStatement  string `json:"boundary"`
	UnperformedReason  string `json:"byteCheckNotPerformedReason,omitempty"`
	FingerprintReason  string `json:"fingerprintCheckNotPerformedReason,omitempty"`
}

const selfBuildIntentBindingBoundary = "the entry held the declaration and the child reported its own readings; l2bClaimed is always false: agreement of digests is not proof that the child derived the same plan from the same meaning"

const selfBuildOuterBoundary = "one child build launched by this TEI binary from a validated workspace; qualified is always false: an artifact is not acceptance, L3, or runtime authority"

type selfBuildOuterArgs struct {
	Workspace string
	Intent    string
	// IntentFile is the declaration this entry holds and hands to the child,
	// and the declared build intent required by SC0-01. -intent stays the
	// logical ID; the file is the byte-level input (§4.8).
	IntentFile string
	Supply     string
	State      string
	Output     string
	Child      string
	// ChildArgs is the extra argv the caller passes to the child as JSON. The
	// outer shape does not invent those arguments, but it must forward them.
	ChildArgs string
	// SupplyVerifier is an exact executable that runs SB-0b's classification.
	// When named, its result is required before the builder starts (SC0-02).
	SupplyVerifier string
	// SupplyVerifierArg is extra argv for the verifier as a JSON array.
	SupplyVerifierArg string
	// IntentResolver is an exact executable that resolves the declaration into
	// a plan, per §4.7 I-2. It is started as a child; the entry reads its JSON.
	IntentResolver string
	// IntentResolverArg is extra argv for the resolver as a JSON array.
	IntentResolverArg string
	// ExpectedPlanFingerprint is a value the caller declared out of band. When
	// given, the resolver's plan must equal it or the attempt stops.
	ExpectedPlanFingerprint string
	// RebuildCheck asks this entry to build the same plan a second time, in a
	// fresh state directory, and compare the artifact bytes. Without it the
	// reproducibility record says why it did not run (§4.21).
	//
	// §4.30: when the caller names a build catalog this entry derives the plan
	// itself, which is the only route that can establish L2-b. That route already
	// opts into the expensive, evidence-rich path, so the second build is turned on
	// for it unless the caller named -rebuild-check=false. RebuildCheckSet records
	// that the caller spoke, so an explicit false is honoured rather than defaulted.
	RebuildCheck    bool
	RebuildCheckSet bool
	// InProcess selects deriving the plan in this process from a caller-named
	// build catalog (§4.22 step 2). It is mutually exclusive with IntentResolver.
	InProcess selfBuildInProcessArgs
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
		case "-workspace", "-intent", "-intent-file", "-supply", "-state", "-output", "-rebuild-check",
			"-build-catalog", "-build-catalog-digest", "-build-source-root":
			return true
		}
	}
	return false
}

// newSelfBuildOuterFlagSet registers every flag this shape accepts. It is separate
// from parsing so the help text and the parser cannot drift apart.
func newSelfBuildOuterFlagSet(parsed *selfBuildOuterArgs) *flag.FlagSet {
	fs := flag.NewFlagSet("self-build", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&parsed.Workspace, "workspace", "", "TEI workspace that declares the build intent")
	fs.StringVar(&parsed.Intent, "intent", "", "build intent ID; recorded, not resolved by this entry")
	fs.StringVar(&parsed.IntentFile, "intent-file", "", "declared build intent JSON this entry holds and hands to the child (required, SC0-01)")
	fs.StringVar(&parsed.Supply, "supply", "", "offline module supply passed to the child")
	fs.StringVar(&parsed.State, "state", "", "isolated state directory for this attempt (required)")
	fs.StringVar(&parsed.Output, "output", "", "artifact path written by the child (no-clobber)")
	fs.StringVar(&parsed.Child, "child", "", "exact child executable (required; never resolved from PATH)")
	fs.StringVar(&parsed.ChildArgs, "child-arg", "", "extra child arguments as a JSON array of strings")
	fs.StringVar(&parsed.SupplyVerifier, "supply-verifier", "", "exact executable that classifies the supply (SB-0b)")
	fs.StringVar(&parsed.SupplyVerifierArg, "supply-verifier-arg", "", "extra arguments for the supply verifier as a JSON array")
	fs.StringVar(&parsed.IntentResolver, "intent-resolver", "", "exact executable that resolves the declaration into a plan (I-2)")
	fs.StringVar(&parsed.IntentResolverArg, "intent-resolver-arg", "", "extra arguments for the intent resolver as a JSON array")
	fs.StringVar(&parsed.ExpectedPlanFingerprint, "expected-plan-fingerprint", "", "planFingerprint the resolved plan must equal")
	fs.BoolVar(&parsed.RebuildCheck, "rebuild-check", false, "build the same plan a second time in fresh state and compare the artifact bytes (§4.21); on by default when -build-catalog is given, and -rebuild-check=false keeps it off (§4.30)")
	fs.StringVar(&parsed.InProcess.CatalogPath, "build-catalog", "", "build catalog JSON the entry resolves against in process (§4.22)")
	fs.StringVar(&parsed.InProcess.CatalogDigest, "build-catalog-digest", "", "sha256 digest of -build-catalog; required with it")
	fs.StringVar(&parsed.InProcess.SourceRoot, "build-source-root", "", "root the declaration's source references resolve from; required with -build-catalog")
	return fs
}

func parseSelfBuildOuterArgs(args []string) (selfBuildOuterArgs, error) {
	var parsed selfBuildOuterArgs
	fs := newSelfBuildOuterFlagSet(&parsed)
	if err := fs.Parse(args); err != nil {
		return selfBuildOuterArgs{}, err
	}
	if fs.NArg() != 0 {
		return selfBuildOuterArgs{}, errors.New("self-build does not accept positional arguments")
	}
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "rebuild-check" {
			parsed.RebuildCheckSet = true
		}
	})
	return parsed, nil
}

// applyEvidenceDefaults decides the evidence a caller did not ask for (§4.30).
//
// Naming a build catalog is already a request for the strongest route this entry
// can take: it is the only route that can establish L2-b. That route pays for the
// evidence, so the second build — the one observation it can still add — is turned
// on for it. A caller that says -rebuild-check=false keeps the lighter attempt.
//
// The child route is untouched: turning the second build on for it would pay the
// cost twice over without enabling any claim, which is the asymmetry §4.25 named.
func applyEvidenceDefaults(parsed selfBuildOuterArgs) selfBuildOuterArgs {
	if parsed.InProcess.requested() && !parsed.RebuildCheckSet {
		parsed.RebuildCheck = true
	}
	return parsed
}

// runSelfBuildOuterShape enforces the §4 refusal conditions before anything is
// started, then delegates to the primitive driver with the mapped arguments.
func runSelfBuildOuterShape(args []string, out io.Writer) (int, error) {
	parsed, err := parseSelfBuildOuterArgs(args)
	if err != nil {
		return 2, fmt.Errorf("INVALID_ARGUMENT: %w", err)
	}
	if err := parsed.InProcess.validate(); err != nil {
		return 2, fmt.Errorf("INVALID_ARGUMENT: %w", err)
	}
	parsed = applyEvidenceDefaults(parsed)
	if parsed.InProcess.requested() && strings.TrimSpace(parsed.IntentResolver) != "" {
		// §4.16 warned that two derivation routes writing the same record make its
		// meaning depend on the route. One attempt gets one derivation.
		return 2, errors.New("INVALID_ARGUMENT: -intent-resolver and -build-catalog cannot both be given; choose the route for this attempt")
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
	// SC0-01: a build attempt must declare its intent. Either a declaration
	// path or a logical intent ID satisfies that; only the declaration path
	// carries bytes this entry can hash, so the absence of bytes is recorded
	// rather than treated as a declaration (§4.8 L2-a, SC0-14).
	if strings.TrimSpace(parsed.IntentFile) == "" && strings.TrimSpace(parsed.Intent) == "" {
		return selfBuildOuterPrecondition(out, report, "one of -intent-file or -intent is required: a build attempt needs a declared build intent")
	}
	if strings.TrimSpace(parsed.IntentFile) != "" {
		declaration, err := readIntentDeclaration(parsed.IntentFile)
		if err != nil {
			report.Error = &cliFailure{Code: selfBuildCodeIntentInvalid, Message: err.Error()}
			if encodeErr := encodeIndented(out, report); encodeErr != nil {
				return 2, encodeErr
			}
			return 2, nil
		}
		report.IntentRecord = &declaration
	}

	// §4.7 I-2: resolution is delegated to an exact resolver child. The entry
	// starts it, reads its receipt, and records what was resolved. A resolver
	// that fails, or a receipt whose plan differs from the declared expectation,
	// stops the attempt before the builder starts.
	if parsed.InProcess.requested() {
		intentBytesDigest := ""
		if report.IntentRecord != nil {
			intentBytesDigest = report.IntentRecord.Digest
		}
		receipt := resolveIntentInProcess(
			parsed.IntentFile,
			intentBytesDigest,
			parsed.InProcess,
			parsed.ExpectedPlanFingerprint,
		)
		report.PlanReceipt = &receipt
		// Recompute the plan fingerprint from the plan the receipt carries, for the
		// same reason the child route does: it proves the recorded plan is the one
		// the recorded fingerprint describes. In this route it also checks the
		// round-trip through JSON, which the child route gets for free.
		if receipt.State == intentResolutionResolved {
			recompute := recomputeReceiptPlanFingerprint(receipt.ReceiptJSON)
			report.PlanRecompute = &recompute
			if recompute.State == planRecomputeMismatched {
				report.Error = &cliFailure{Code: selfBuildCodeIntentResolutionFailed, Message: recompute.FailureCause}
				if encodeErr := encodeIndented(out, report); encodeErr != nil {
					return 2, encodeErr
				}
				return 2, nil
			}
		}
		if receipt.State != intentResolutionResolved {
			report.Error = &cliFailure{Code: selfBuildCodeIntentResolutionFailed, Message: receipt.FailureCause}
			if encodeErr := encodeIndented(out, report); encodeErr != nil {
				return 2, encodeErr
			}
			return 2, nil
		}
		if parsed.ExpectedPlanFingerprint != "" && !receipt.PlanMatchesExpected {
			report.Error = &cliFailure{Code: selfBuildCodePlanMismatch, Message: receipt.FailureCause}
			if encodeErr := encodeIndented(out, report); encodeErr != nil {
				return 2, encodeErr
			}
			return 2, nil
		}
	}

	if strings.TrimSpace(parsed.IntentResolver) != "" {
		resolverArgs := []string{}
		if strings.TrimSpace(parsed.IntentResolverArg) != "" {
			if err := json.Unmarshal([]byte(parsed.IntentResolverArg), &resolverArgs); err != nil {
				return 2, fmt.Errorf("INVALID_ARGUMENT: -intent-resolver-arg is not a JSON array of strings: %w", err)
			}
		}
		resolverArgs = append(resolverArgs, "-intent", parsed.IntentFile)
		receipt := resolveIntentVia(parsed.IntentResolver, resolverArgs, parsed.ExpectedPlanFingerprint)
		report.PlanReceipt = &receipt
		// The entry recomputes the plan fingerprint from the plan the receipt
		// carries. A receipt whose plan does not hash to its own reported
		// fingerprint stops the attempt: the resolver's arithmetic is part of
		// what the entry is relying on.
		if receipt.State == intentResolutionResolved {
			recompute := recomputeReceiptPlanFingerprint(receipt.ReceiptJSON)
			report.PlanRecompute = &recompute
			if recompute.State == planRecomputeMismatched {
				report.Error = &cliFailure{Code: selfBuildCodeIntentResolutionFailed, Message: recompute.FailureCause}
				if encodeErr := encodeIndented(out, report); encodeErr != nil {
					return 2, encodeErr
				}
				return 2, nil
			}
		}
		if receipt.State != intentResolutionResolved {
			report.Error = &cliFailure{Code: selfBuildCodeIntentResolutionFailed, Message: receipt.FailureCause}
			if encodeErr := encodeIndented(out, report); encodeErr != nil {
				return 2, encodeErr
			}
			return 2, nil
		}
		if parsed.ExpectedPlanFingerprint != "" && !receipt.PlanMatchesExpected {
			report.Error = &cliFailure{Code: selfBuildCodePlanMismatch, Message: receipt.FailureCause}
			if encodeErr := encodeIndented(out, report); encodeErr != nil {
				return 2, encodeErr
			}
			return 2, nil
		}
	}

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
	// Three of the child's flags are this entry's own contract with it: the
	// artifact path, the isolated state and the supply. A caller that repeats
	// one of them in -child-arg would give the child two values for one thing,
	// and which wins depends on the child's flag parser (Go takes the last,
	// some take the first). Either way the record would describe one value
	// while the child used another. Refusing is the same rule the read-only
	// command allowlist applies to its own flags: no silent loss of intent.
	if reserved := reservedChildFlagIn(childArgs); reserved != "" {
		return 2, fmt.Errorf("INVALID_ARGUMENT: -child-arg must not set %s; this entry supplies it (pass child-specific arguments only)", reserved)
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
		// The entry's own reading of the same bytes, derived with the package the
		// reference resolver shares. This is the basis the byte check cannot
		// provide: the byte check needs the child to report what it read, while
		// this needs nobody but the entry.
		if report.IntentRecord != nil {
			intentBinding.EntryIntentFingerprint = report.IntentRecord.Fingerprint
		}
		// The declaration is NOT injected into the builder's argv: the entry does
		// not invent child arguments, and the builder it is pointed at may have no
		// -intent flag at all. A caller that wants the byte check passes the
		// reporting arguments through -child-arg; without them the check records
		// why it was not performed.
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

		// The fingerprint comparison runs on its own evidence: the entry's
		// derived value and the child's reported one. It does not inherit the
		// byte check's outcome, so a child that reports a fingerprint but not
		// the bytes it read still gets compared.
		switch {
		case intentBinding.EntryIntentFingerprint == "":
			intentBinding.FingerprintReason = "the entry derived no intent fingerprint from the declaration it held"
		case evidence.ChildIntentFingerprint == "":
			intentBinding.FingerprintReason = "the child did not report the intent fingerprint of the declaration it read"
		default:
			intentBinding.FingerprintCheckPerformed = true
			intentBinding.FingerprintMatched = evidence.ChildIntentFingerprint == intentBinding.EntryIntentFingerprint
			if !intentBinding.FingerprintMatched {
				appendSelfBuildFailure(&evidence, "child reported an intent fingerprint that differs from the one this entry derived")
			}
		}
	}
	report.IntentBinding = intentBinding

	// §4.8's third observation: compare the resolver's plan with the plan the
	// builder reports. This runs after both children, so a mismatch is recorded
	// as a finding instead of being mistaken for a precondition.
	if report.PlanReceipt != nil {
		record := crossCheckPlan(report.PlanReceipt.PlanFingerprint, evidence.ChildPlanFingerprint)
		report.PlanCrossCheck = &record
		if record.State == planCrossCheckMismatched {
			appendSelfBuildFailure(&evidence, planMismatchCause().Error())
		}
		// §4.8(c): bind the builder's plan fingerprint to the artifact digest it
		// reported, and confirm that digest against the bytes on disk. This is a
		// separate record from the cross-check: agreement of two plan fingerprints
		// says nothing about which artifact came out.
		// The entry's own derivation goes in as a basis that does not come from
		// the child. It is used only when the child is silent; which one the lock
		// carries is recorded beside it.
		entryPlanFingerprint := ""
		if report.PlanRecompute != nil {
			entryPlanFingerprint = report.PlanRecompute.RecomputedFingerprint
		}
		binding := bindPlanToArtifact(
			report.PlanReceipt,
			entryPlanFingerprint,
			evidence.ChildPlanFingerprint,
			evidence.ChildArtifact,
			evidence.CrossChecked,
			evidence.ChildLock,
		)
		report.PlanArtifactBinding = &binding
		if binding.State == planArtifactMismatched {
			appendSelfBuildFailure(&evidence, planArtifactMismatchCause(binding))
		}

		// §4.36's builder-side refusal only holds for a builder that implements
		// it. This compares the same two values on this side, so a builder that
		// skipped the check still cannot report a definition the plan does not
		// name without the attempt failing.
		definitionBinding := bindDefinitionToPlan(report.PlanReceipt, evidence.ChildDefinitionDigest)
		report.DefinitionBinding = &definitionBinding
		if definitionBinding.State == definitionBindingFailed {
			appendSelfBuildFailure(&evidence, definitionBinding.Reason)
		}
	}

	// §4.21: a second independent build answers §4.8(c) — whether the artifact on
	// disk is the one this plan produces — which the lock alone cannot.
	//
	// The record is always emitted. When the caller did not ask, it says so with a
	// reason, because a missing field would let a reader take silence for
	// agreement (the same rule SC0-18 applies to the child's declaration report).
	unperformedReason := "the caller did not ask for a second build (-rebuild-check)"
	if parsed.InProcess.requested() && parsed.RebuildCheckSet {
		// In-process, the second build is the default, so its absence is a
		// decision the caller made rather than a step it never reached.
		unperformedReason = "the caller turned off the second build (-rebuild-check=false)"
	}
	reproducibility := selfBuildReproducibility{
		State:             reproducibilityUnperformed,
		UnperformedReason: unperformedReason,
		Boundary:          reproducibilityBoundary,
	}
	if parsed.RebuildCheck {
		reproducibility = checkArtifactReproducibility(
			parsed.Child,
			childArgs,
			evidence.ChildArtifact,
			evidence.Invoked.ExecutableDigest,
			parsed.State,
			evidence.ChildStateDir,
			parsed.Output,
		)
		switch reproducibility.State {
		case reproducibilityMismatched:
			appendSelfBuildFailure(&evidence, "a second build of the same plan produced different artifact bytes")
		case reproducibilityUnperformed:
			appendSelfBuildFailure(&evidence, "the reproducibility check did not run: "+reproducibility.UnperformedReason)
		}
	}
	report.ArtifactReproducibility = &reproducibility
	report.OK = evidence.CrossCheckOK && evidence.FailureCause == ""
	if !report.OK {
		report.Error = &cliFailure{Code: "SELF_BUILD_CHILD_FAILED", Message: evidence.FailureCause}
	}
	// §4.23: the L2-b predicate is evaluated last, over every observation this
	// attempt recorded. It is reported even when the attempt failed, so a reader
	// can see which conditions were missing rather than only that ok is false.
	l2b := assessL2B(report)
	report.L2B = &l2b
	if err := encodeIndented(out, report); err != nil {
		return 1, err
	}
	if !report.OK {
		return 1, nil
	}
	return 0, nil
}

// reservedChildFlags are the child flags this entry writes itself. A caller may
// not override them through -child-arg.
var reservedChildFlags = []string{"-artifact-out", "-state", "-supply"}

// reservedChildFlagIn reports the first reserved flag present in the caller's
// child arguments, including the -flag=value spelling, or "" when none is.
func reservedChildFlagIn(childArgs []string) string {
	for _, arg := range childArgs {
		name := arg
		if index := strings.IndexByte(name, '='); index >= 0 {
			name = name[:index]
		}
		for _, reserved := range reservedChildFlags {
			if name == reserved {
				return reserved
			}
		}
	}
	return ""
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
