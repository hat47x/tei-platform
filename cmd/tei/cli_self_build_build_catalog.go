package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/buildintent"
)

// This file adds the in-process resolution route that SB-C0 §4.22 chose as
// stage 2: the entry derives the plan itself from a caller-named build catalog,
// instead of starting a resolver child and reading its receipt.
//
// What this changes and what it does not:
//
//	changes     the entry no longer needs a separate resolver executable to
//	            obtain a plan; it reads the declaration, reads the catalog, and
//	            derives the plan with the same package the reference resolver
//	            uses
//	does not    it does not create an Authority. The catalog is named by the
//	            caller and its digest is required, in the same shape as -supply
//	            and -intent-resolver. The entry does not discover builders, and
//	            it does not claim the catalog is correct.
//
// The catalog's provenance is K-3 from §4.22: the caller names the data and the
// digest. Requiring both is what makes "which catalog did this use" recordable;
// naming only the path would leave the same ambiguity that
// planReceipt.resolverDigest avoids for a resolver child.

const (
	selfBuildCodeBuildCatalogRefused = "SELF_BUILD_BUILD_CATALOG_REFUSED"

	// The authority a receipt produced in-process declares. It is deliberately
	// distinct from "derived-cache", which is what a resolver child prints, so a
	// reader can tell which route produced the plan.
	inProcessAuthority = "derived-cache"
)

// buildCatalogResolution is what the entry learned by deriving the plan itself.
// Every field is either read from the catalog by this process or derived by this
// process; nothing is copied from a child's report.
type buildCatalogResolution struct {
	State         string `json:"state"`
	CatalogPath   string `json:"catalogPath,omitempty"`
	CatalogDigest string `json:"catalogDigest,omitempty"`
	// DeclaredCatalogDigest is what the caller said the catalog's digest is. It
	// must equal CatalogDigest, which is what this entry measured.
	DeclaredCatalogDigest string `json:"declaredCatalogDigest,omitempty"`
	IntentFingerprint     string `json:"intentFingerprint,omitempty"`
	PlanFingerprint       string `json:"planFingerprint,omitempty"`
	SourceClosure         string `json:"sourceClosureFingerprint,omitempty"`
	Authority             string `json:"authority,omitempty"`
	// ReceiptJSON is the receipt this entry produced, in the same shape a
	// resolver child prints. Keeping the same shape means the rest of the entry
	// does not need to know which route produced it.
	ReceiptJSON    string `json:"-"`
	FailureCause   string `json:"failureCause,omitempty"`
	L2BClaimed     bool   `json:"l2bClaimed"`
	RuntimeClaimed bool   `json:"runtimeAuthorityClaimed"`
	Boundary       string `json:"boundary"`
}

const buildCatalogBoundary = "this entry read the declaration and the catalog and derived the plan itself; that is the entry's own interpretation, not acceptance of the declaration's meaning, not runtime authority, and not a claim that the catalog is correct"

