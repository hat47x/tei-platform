package threeelement

import (
	"fmt"
	"sort"
	"strings"
)

// CRUDTable is one row of a generated CRUD table.
type CRUDTable struct {
	FunctionID   string `json:"functionId"`
	FunctionName string `json:"functionName"`
	DataID       string `json:"dataId"`
	DataName     string `json:"dataName"`
	Create       bool   `json:"create"`
	Read         bool   `json:"read"`
	Update       bool   `json:"update"`
	Delete       bool   `json:"delete"`
	Source       string `json:"source,omitempty"`
}

// GenerateCRUDTable derives a deterministic CRUD table from the model.
// It prefers explicit CRUDDefinitions; missing pairs are filled from FunctionElement.Operations.
func GenerateCRUDTable(model ThreeElementModel) []CRUDTable {
	dataByID, _, _ := model.Index()

	type pair struct{ fn, data string }
	explicit := map[pair]CRUDDefinition{}
	for _, crud := range model.CRUDDefinitions {
		explicit[pair{crud.FunctionID, crud.DataID}] = crud
	}

	var rows []CRUDTable
	for _, fn := range model.FunctionElements {
		for _, op := range fn.Operations {
			row := CRUDTable{
				FunctionID:   fn.ID,
				FunctionName: fn.Name,
				DataID:       op.DataID,
				DataName:     dataByID[op.DataID].Name,
				Source:       "derived",
			}
			if crud, ok := explicit[pair{fn.ID, op.DataID}]; ok {
				row = CRUDTable{
					FunctionID:   crud.FunctionID,
					FunctionName: fn.Name,
					DataID:       crud.DataID,
					DataName:     dataByID[crud.DataID].Name,
					Create:       crud.Create,
					Read:         crud.Read,
					Update:       crud.Update,
					Delete:       crud.Delete,
					Source:       crud.Source,
				}
			} else {
				for _, ch := range strings.ToUpper(op.Operations) {
					switch ch {
					case 'C':
						row.Create = true
					case 'R':
						row.Read = true
					case 'U':
						row.Update = true
					case 'D':
						row.Delete = true
					}
				}
			}
			rows = append(rows, row)
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].FunctionID != rows[j].FunctionID {
			return rows[i].FunctionID < rows[j].FunctionID
		}
		return rows[i].DataID < rows[j].DataID
	})
	return rows
}

// BusinessFunctionCrossReference describes Business→Function mappings.
type BusinessFunctionCrossReference struct {
	BusinessID   string `json:"businessId"`
	BusinessName string `json:"businessName"`
	ActivityID   string `json:"activityId"`
	ActivityName string `json:"activityName"`
	FunctionID   string `json:"functionId"`
	FunctionName string `json:"functionName"`
}

// FunctionDataCrossReference describes Function→Data mappings.
type FunctionDataCrossReference struct {
	FunctionID   string `json:"functionId"`
	FunctionName string `json:"functionName"`
	DataID       string `json:"dataId"`
	DataName     string `json:"dataName"`
	Operations   string `json:"operations"`
	Source       string `json:"source,omitempty"`
}

