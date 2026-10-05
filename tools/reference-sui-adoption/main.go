package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/ir"
	"tei/internal/workspace"
	suihierarchy "tei/tools/reference-sui-adoption/hierarchy"
)

const (
	reportSchemaVersion    = "sui-reference-adoption-stage0/v2"
	sourceRepository       = "hat47x/sui-sensemaking"
	sourceRevision         = "725e9d9c8958e63c18d8edca94f19c4b98db0dde"
	sourceContractBlob     = "5d63cbec446abf0dd8054a5c498f726377c51a9f"
	sourceHandoffBlob      = "9bf17490a8d5936b4b7679370da861439e90a8ab"
	sourceContractTestBlob = "4c91399cbc2da4526171812178ecdf98f9272027"
	sourceHandoffTestBlob  = "e069871a13c932dbb1c11cbed5e08e318f93be8c"
)

type hierarchyDocument = suihierarchy.Document
type hierIsland = suihierarchy.Island
type diagnostic = suihierarchy.Diagnostic
type parentLink = suihierarchy.ParentLink

type mappingSummary struct {
	DataDefinition          string `json:"dataDefinition"`
	ParentRelation          string `json:"parentRelation"`
	RelationTarget          string `json:"relationTarget"`
	RelationCardinal        string `json:"relationCardinality"`
	ParentPresenceSemantics string `json:"parentPresenceSemantics"`
	Lossless                bool   `json:"lossless"`
}

type constraintCoverage struct {
	ID                string `json:"id"`
	SourceConstraint  string `json:"sourceConstraint"`
	TEIRepresentation string `json:"teiRepresentation"`
	Status            string `json:"status"`
	ProbeEnforcement  string `json:"probeEnforcement"`
}

type report struct {
	SchemaVersion          string               `json:"schemaVersion"`
	SourceContract         string               `json:"sourceContract"`
	SourceAuthority        string               `json:"sourceAuthority"`
	SourceRepository       string               `json:"sourceRepository"`
	SourceRevision         string               `json:"sourceRevision"`
	SourceContractBlob     string               `json:"sourceContractBlob"`
	SourceHandoffBlob      string               `json:"sourceHandoffBlob"`
	SourceContractTestBlob string               `json:"sourceContractTestBlob"`
	SourceHandoffTestBlob  string               `json:"sourceHandoffTestBlob"`
	TEIAuthority           string               `json:"teiAuthority"`
	Fixture                string               `json:"fixture"`
	FixtureValid           bool                 `json:"fixtureValid"`
	InstanceCount          int                  `json:"instanceCount"`
	RootIslandIDs          []string             `json:"rootIslandIds,omitempty"`
	ParentLinks            []parentLink         `json:"parentLinks,omitempty"`
	Diagnostics            []diagnostic         `json:"diagnostics,omitempty"`
	Mapping                mappingSummary       `json:"mapping"`
	ConstraintCoverage     []constraintCoverage `json:"constraintCoverage"`
	WriteAttempted         bool                 `json:"writeAttempted"`
	CanonicalizedData      bool                 `json:"canonicalizedInstanceData"`
}

func main() {
	workspacePath := flag.String("workspace", "reference/sui-adoption/workspace", "TEI Stage 0 Reference Workspace")
	fixturePath := flag.String("fixture", "reference/sui-adoption/M2-three-level.valid.json", "SUI IslandHierarchyContractV1 fixture")
	expect := flag.String("expect", "valid", "expected fixture result: valid|invalid")
	flag.Parse()

	result, err := run(*workspacePath, *fixturePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	switch *expect {
	case "valid":
		if !result.FixtureValid {
			fmt.Fprintln(os.Stderr, "fixture was expected to be valid")
			os.Exit(1)
		}
	case "invalid":
		if result.FixtureValid {
			fmt.Fprintln(os.Stderr, "fixture was expected to be invalid")
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "-expect must be valid or invalid")
		os.Exit(2)
	}
}

func run(workspacePath, fixturePath string) (report, error) {
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		return report{}, fmt.Errorf("TEI SUI Reference Workspaceを読み込めません: %w", err)
	}

	mapping, err := resolveStage0Mapping(loaded.World)
	if err != nil {
		return report{}, err
	}

	doc, err := readHierarchyFixture(fixturePath)
	if err != nil {
		return report{}, err
	}

	diagnostics := validateHierarchy(doc)
	roots, links := projectInstanceSummary(doc)

	return report{
		SchemaVersion:          reportSchemaVersion,
		SourceContract:         "IslandHierarchyContractV1",
		SourceAuthority:        "sui-sensemaking",
		SourceRepository:       sourceRepository,
		SourceRevision:         sourceRevision,
		SourceContractBlob:     sourceContractBlob,
		SourceHandoffBlob:      sourceHandoffBlob,
		SourceContractTestBlob: sourceContractTestBlob,
		SourceHandoffTestBlob:  sourceHandoffTestBlob,
		TEIAuthority:           "definition-only",
		Fixture:                fixturePath,
		FixtureValid:           len(diagnostics) == 0,
		InstanceCount:          len(doc.Islands),
		RootIslandIDs:          roots,
		ParentLinks:            links,
		Diagnostics:            diagnostics,
		Mapping:                mapping,
		ConstraintCoverage:     stage0ConstraintCoverage(),
		WriteAttempted:         false,
		CanonicalizedData:      false,
	}, nil
}

