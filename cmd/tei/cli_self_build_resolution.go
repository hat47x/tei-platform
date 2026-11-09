package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// This file adds the I-2 connection §4.7 selected: the entry does not
// interpret the build intent itself. It starts an exact resolver child, reads
// the receipt that child prints, and records what was resolved and on what
// basis.
//
// It does NOT claim L2. §4.8 splits the claim into two layers, and both stay
// false here:
//
//	L2-a (hand-off)   this entry passed a specific declaration to a child
//	L2-b (interpretation) a child read that declaration and derived a plan
//
// The entry can record the hand-off. It cannot say the plan it received came
// from the same bytes as the artifact, because the artifact was produced by a
// separate builder child whose own reading of the intent is also only reported.
// l2Claimed is therefore always false, and the boundary says why.

const (
	intentResolutionRefused  = "refused"
	intentResolutionResolved = "resolved"

	selfBuildCodeIntentResolutionFailed = "SELF_BUILD_INTENT_RESOLUTION_FAILED"
	selfBuildCodePlanMismatch           = "SELF_BUILD_PLAN_MISMATCH"

	// The receipt shape this entry knows how to read. Anything else is refused
	// rather than guessed at.
	derivedCacheAuthority = "derived-cache"
)

// PlanReceipt records what the resolver child produced. Every field is copied
// from the child's JSON; nothing here is recomputed by the entry.
type PlanReceipt struct {
	State                    string `json:"state"`
	ResolverPath             string `json:"resolverPath,omitempty"`
	ResolverDigest           string `json:"resolverDigest,omitempty"`
	ReceiptDigest            string `json:"receiptDigest,omitempty"`
	Authority                string `json:"authority,omitempty"`
	FormatVersion            int    `json:"receiptFormatVersion,omitempty"`
	IntentFingerprint        string `json:"intentFingerprint,omitempty"`
	PlanFingerprint          string `json:"planFingerprint,omitempty"`
	SourceClosureFingerprint string `json:"sourceClosureFingerprint,omitempty"`
	ExpectedPlanFingerprint  string `json:"expectedPlanFingerprint,omitempty"`
	PlanMatchesExpected      bool   `json:"planMatchesExpected"`
	L2Claimed                bool   `json:"l2Claimed"`
	BindingNote              string `json:"bindingNote,omitempty"`
	FailureCause             string `json:"failureCause,omitempty"`
	// DerivedBy names the route that produced this plan. "entry-in-process" means
	// this binary resolved the declaration itself; empty means a resolver child
	// did. Keeping the route explicit stops a reader from crediting the entry
	// with a derivation a child performed (§4.22 step 2).
	DerivedBy string `json:"derivedBy,omitempty"`
	// The inputs the in-process route used. They are empty for a child receipt,
	// which records the resolver executable instead.
	CatalogPath   string `json:"catalogPath,omitempty"`
	CatalogDigest string `json:"catalogDigest,omitempty"`
	SourceRoot    string `json:"sourceRoot,omitempty"`
	// ReceiptJSON is the resolver's raw stdout. The entry keeps it so the plan
	// fingerprint can be recomputed from the plan the receipt carries, instead
	// of trusting the reported value.
	ReceiptJSON string `json:"-"`
}

const derivedByEntryInProcess = "entry-in-process"

const planReceiptBindingNote = "the entry passed the same declaration file to a resolver child, recomputed the plan fingerprint from the plan inside the receipt, and compared it with a caller-declared value; it cannot prove that the builder child used the same reading"

// resolveIntentVia runs an exact resolver executable and reads one JSON receipt
// from its stdout. A non-zero exit, an unparseable receipt, a receipt that is
// not a derived cache, or a receipt without a planFingerprint is a refusal.
func resolveIntentVia(exactResolver string, resolverArgs []string, expectedPlanFingerprint string) PlanReceipt {
	receipt := PlanReceipt{
		State:                   intentResolutionRefused,
		ResolverPath:            exactResolver,
		ExpectedPlanFingerprint: expectedPlanFingerprint,
		L2Claimed:               false,
		BindingNote:             planReceiptBindingNote,
	}
	digest, err := digestOfFile(exactResolver)
	if err != nil {
		receipt.FailureCause = "intent resolver is not readable: " + err.Error()
		return receipt
	}
	receipt.ResolverDigest = digest

	cmd := exec.Command(exactResolver, resolverArgs...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if runErr != nil {
		exitCode := -1
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = runErr.Error()
		}
		receipt.FailureCause = fmt.Sprintf("intent resolver exited with %d: %s", exitCode, message)
		return receipt
	}

	raw := strings.TrimSpace(stdout.String())
	var parsed struct {
		FormatVersion            int    `json:"formatVersion"`
		Authority                string `json:"authority"`
		IntentFingerprint        string `json:"intentFingerprint"`
		PlanFingerprint          string `json:"planFingerprint"`
		SourceClosureFingerprint string `json:"sourceClosureFingerprint"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		receipt.FailureCause = "intent resolver output is not one JSON object: " + err.Error()
		return receipt
	}
	receipt.ReceiptJSON = raw
	receipt.ReceiptDigest = digestOfBytes([]byte(raw))
	receipt.FormatVersion = parsed.FormatVersion
	receipt.Authority = parsed.Authority
	receipt.IntentFingerprint = parsed.IntentFingerprint
	receipt.PlanFingerprint = parsed.PlanFingerprint
	receipt.SourceClosureFingerprint = parsed.SourceClosureFingerprint

	// A receipt that does not identify itself as a derived cache is not the
	// shape this entry knows. Refuse instead of treating unknown output as a
	// resolved plan.
	if parsed.Authority != derivedCacheAuthority {
		receipt.FailureCause = "intent resolver receipt does not declare authority=derived-cache; its result is not treated as a resolved plan"
		return receipt
	}
	if strings.TrimSpace(parsed.PlanFingerprint) == "" {
		receipt.FailureCause = "intent resolver receipt has no planFingerprint"
		return receipt
	}
	receipt.State = intentResolutionResolved
	if expectedPlanFingerprint != "" {
		receipt.PlanMatchesExpected = parsed.PlanFingerprint == expectedPlanFingerprint
		if !receipt.PlanMatchesExpected {
			receipt.FailureCause = "the resolved plan fingerprint differs from the caller-declared expectation"
		}
	}
	return receipt
}

// readIntentResolvable reports whether a declaration is at least readable as
// the intent shape the resolver expects. It does not resolve anything: a
// declaration that parses is not a resolved plan.
func readIntentResolvable(path string) (int, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, "", fmt.Errorf("build intent is not readable: %w", err)
	}
	var top struct {
		FormatVersion int `json:"formatVersion"`
		Subject       struct {
			ID string `json:"id"`
		} `json:"subject"`
	}
	if err := json.Unmarshal(raw, &top); err != nil {
		return 0, "", fmt.Errorf("build intent is not a JSON object: %w", err)
	}
	if top.FormatVersion == 0 {
		return 0, "", errors.New("build intent has no formatVersion")
	}
	if strings.TrimSpace(top.Subject.ID) == "" {
		return 0, "", errors.New("build intent subject has no id")
	}
	return top.FormatVersion, top.Subject.ID, nil
}