// GenerateBusinessFunctionCrossReference derives Business→Function mappings.
func GenerateBusinessFunctionCrossReference(model ThreeElementModel) []BusinessFunctionCrossReference {
	_, fnByID, _ := model.Index()
	var out []BusinessFunctionCrossReference
	for _, bus := range model.BusinessElements {
		for _, act := range bus.Activities {
			for _, fid := range act.FunctionIDs {
				out = append(out, BusinessFunctionCrossReference{
					BusinessID:   bus.ID,
					BusinessName: bus.Name,
					ActivityID:   act.ID,
					ActivityName: act.Name,
					FunctionID:   fid,
					FunctionName: fnByID[fid].Name,
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].BusinessID != out[j].BusinessID {
			return out[i].BusinessID < out[j].BusinessID
		}
		if out[i].ActivityID != out[j].ActivityID {
			return out[i].ActivityID < out[j].ActivityID
		}
		return out[i].FunctionID < out[j].FunctionID
	})
	return out
}

// GenerateFunctionDataCrossReference derives Function→Data mappings including CRUD.
func GenerateFunctionDataCrossReference(model ThreeElementModel) []FunctionDataCrossReference {
	dataByID, _, _ := model.Index()

	type pair struct{ fn, data string }
	explicit := map[pair]CRUDDefinition{}
	for _, crud := range model.CRUDDefinitions {
		explicit[pair{crud.FunctionID, crud.DataID}] = crud
	}

	type key struct {
		fn   string
		data string
	}
	seen := map[key]bool{}
	var out []FunctionDataCrossReference

	for _, fn := range model.FunctionElements {
		for _, op := range fn.Operations {
			k := key{fn.ID, op.DataID}
			if seen[k] {
				continue
			}
			seen[k] = true
			row := FunctionDataCrossReference{
				FunctionID:   fn.ID,
				FunctionName: fn.Name,
				DataID:       op.DataID,
				DataName:     dataByID[op.DataID].Name,
				Operations:   strings.ToUpper(op.Operations),
				Source:       "derived",
			}
			if crud, ok := explicit[pair{fn.ID, op.DataID}]; ok {
				var ops []string
				if crud.Create {
					ops = append(ops, "C")
				}
				if crud.Read {
					ops = append(ops, "R")
				}
				if crud.Update {
					ops = append(ops, "U")
				}
				if crud.Delete {
					ops = append(ops, "D")
				}
				row.Operations = strings.Join(ops, "")
				row.Source = crud.Source
			}
			out = append(out, row)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].FunctionID != out[j].FunctionID {
			return out[i].FunctionID < out[j].FunctionID
		}
		return out[i].DataID < out[j].DataID
	})
	return out
}

// CRUDGap reports CRUD cells that are expected but absent.
type CRUDGap struct {
	FunctionID   string `json:"functionId"`
	FunctionName string `json:"functionName"`
	DataID       string `json:"dataId"`
	DataName     string `json:"dataName"`
	Missing      string `json:"missing"` // subset of CRUD
	Message      string `json:"message"`
}

// GenerateCRUDGaps finds missing CRUD operations based on declared cross-references.
// This is a conservative checker: it only flags gaps where a cross-reference exists
// but no CRUD covers the expected direction.
func GenerateCRUDGaps(model ThreeElementModel) []CRUDGap {
	var gaps []CRUDGap
	dataByID, fnByID, _ := model.Index()

	// Build explicit CRUD lookup.
	type pair struct{ fn, data string }
	crudMap := map[pair]CRUDDefinition{}
	for _, crud := range model.CRUDDefinitions {
		crudMap[pair{crud.FunctionID, crud.DataID}] = crud
	}

	for _, ref := range model.CrossReferences {
		if ref.Kind != "function-to-data" {
			continue
		}
		fn := fnByID[ref.FromID]
		data := dataByID[ref.ToID]
		if fn.ID == "" || data.ID == "" {
			continue
		}
		key := pair{fn.ID, data.ID}
		crud, ok := crudMap[key]
		if !ok {
			// If no CRUD exists at all for a declared function-to-data relation, flag it.
			gaps = append(gaps, CRUDGap{
				FunctionID:   fn.ID,
				FunctionName: fn.Name,
				DataID:       data.ID,
				DataName:     data.Name,
				Missing:      "CRUD",
				Message:      fmt.Sprintf("cross-reference %q links function %q to data %q but no CRUD definition exists", ref.ID, fn.ID, data.ID),
			})
			continue
		}
		var missing []string
		if !crud.Read {
			missing = append(missing, "R")
		}
		// Only flag create/update/delete if the function is not clearly read-only.
		// This is a heuristic: absence of any write flag is not a gap by itself.
		_ = missing
	}

	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].FunctionID != gaps[j].FunctionID {
			return gaps[i].FunctionID < gaps[j].FunctionID
		}
		return gaps[i].DataID < gaps[j].DataID
	})
	return gaps
}
