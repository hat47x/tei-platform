package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"tei/internal/buildintent"
)

// This file adds the in-process resolution path SB-C0 §4.22 migration step 2
// called for: the entry reads a caller-named build catalog, derives the plan
// itself, and records what it used.
//
// §4.22 decided the catalog is data the caller names and digests, so the entry
// never chooses a builder from a table compiled into it. Naming the catalog and
// its digest is what makes "this entry resolved the declaration with this data"
// a statement about observed inputs rather than a claim about defaults.
//
// The child path (-intent-resolver) and this path are mutually exclusive. §4.16
// warned that two paths writing the same output make the record's meaning depend
// on the route; refusing both at once keeps one attempt to one derivation.

const inProcessDerivationNote = "this entry read the declaration, resolved it against the catalog named by the caller, and derived the plan itself; the catalog is the caller's data and this records which data was used, not that the catalog is correct"

// selfBuildInProcessArgs is the flag set that selects in-process resolution.
type selfBuildInProcessArgs struct {
	CatalogPath   string
	CatalogDigest string
	SourceRoot    string
}

// validate reports whether the three flags were given together. A partial set is
// a refusal: a catalog without its digest cannot be recorded, and without a
// source root the declaration's references cannot be read.
func (a selfBuildInProcessArgs) validate() error {
	given := 0
	for _, value := range []string{a.CatalogPath, a.CatalogDigest, a.SourceRoot} {
		if strings.TrimSpace(value) != "" {
			given++
		}
	}
	if given == 0 {
		return nil
	}
	if given != 3 {
		return fmt.Errorf("-build-catalog, -build-catalog-digest and -build-source-root must be given together")
	}
	return nil
}

func (a selfBuildInProcessArgs) requested() bool {
	return strings.TrimSpace(a.CatalogPath) != ""
}

// resolveIntentInProcess derives the plan in this process and returns the same
// PlanReceipt shape the child path produces, so every downstream observation
// keeps working without knowing which route produced the plan.
//
// It reads the declaration itself and refuses when those bytes differ from the
// ones the entry already recorded: resolving a declaration the entry did not
// hold would make the receipt describe a different input.
func resolveIntentInProcess(
	intentPath string,
	intentBytesDigest string,
	args selfBuildInProcessArgs,
	expectedPlanFingerprint string,
) PlanReceipt {
	receipt := PlanReceipt{
		State:                   intentResolutionRefused,
		ExpectedPlanFingerprint: expectedPlanFingerprint,
		DerivedBy:               derivedByEntryInProcess,
		CatalogPath:             args.CatalogPath,
		CatalogDigest:           args.CatalogDigest,
		SourceRoot:              args.SourceRoot,
		Authority:               derivedCacheAuthority,
		L2Claimed:               false,
		BindingNote:             inProcessDerivationNote,
	}

	catalogBytes, err := os.ReadFile(args.CatalogPath)
	if err != nil {
		receipt.FailureCause = "build catalog is not readable: " + err.Error()
		return receipt
	}
	actualCatalogDigest := digestOfBytes(catalogBytes)
	if actualCatalogDigest != args.CatalogDigest {
		receipt.FailureCause = fmt.Sprintf(
			"build catalog digest is %s but the caller declared %s", actualCatalogDigest, args.CatalogDigest,
		)
		return receipt
	}

	catalog, err := buildintent.ParseBuildCatalog(catalogBytes)
	if err != nil {
		receipt.FailureCause = err.Error()
		return receipt
	}

	intentBytes, err := os.ReadFile(intentPath)
	if err != nil {
		receipt.FailureCause = "build intent is not readable: " + err.Error()
		return receipt
	}
	if intentBytesDigest != "" && digestOfBytes(intentBytes) != intentBytesDigest {
		receipt.FailureCause = "the declaration changed between being recorded and being resolved; this receipt does not describe the bytes the entry held"
		return receipt
	}
	intent, err := buildintent.ParseIntent(intentBytes)
	if err != nil {
		receipt.FailureCause = err.Error()
		return receipt
	}

	context := buildintent.NewBuildContext(args.SourceRoot, catalog)
	derived, err := buildintent.Resolve(intent, context)
	if err != nil {
		receipt.FailureCause = "resolve: " + err.Error()
		return receipt
	}

	receipt.IntentFingerprint = derived.IntentFingerprint
	receipt.PlanFingerprint = derived.PlanFingerprint
	receipt.SourceClosureFingerprint = derived.ResolutionContext.SourceClosure
	receipt.Authority = derived.Authority
	receipt.FormatVersion = derived.FormatVersion
	receipt.State = intentResolutionResolved

	// Keep the derived receipt in the same shape a resolver child emits, so the
	// observations downstream (the plan-to-artifact binding and the arithmetic
	// recompute) read a plan instead of being skipped for lack of one. The shape
	// is the one the reference resolver produces; only the producer differs.
	raw, marshalErr := json.Marshal(derived)
	if marshalErr != nil {
		receipt.State = intentResolutionRefused
		receipt.FailureCause = "the derived receipt could not be serialized: " + marshalErr.Error()
		return receipt
	}
	receipt.ReceiptJSON = string(raw)
	receipt.ReceiptDigest = digestOfBytes(raw)

	if expectedPlanFingerprint != "" {
		receipt.PlanMatchesExpected = derived.PlanFingerprint == expectedPlanFingerprint
		if !receipt.PlanMatchesExpected {
			receipt.FailureCause = "the plan derived in this process differs from the caller-declared expectation"
		}
	}
	return receipt
}