// resolveInProcess derives a plan from a caller-named catalog. It mirrors the
// refusals a resolver child makes, so a route change does not weaken them.
func resolveInProcess(intentPath, catalogPath, declaredCatalogDigest, sourceRoot, expectedPlanFingerprint string) buildCatalogResolution {
	record := buildCatalogResolution{
		State:                 intentResolutionRefused,
		CatalogPath:           catalogPath,
		DeclaredCatalogDigest: declaredCatalogDigest,
		Authority:             inProcessAuthority,
		L2BClaimed:            false,
		RuntimeClaimed:        false,
		Boundary:              buildCatalogBoundary,
	}
	if strings.TrimSpace(catalogPath) == "" {
		record.FailureCause = "no build catalog was named, so the entry has nothing to derive a plan from"
		return record
	}

	// The catalog's digest is measured here, not taken on trust. A caller whose
	// declared digest disagrees is refused: the pair exists so that "which
	// catalog" is answerable later.
	catalogRaw, err := os.ReadFile(catalogPath)
	if err != nil {
		record.FailureCause = "build catalog is not readable: " + err.Error()
		return record
	}
	record.CatalogDigest = digestOfBytes(catalogRaw)
	if declaredCatalogDigest == "" {
		record.FailureCause = "the build catalog's digest was not declared; a catalog is accepted only together with its digest"
		return record
	}
	if declaredCatalogDigest != record.CatalogDigest {
		record.FailureCause = fmt.Sprintf("the declared catalog digest %s does not match the catalog's measured digest %s", declaredCatalogDigest, record.CatalogDigest)
		return record
	}

	intentRaw, err := os.ReadFile(intentPath)
	if err != nil {
		record.FailureCause = "build intent is not readable: " + err.Error()
		return record
	}
	intent, err := buildintent.ParseIntent(intentRaw)
	if err != nil {
		record.FailureCause = "declaration: " + err.Error()
		return record
	}
	catalog, err := buildintent.ParseBuildCatalog(catalogRaw)
	if err != nil {
		record.FailureCause = "build catalog: " + err.Error()
		return record
	}

	// The source root is where the declaration's references resolve from. It is
	// a separate flag because a catalog and the sources it names need not sit in
	// the same directory; the catalog's own directory is only the default.
	if strings.TrimSpace(sourceRoot) == "" {
		sourceRoot = filepath.Dir(catalogPath)
	}
	receipt, err := buildintent.Resolve(intent, buildintent.NewBuildContext(sourceRoot, catalog))
	if err != nil {
		record.FailureCause = "resolution: " + err.Error()
		return record
	}
	raw, err := buildintent.MarshalReceipt(receipt)
	if err != nil {
		record.FailureCause = "receipt: " + err.Error()
		return record
	}

	record.State = intentResolutionResolved
	record.IntentFingerprint = receipt.IntentFingerprint
	record.PlanFingerprint = receipt.PlanFingerprint
	record.SourceClosure = receipt.SourceClosureFingerprint
	record.ReceiptJSON = string(raw)
	if expectedPlanFingerprint != "" && receipt.PlanFingerprint != expectedPlanFingerprint {
		record.FailureCause = "the derived plan fingerprint differs from the caller-declared expectation"
	}
	return record
}

// planReceiptFromInProcess converts the entry's own derivation into the receipt
// shape the rest of the entry already consumes, so everything downstream works
// identically on both routes.
func planReceiptFromInProcess(record buildCatalogResolution, expectedPlanFingerprint string) PlanReceipt {
	receipt := PlanReceipt{
		State:                    record.State,
		ResolverPath:             "",
		ResolverDigest:           "",
		ReceiptDigest:            "",
		Authority:                record.Authority,
		IntentFingerprint:        record.IntentFingerprint,
		PlanFingerprint:          record.PlanFingerprint,
		SourceClosureFingerprint: record.SourceClosure,
		ExpectedPlanFingerprint:  expectedPlanFingerprint,
		L2Claimed:                false,
		BindingNote:              planReceiptBindingNote,
		ReceiptJSON:              record.ReceiptJSON,
	}
	if record.State != intentResolutionResolved {
		receipt.FailureCause = record.FailureCause
		return receipt
	}
	if expectedPlanFingerprint != "" {
		receipt.PlanMatchesExpected = record.PlanFingerprint == expectedPlanFingerprint
		if !receipt.PlanMatchesExpected {
			receipt.FailureCause = record.FailureCause
		}
	}
	if record.ReceiptJSON != "" {
		receipt.ReceiptDigest = digestOfBytes([]byte(record.ReceiptJSON))
	}
	return receipt
}

// buildCatalogResolutionJSON is used by tests and by the report encoder.
func buildCatalogResolutionJSON(record buildCatalogResolution) ([]byte, error) {
	return json.MarshalIndent(record, "", "  ")
}