func readHierarchyFixture(path string) (hierarchyDocument, error) {
	return suihierarchy.ReadFile(path)
}

func validateHierarchy(doc hierarchyDocument) []diagnostic {
	return suihierarchy.Validate(doc)
}

func projectInstanceSummary(doc hierarchyDocument) ([]string, []parentLink) {
	return suihierarchy.ProjectInstanceSummary(doc)
}

func stage0ConstraintCoverage() []constraintCoverage {
	return []constraintCoverage{
		{
			ID:                "parent-single-valued",
			SourceConstraint:  "parentIslandId is a single scalar reference",
			TEIRepresentation: "relation cardinality one",
			Status:            "shape-only",
			ProbeEnforcement:  "tei-definition",
		},
		{
			ID:                "parent-presence",
			SourceConstraint:  "parentIslandId may be null for root islands",
			TEIRepresentation: "none",
			Status:            "unresolved",
			ProbeEnforcement:  "source-contract",
		},
		{
			ID:                "duplicate-island-id",
			SourceConstraint:  "duplicate island id is invalid",
			TEIRepresentation: "none",
			Status:            "shadow-only",
			ProbeEnforcement:  "reference-local-shadow",
		},
		{
			ID:                "duplicate-child-reference",
			SourceConstraint:  "duplicate child reference is invalid",
			TEIRepresentation: "none",
			Status:            "shadow-only",
			ProbeEnforcement:  "reference-local-shadow",
		},
		{
			ID:                "parent-exists",
			SourceConstraint:  "non-null parent must reference an existing island",
			TEIRepresentation: "none",
			Status:            "shadow-only",
			ProbeEnforcement:  "reference-local-shadow",
		},
		{
			ID:                "child-exists",
			SourceConstraint:  "child reference must identify an existing island",
			TEIRepresentation: "none",
			Status:            "shadow-only",
			ProbeEnforcement:  "reference-local-shadow",
		},
		{
			ID:                "parent-child-consistency",
			SourceConstraint:  "parent and child references must agree",
			TEIRepresentation: "none",
			Status:            "shadow-only",
			ProbeEnforcement:  "reference-local-shadow",
		},
		{
			ID:                "acyclic-hierarchy",
			SourceConstraint:  "island hierarchy must be acyclic",
			TEIRepresentation: "none",
			Status:            "shadow-only",
			ProbeEnforcement:  "reference-local-shadow",
		},
	}
}

func resolveStage0Mapping(world ir.World) (mappingSummary, error) {
	var dataCount int
	var relationCount int
	var result mappingSummary

	for _, ns := range world.Semantics {
		for _, data := range ns.Data {
			if string(data.ID) != "sui.island" {
				continue
			}
			dataCount++
			result.DataDefinition = string(data.ID)
			for _, relation := range data.Relations {
				if string(relation.ID) != "sui.island.parent" {
					continue
				}
				relationCount++
				result.ParentRelation = string(relation.ID)
				result.RelationTarget = string(relation.Target)
				result.RelationCardinal = string(relation.Cardinality)
			}
		}
	}

	if dataCount != 1 {
		return mappingSummary{}, fmt.Errorf("TEI Stage 0 mapping requires exactly one sui.island definition; got %d", dataCount)
	}
	if relationCount != 1 {
		return mappingSummary{}, fmt.Errorf("TEI Stage 0 mapping requires exactly one sui.island.parent relation; got %d", relationCount)
	}
	if result.RelationTarget != "sui.island" {
		return mappingSummary{}, fmt.Errorf("sui.island.parent must target sui.island; got %s", result.RelationTarget)
	}
	if result.RelationCardinal != "one" {
		return mappingSummary{}, fmt.Errorf("sui.island.parent must have cardinality one; got %s", result.RelationCardinal)
	}

	// SUI parentIslandId is nullable for root islands. Current TEI Relation only
	// exposes one|many cardinality, so Stage 0 has not proven the 0..1 presence
	// semantics lossless. Keep this unresolved instead of inventing optionality.
	result.ParentPresenceSemantics = "unresolved"
	result.Lossless = false

	return result, nil
}
