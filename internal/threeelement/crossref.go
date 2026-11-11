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
func GenerateBusinessDataCrossReference(model ThreeElementModel) []BusinessDataCrossReference {
	_, fnByID, dataByID := model.Index()

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
					out = append(out, BusinessDataCrossReference{
						BusinessID:   bus.ID,
						BusinessName: bus.Name,
						ActivityID:   act.ID,
						ActivityName: act.Name,
						FunctionID:   fn.ID,
						FunctionName: fn.Name,
						DataID:       op.DataID,
						DataName:     dataByID[op.DataID].Name,
						Operations:   op.Operations,
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
type DataDataCrossReference struct {
	SourceID   string `json:"sourceId"`
	SourceName string `json:"sourceName"`
	TargetID   string `json:"targetId"`
	TargetName string `json:"targetName"`
	Kind       string `json:"kind"`
	Role       string `json:"role,omitempty"`
	Directed   bool   `json:"directed,omitempty"`
}

// GenerateDataDataCrossReference derives Data→Data relations.
func GenerateDataDataCrossReference(model ThreeElementModel) []DataDataCrossReference {
	dataByID, _, _ := model.Index()
	var out []DataDataCrossReference
	for _, d := range model.DataElements {
		for _, rel := range d.Relations {
			out = append(out, DataDataCrossReference{
				SourceID:   d.ID,
				SourceName: d.Name,
				TargetID:   rel.TargetID,
				TargetName: dataByID[rel.TargetID].Name,
				Kind:       rel.Kind,
				Role:       rel.Role,
				Directed:   rel.Directed,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SourceID != out[j].SourceID {
			return out[i].SourceID < out[j].SourceID
		}
		return out[i].TargetID < out[j].TargetID
	})
	return out
}
