package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"tei/internal/ir"
	"tei/internal/workspace"
)

const (
	reportSchemaVersion         = "sui-reference-adoption-stage0/v1"
	sourceRepository            = "hat47x/sui-sensemaking"
	sourceRevision              = "725e9d9c8958e63c18d8edca94f19c4b98db0dde"
	sourceContractBlob          = "5d63cbec446abf0dd8054a5c498f726377c51a9f"
	sourceHandoffBlob           = "9bf17490a8d5936b4b7679370da861439e90a8ab"
	sourceContractTestBlob      = "4c91399cbc2da4526171812178ecdf98f9272027"
	sourceHandoffTestBlob       = "e069871a13c932dbb1c11cbed5e08e318f93be8c"
)

type hierarchyDocument struct {
	SchemaVersion string       `json:"schemaVersion"`
	Islands       []hierIsland `json:"islands"`
}

type hierIsland struct {
	ID             string   `json:"id"`
	ParentIslandID *string  `json:"parentIslandId"`
	ChildIslandIDs []string `json:"childIslandIds"`
}

type diagnostic struct {
	Code    string `json:"code"`
	Island  string `json:"island,omitempty"`
	Message string `json:"message"`
}

type parentLink struct {
	Child  string `json:"child"`
	Parent string `json:"parent"`
}

type mappingSummary struct {
	DataDefinition          string `json:"dataDefinition"`
	ParentRelation          string `json:"parentRelation"`
	RelationTarget          string `json:"relationTarget"`
	RelationCardinal        string `json:"relationCardinality"`
	ParentPresenceSemantics string `json:"parentPresenceSemantics"`
	Lossless                bool   `json:"lossless"`
}

type report struct {
	SchemaVersion          string         `json:"schemaVersion"`
	SourceContract         string         `json:"sourceContract"`
	SourceAuthority        string         `json:"sourceAuthority"`
	SourceRepository       string         `json:"sourceRepository"`
	SourceRevision         string         `json:"sourceRevision"`
	SourceContractBlob     string         `json:"sourceContractBlob"`
	SourceHandoffBlob      string         `json:"sourceHandoffBlob"`
	SourceContractTestBlob string         `json:"sourceContractTestBlob"`
	SourceHandoffTestBlob  string         `json:"sourceHandoffTestBlob"`
	TEIAuthority      string         `json:"teiAuthority"`
	Fixture           string         `json:"fixture"`
	FixtureValid      bool           `json:"fixtureValid"`
	InstanceCount     int            `json:"instanceCount"`
	RootIslandIDs     []string       `json:"rootIslandIds,omitempty"`
	ParentLinks       []parentLink   `json:"parentLinks,omitempty"`
	Diagnostics       []diagnostic   `json:"diagnostics,omitempty"`
	Mapping           mappingSummary `json:"mapping"`
	WriteAttempted    bool           `json:"writeAttempted"`
	CanonicalizedData bool           `json:"canonicalizedInstanceData"`
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
		Fixture:           fixturePath,
		FixtureValid:      len(diagnostics) == 0,
		InstanceCount:     len(doc.Islands),
		RootIslandIDs:     roots,
		ParentLinks:       links,
		Diagnostics:       diagnostics,
		Mapping:           mapping,
		WriteAttempted:    false,
		CanonicalizedData: false,
	}, nil
}

func readHierarchyFixture(path string) (hierarchyDocument, error) {
	file, err := os.Open(path)
	if err != nil {
		return hierarchyDocument{}, fmt.Errorf("SUI hierarchy fixture %sを読み込めません: %w", path, err)
	}
	defer file.Close()

	var doc hierarchyDocument
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return hierarchyDocument{}, fmt.Errorf("SUI hierarchy fixture %sが不正です: %w", path, err)
	}
	if err := requireEOF(decoder); err != nil {
		return hierarchyDocument{}, fmt.Errorf("SUI hierarchy fixture %sが不正です: %w", path, err)
	}
	if doc.SchemaVersion == "" {
		return hierarchyDocument{}, fmt.Errorf("SUI hierarchy fixture %s: schemaVersion is required", path)
	}
	return doc, nil
}

func requireEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("JSON documentの後ろに追加valueがあります")
}

func validateHierarchy(doc hierarchyDocument) []diagnostic {
	var diagnostics []diagnostic
	islandByID := map[string]hierIsland{}
	duplicateID := false

	for _, island := range doc.Islands {
		if _, exists := islandByID[island.ID]; exists {
			duplicateID = true
			diagnostics = append(diagnostics, diagnostic{
				Code:    "SUI_HIERARCHY_DUPLICATE_ID",
				Island:  island.ID,
				Message: "duplicate island id",
			})
			continue
		}

		childSet := map[string]struct{}{}
		for _, childID := range island.ChildIslandIDs {
			if _, exists := childSet[childID]; exists {
				diagnostics = append(diagnostics, diagnostic{
					Code:    "SUI_HIERARCHY_DUPLICATE_CHILD",
					Island:  island.ID,
					Message: fmt.Sprintf("duplicate child island %s", childID),
				})
			}
			childSet[childID] = struct{}{}
		}
		islandByID[island.ID] = island
	}

	for _, island := range doc.Islands {
		if island.ParentIslandID != nil {
			parentID := *island.ParentIslandID
			if _, exists := islandByID[parentID]; !exists {
				diagnostics = append(diagnostics, diagnostic{
					Code:    "SUI_HIERARCHY_PARENT_MISSING",
					Island:  island.ID,
					Message: fmt.Sprintf("missing parent island %s", parentID),
				})
			}
		}

		for _, childID := range island.ChildIslandIDs {
			child, exists := islandByID[childID]
			if !exists {
				diagnostics = append(diagnostics, diagnostic{
					Code:    "SUI_HIERARCHY_CHILD_MISSING",
					Island:  island.ID,
					Message: fmt.Sprintf("missing child island %s", childID),
				})
				continue
			}
			if child.ParentIslandID == nil || *child.ParentIslandID != island.ID {
				diagnostics = append(diagnostics, diagnostic{
					Code:    "SUI_HIERARCHY_PARENT_CHILD_MISMATCH",
					Island:  island.ID,
					Message: fmt.Sprintf("parent/child link mismatch: %s -> %s", island.ID, childID),
				})
			}
		}
	}

	if !duplicateID {
		state := map[string]uint8{}
		var visit func(string) bool
		visit = func(id string) bool {
			switch state[id] {
			case 1:
				return true
			case 2:
				return false
			}
			state[id] = 1
			island, ok := islandByID[id]
			if ok {
				for _, childID := range island.ChildIslandIDs {
					if _, exists := islandByID[childID]; exists && visit(childID) {
						return true
					}
				}
			}
			state[id] = 2
			return false
		}

		for id := range islandByID {
			if visit(id) {
				diagnostics = append(diagnostics, diagnostic{
					Code:    "SUI_HIERARCHY_CYCLE",
					Island:  id,
					Message: "cycle detected",
				})
				break
			}
		}
	}

	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].Code != diagnostics[j].Code {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		if diagnostics[i].Island != diagnostics[j].Island {
			return diagnostics[i].Island < diagnostics[j].Island
		}
		return diagnostics[i].Message < diagnostics[j].Message
	})
	return diagnostics
}

func projectInstanceSummary(doc hierarchyDocument) ([]string, []parentLink) {
	var roots []string
	var links []parentLink

	for _, island := range doc.Islands {
		if island.ParentIslandID == nil {
			roots = append(roots, island.ID)
			continue
		}
		links = append(links, parentLink{Child: island.ID, Parent: *island.ParentIslandID})
	}

	sort.Strings(roots)
	sort.Slice(links, func(i, j int) bool {
		if links[i].Parent != links[j].Parent {
			return links[i].Parent < links[j].Parent
		}
		return links[i].Child < links[j].Child
	})
	return roots, links
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
