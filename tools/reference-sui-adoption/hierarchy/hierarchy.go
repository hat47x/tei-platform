package hierarchy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
)

type Document struct {
	SchemaVersion string   `json:"schemaVersion"`
	Islands       []Island `json:"islands"`
}

type Island struct {
	ID             string   `json:"id"`
	ParentIslandID *string  `json:"parentIslandId"`
	ChildIslandIDs []string `json:"childIslandIds"`
}

type Diagnostic struct {
	Code    string `json:"code"`
	Island  string `json:"island,omitempty"`
	Message string `json:"message"`
}

type ParentLink struct {
	Child  string `json:"child"`
	Parent string `json:"parent"`
}

func ReadFile(path string) (Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Document{}, fmt.Errorf("SUI hierarchy fixture %sを読み込めません: %w", path, err)
	}
	doc, err := Decode(content)
	if err != nil {
		return Document{}, fmt.Errorf("SUI hierarchy fixture %sが不正です: %w", path, err)
	}
	return doc, nil
}

func Decode(content []byte) (Document, error) {
	var doc Document
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return Document{}, err
	}
	if err := requireEOF(decoder); err != nil {
		return Document{}, err
	}
	if doc.SchemaVersion == "" {
		return Document{}, fmt.Errorf("schemaVersion is required")
	}
	return doc, nil
}

func Validate(doc Document) []Diagnostic {
	var diagnostics []Diagnostic
	islandByID := map[string]Island{}
	duplicateID := false

	for _, island := range doc.Islands {
		if _, exists := islandByID[island.ID]; exists {
			duplicateID = true
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "SUI_HIERARCHY_DUPLICATE_ID",
				Island:  island.ID,
				Message: "duplicate island id",
			})
			continue
		}

		childSet := map[string]struct{}{}
		for _, childID := range island.ChildIslandIDs {
			if _, exists := childSet[childID]; exists {
				diagnostics = append(diagnostics, Diagnostic{
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
				diagnostics = append(diagnostics, Diagnostic{
					Code:    "SUI_HIERARCHY_PARENT_MISSING",
					Island:  island.ID,
					Message: fmt.Sprintf("missing parent island %s", parentID),
				})
			}
		}

		for _, childID := range island.ChildIslandIDs {
			child, exists := islandByID[childID]
			if !exists {
				diagnostics = append(diagnostics, Diagnostic{
					Code:    "SUI_HIERARCHY_CHILD_MISSING",
					Island:  island.ID,
					Message: fmt.Sprintf("missing child island %s", childID),
				})
				continue
			}
			if child.ParentIslandID == nil || *child.ParentIslandID != island.ID {
				diagnostics = append(diagnostics, Diagnostic{
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
				diagnostics = append(diagnostics, Diagnostic{
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

func ProjectInstanceSummary(doc Document) ([]string, []ParentLink) {
	var roots []string
	var links []ParentLink

	for _, island := range doc.Islands {
		if island.ParentIslandID == nil {
			roots = append(roots, island.ID)
			continue
		}
		links = append(links, ParentLink{Child: island.ID, Parent: *island.ParentIslandID})
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

func requireEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("JSON documentの後ろに追加valueがあります")
}
