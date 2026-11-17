package threeelement

import "sort"

// CRUDFactOccurrence identifies one declaration inside a particular
// ThreeElementModel snapshot. Positions are 1-based, model-local coordinates,
// not durable subject IDs, Source locators, or proof of accepted meaning.
type CRUDFactOccurrence struct {
	Kind               string `json:"kind"` // function-operation | crud-definition
	FunctionPosition   int    `json:"functionPosition,omitempty"`
	OperationPosition  int    `json:"operationPosition,omitempty"`
	DefinitionPosition int    `json:"definitionPosition,omitempty"`
	DefinitionID       string `json:"definitionId,omitempty"`
	Operations         string `json:"operations"`
	SourceLabel        string `json:"sourceLabel,omitempty"`
}

// CRUDCellOrigin gives reviewers the original declarations underlying a
// Function/Data cell. It does not replace or authorize GenerateCRUDTable.
type CRUDCellOrigin struct {
	FunctionID string               `json:"functionId"`
	DataID     string               `json:"dataId"`
	Facts      []CRUDFactOccurrence `json:"facts"`
}

// GenerateCRUDCellOrigins retains each FunctionOperation and CRUDDefinition
// behind a single Function/Data display cell. It never normalizes, drops, or
// resolves conflicting declarations. A fact's position is valid only against
// the input model revision; an editor must separately qualify the current
// Source and the exact target before attempting any change.
func GenerateCRUDCellOrigins(model ThreeElementModel) []CRUDCellOrigin {
	type pair struct{ functionID, dataID string }
	byPair := make(map[pair][]CRUDFactOccurrence)
	for fi, fn := range model.FunctionElements {
		for oi, op := range fn.Operations {
			key := pair{fn.ID, op.DataID}
			byPair[key] = append(byPair[key], CRUDFactOccurrence{
				Kind:              "function-operation",
				FunctionPosition:  fi + 1,
				OperationPosition: oi + 1,
				Operations:        op.Operations,
			})
		}
	}
	for di, crud := range model.CRUDDefinitions {
		key := pair{crud.FunctionID, crud.DataID}
		byPair[key] = append(byPair[key], CRUDFactOccurrence{
			Kind:               "crud-definition",
			DefinitionPosition: di + 1,
			DefinitionID:       crud.ID,
			Operations:         operationsForCRUD(crud),
			SourceLabel:        crud.Source,
		})
	}

	keys := make([]pair, 0, len(byPair))
	for key := range byPair {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].functionID != keys[j].functionID {
			return keys[i].functionID < keys[j].functionID
		}
		return keys[i].dataID < keys[j].dataID
	})

	out := make([]CRUDCellOrigin, 0, len(keys))
	for _, key := range keys {
		out = append(out, CRUDCellOrigin{
			FunctionID: key.functionID,
			DataID:     key.dataID,
			Facts:      byPair[key],
		})
	}
	return out
}
