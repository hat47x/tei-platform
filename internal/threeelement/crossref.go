package threeelement

import (
	"sort"
)

// BusinessDataCrossReference describes Business→Data mappings derived through functions.
type BusinessDataCrossReference struct {
	BusinessID   string `json:"businessId"`
	BusinessName string `json:"businessName"`
	ActivityID   string `json:"activityId"`
	ActivityName string `json:"activityName"`
	FunctionID   string `json:"functionId"`
	FunctionName string `json:"functionName"`
	DataID       string `json:"dataId"`
	DataName     string `json:"dataName"`
	Operations   string `json:"operations,omitempty"`
}

// GenerateBusinessDataCrossReference derives Business→Data mappings via Function chain.
// Operation flags resolve the same way as GenerateFunctionDataCrossReference, so
// the two sections of one cross-reference result cannot disagree about the same
// Function/Data pair.
func GenerateBusinessDataCrossReference(model ThreeElementModel) []BusinessDataCrossReference {
	dataByID, fnByID, _ := model.Index()
	derived := derivedOperationsByPair(model)

	type key struct {
		bus, act, fn, data string
	}
	seen := map[key]bool{}
	var out []BusinessDataCrossReference

	for _, bus := range model.BusinessElements {
		for _, act := range bus.Activities {
			for _, fid := range act.FunctionIDs {
				fn := fnByID[fid]
				for _, op := range fn.Operations {
					k := key{bus.ID, act.ID, fn.ID, op.DataID}
					if seen[k] {
						continue
					}
					seen[k] = true
					// An explicit declaration takes precedence as a projection
					// convention; it must not be reported under a different
					// operation set here than in the Function/Data section.
					operations := derived[[2]string{fn.ID, op.DataID}]
					if crud, ok := explicitCRUDFor(model, fn.ID, op.DataID); ok {
						operations = operationsForCRUD(crud)
					}
					out = append(out, BusinessDataCrossReference{
						BusinessID:   bus.ID,
						BusinessName: bus.Name,
						ActivityID:   act.ID,
						ActivityName: act.Name,
						FunctionID:   fn.ID,
						FunctionName: fn.Name,
						DataID:       op.DataID,
						DataName:     dataByID[op.DataID].Name,
						Operations:   operations,
					})
				}
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
		if out[i].FunctionID != out[j].FunctionID {
			return out[i].FunctionID < out[j].FunctionID
		}
		return out[i].DataID < out[j].DataID
	})
	return out
}

// DataDataCrossReference describes Data→Data relations.
//
// RelationID is the DataRelation identity inside the DataElement named by
// SourceID. Two relations between the same pair of Data elements are different
// declarations, and a projection that drops the identity cannot say which row
// is which (decision section 33.6 SDM-01). The pair SourceID+RelationID is the
// qualified identity; a RelationID alone is not, because each Data element owns
// its own relation ids.
type DataDataCrossReference struct {
	SourceID   string `json:"sourceId"`
	SourceName string `json:"sourceName"`
	RelationID string `json:"relationId"`
	TargetID   string `json:"targetId"`
	TargetName string `json:"targetName"`
	Kind       string `json:"kind"`
	Role       string `json:"role,omitempty"`
	Directed   bool   `json:"directed,omitempty"`
}

// GenerateDataDataCrossReference derives Data→Data relations.
//
// The order is total and independent of input order: source, target, then the
// relation identity. Without the last key two parallel relations that agree on
// every other projected field would come out in whatever order the sort
// happened to produce, so a Consumer comparing two projections of the same
// model could see a difference that is not a difference in the model.
func GenerateDataDataCrossReference(model ThreeElementModel) []DataDataCrossReference {
	dataByID, _, _ := model.Index()
	var out []DataDataCrossReference
	for _, d := range model.DataElements {
		for _, rel := range d.Relations {
			out = append(out, DataDataCrossReference{
				SourceID:   d.ID,
				SourceName: d.Name,
				RelationID: rel.ID,
				TargetID:   rel.TargetID,
				TargetName: dataByID[rel.TargetID].Name,
				Kind:       rel.Kind,
				Role:       rel.Role,
				Directed:   rel.Directed,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SourceID != out[j].SourceID {
			return out[i].SourceID < out[j].SourceID
		}
		if out[i].TargetID != out[j].TargetID {
			return out[i].TargetID < out[j].TargetID
		}
		return out[i].RelationID < out[j].RelationID
	})
	return out
}
