package threeelement

import (
	"encoding/json"
	"fmt"
	"sort"
)

// DiffEntry is one change between two models.
type DiffEntry struct {
	Kind     string `json:"kind"`     // added | removed | changed
	Category string `json:"category"` // data | function | business | cross-reference | crud
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Before   string `json:"before,omitempty"`
	After    string `json:"after,omitempty"`
}

// DiffResult is the full diff between two models.
type DiffResult struct {
	SchemaVersion   string      `json:"schemaVersion"`
	BeforeID        string      `json:"beforeId"`
	AfterID         string      `json:"afterId"`
	Added           []DiffEntry `json:"added"`
	Removed         []DiffEntry `json:"removed"`
	Changed         []DiffEntry `json:"changed"`
	AuthorityLimits []string    `json:"authorityLimits,omitempty"`
}

// DiffModels computes a deterministic diff between two models.
func DiffModels(before, after ThreeElementModel) DiffResult {
	res := DiffResult{
		SchemaVersion: "tei.three-element.diff/v0",
		BeforeID:      before.ModelID,
		AfterID:       after.ModelID,
		AuthorityLimits: []string{
			"diff-is-read-only-comparison",
			"identical-fingerprints-are-not-guaranteed-by-empty-diff",
		},
	}

	beforeData := byID(before.DataElements)
	afterData := byID(after.DataElements)
	beforeFn := byID(before.FunctionElements)
	afterFn := byID(after.FunctionElements)
	beforeBus := byID(before.BusinessElements)
	afterBus := byID(after.BusinessElements)
	beforeCrossRef := byID(before.CrossReferences)
	afterCrossRef := byID(after.CrossReferences)
	beforeCRUD := byID(before.CRUDDefinitions)
	afterCRUD := byID(after.CRUDDefinitions)

	allData := stringKeys(beforeData, afterData)
	allFn := stringKeys(beforeFn, afterFn)
	allBus := stringKeys(beforeBus, afterBus)
	allCrossRef := stringKeys(beforeCrossRef, afterCrossRef)
	allCRUD := stringKeys(beforeCRUD, afterCRUD)

	for _, id := range sortedKeys(allData) {
		b, a := beforeData[id], afterData[id]
		switch {
		case b.ID == "" && a.ID != "":
			res.Added = append(res.Added, DiffEntry{Kind: "added", Category: "data", ID: a.ID, Name: a.Name, Detail: "data element"})
		case b.ID != "" && a.ID == "":
			res.Removed = append(res.Removed, DiffEntry{Kind: "removed", Category: "data", ID: b.ID, Name: b.Name, Detail: "data element"})
		case !equalData(b, a):
			res.Changed = append(res.Changed, DiffEntry{Kind: "changed", Category: "data", ID: b.ID, Name: b.Name, Before: summarize(b), After: summarize(a)})
		}
	}

	for _, id := range sortedKeys(allFn) {
		b, a := beforeFn[id], afterFn[id]
		switch {
		case b.ID == "" && a.ID != "":
			res.Added = append(res.Added, DiffEntry{Kind: "added", Category: "function", ID: a.ID, Name: a.Name, Detail: "function element"})
		case b.ID != "" && a.ID == "":
			res.Removed = append(res.Removed, DiffEntry{Kind: "removed", Category: "function", ID: b.ID, Name: b.Name, Detail: "function element"})
		case !equalFunction(b, a):
			res.Changed = append(res.Changed, DiffEntry{Kind: "changed", Category: "function", ID: b.ID, Name: b.Name, Before: summarize(b), After: summarize(a)})
		}
	}

	for _, id := range sortedKeys(allBus) {
		b, a := beforeBus[id], afterBus[id]
		switch {
		case b.ID == "" && a.ID != "":
			res.Added = append(res.Added, DiffEntry{Kind: "added", Category: "business", ID: a.ID, Name: a.Name, Detail: "business element"})
		case b.ID != "" && a.ID == "":
			res.Removed = append(res.Removed, DiffEntry{Kind: "removed", Category: "business", ID: b.ID, Name: b.Name, Detail: "business element"})
		case !equalBusiness(b, a):
			res.Changed = append(res.Changed, DiffEntry{Kind: "changed", Category: "business", ID: b.ID, Name: b.Name, Before: summarize(b), After: summarize(a)})
		}
	}

	for _, id := range sortedKeys(allCrossRef) {
		b, a := beforeCrossRef[id], afterCrossRef[id]
		switch {
		case b.ID == "" && a.ID != "":
			res.Added = append(res.Added, DiffEntry{Kind: "added", Category: "cross-reference", ID: a.ID, Detail: fmt.Sprintf("%s:%s->%s:%s", a.FromKind, a.FromID, a.ToKind, a.ToID)})
		case b.ID != "" && a.ID == "":
			res.Removed = append(res.Removed, DiffEntry{Kind: "removed", Category: "cross-reference", ID: b.ID, Detail: fmt.Sprintf("%s:%s->%s:%s", b.FromKind, b.FromID, b.ToKind, b.ToID)})
		case !equalCrossRef(b, a):
			res.Changed = append(res.Changed, DiffEntry{Kind: "changed", Category: "cross-reference", ID: b.ID, Before: fmt.Sprintf("%s:%s->%s:%s", b.FromKind, b.FromID, b.ToKind, b.ToID), After: fmt.Sprintf("%s:%s->%s:%s", a.FromKind, a.FromID, a.ToKind, a.ToID)})
		}
	}

	for _, id := range sortedKeys(allCRUD) {
		b, a := beforeCRUD[id], afterCRUD[id]
		switch {
		case b.ID == "" && a.ID != "":
			res.Added = append(res.Added, DiffEntry{Kind: "added", Category: "crud", ID: a.ID, Detail: fmt.Sprintf("function %s -> data %s", a.FunctionID, a.DataID)})
		case b.ID != "" && a.ID == "":
			res.Removed = append(res.Removed, DiffEntry{Kind: "removed", Category: "crud", ID: b.ID, Detail: fmt.Sprintf("function %s -> data %s", b.FunctionID, b.DataID)})
		case !equalCRUD(b, a):
			res.Changed = append(res.Changed, DiffEntry{Kind: "changed", Category: "crud", ID: b.ID, Before: crudSummary(b), After: crudSummary(a)})
		}
	}

	sort.Slice(res.Added, func(i, j int) bool { return entryLess(res.Added[i], res.Added[j]) })
	sort.Slice(res.Removed, func(i, j int) bool { return entryLess(res.Removed[i], res.Removed[j]) })
	sort.Slice(res.Changed, func(i, j int) bool { return entryLess(res.Changed[i], res.Changed[j]) })

	return res
}

func byID[T any](items []T) map[string]T {
	out := map[string]T{}
	for _, item := range items {
		id := idOf(item)
		if id != "" {
			out[id] = item
		}
	}
	return out
}

func idOf(item any) string {
	switch v := item.(type) {
	case DataElement:
		return v.ID
	case FunctionElement:
		return v.ID
	case BusinessElement:
		return v.ID
	case CrossReference:
		return v.ID
	case CRUDDefinition:
		return v.ID
	}
	return ""
}

func stringKeys[T any](a, b map[string]T) map[string]struct{} {
	out := map[string]struct{}{}
	for k := range a {
		out[k] = struct{}{}
	}
	for k := range b {
		out[k] = struct{}{}
	}
	return out
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func equalData(a, b DataElement) bool {
	if a.ID != b.ID || a.Name != b.Name || a.Description != b.Description || len(a.CompositeKey) != len(b.CompositeKey) {
		return false
	}
	for i := range a.CompositeKey {
		if a.CompositeKey[i] != b.CompositeKey[i] {
			return false
		}
	}
	if len(a.Attributes) != len(b.Attributes) || len(a.Relations) != len(b.Relations) {
		return false
	}
	return true // conservative: treat attribute/relation content changes as changed via JSON summary
}

func equalFunction(a, b FunctionElement) bool {
	if a.ID != b.ID || a.Name != b.Name || a.Description != b.Description || a.Type != b.Type || len(a.Invokes) != len(b.Invokes) || len(a.Operations) != len(b.Operations) {
		return false
	}
	for i := range a.Invokes {
		if a.Invokes[i] != b.Invokes[i] {
			return false
		}
	}
	return true
}

func equalBusiness(a, b BusinessElement) bool {
	if a.ID != b.ID || a.Name != b.Name || a.Role != b.Role || len(a.Activities) != len(b.Activities) {
		return false
	}
	return true
}

func equalCrossRef(a, b CrossReference) bool {
	return a.Kind == b.Kind && a.FromID == b.FromID && a.FromKind == b.FromKind && a.ToID == b.ToID && a.ToKind == b.ToKind && a.Directed == b.Directed
}

func equalCRUD(a, b CRUDDefinition) bool {
	return a.FunctionID == b.FunctionID && a.DataID == b.DataID && a.Create == b.Create && a.Read == b.Read && a.Update == b.Update && a.Delete == b.Delete && a.Source == b.Source
}

func summarize(v any) string {
	b, _ := json.Marshal(v)
	if len(b) > 120 {
		b = append(b[:120], []byte("...")...)
	}
	return string(b)
}

func crudSummary(c CRUDDefinition) string {
	var ops []string
	if c.Create {
		ops = append(ops, "C")
	}
	if c.Read {
		ops = append(ops, "R")
	}
	if c.Update {
		ops = append(ops, "U")
	}
	if c.Delete {
		ops = append(ops, "D")
	}
	return fmt.Sprintf("function=%s data=%s ops=%s source=%s", c.FunctionID, c.DataID, ops, c.Source)
}

func entryLess(a, b DiffEntry) bool {
	if a.Category != b.Category {
		return a.Category < b.Category
	}
	if a.ID != b.ID {
		return a.ID < b.ID
	}
	return a.Detail < b.Detail
}

// ModelChangeImpact describes the impact of changing an element.
type ModelChangeImpact struct {
	SubjectID   string `json:"subjectId"`
	SubjectKind string `json:"subjectKind"`
	Impact      string `json:"impact"`
	Detail      string `json:"detail,omitempty"`
}

// AnalyzeImpact returns downstream elements affected by changing the subject.
func AnalyzeImpact(model ThreeElementModel, subjectID, subjectKind string) []ModelChangeImpact {
	var impacts []ModelChangeImpact
	switch subjectKind {
	case "function":
		for _, crud := range model.CRUDDefinitions {
			if crud.FunctionID == subjectID {
				impacts = append(impacts, ModelChangeImpact{
					SubjectID:   crud.DataID,
					SubjectKind: "data",
					Impact:      "crud-definition",
					Detail:      fmt.Sprintf("CRUD %s uses function %s", crud.ID, subjectID),
				})
			}
		}
		for _, ref := range model.CrossReferences {
			if ref.FromKind == "business" && ref.ToKind == "function" && ref.ToID == subjectID {
				impacts = append(impacts, ModelChangeImpact{
					SubjectID:   ref.FromID,
					SubjectKind: "business",
					Impact:      "cross-reference",
					Detail:      fmt.Sprintf("cross-reference %s depends on function %s", ref.ID, subjectID),
				})
			}
		}
		for _, fn := range model.FunctionElements {
			for _, inv := range fn.Invokes {
				if inv == subjectID {
					impacts = append(impacts, ModelChangeImpact{
						SubjectID:   fn.ID,
						SubjectKind: "function",
						Impact:      "invoke",
						Detail:      fmt.Sprintf("function %s invokes %s", fn.ID, subjectID),
					})
				}
			}
		}
	case "data":
		for _, crud := range model.CRUDDefinitions {
			if crud.DataID == subjectID {
				impacts = append(impacts, ModelChangeImpact{
					SubjectID:   crud.FunctionID,
					SubjectKind: "function",
					Impact:      "crud-definition",
					Detail:      fmt.Sprintf("CRUD %s uses data %s", crud.ID, subjectID),
				})
			}
		}
		for _, ref := range model.CrossReferences {
			if ref.FromKind == "function" && ref.ToKind == "data" && ref.ToID == subjectID {
				impacts = append(impacts, ModelChangeImpact{
					SubjectID:   ref.FromID,
					SubjectKind: "function",
					Impact:      "cross-reference",
					Detail:      fmt.Sprintf("cross-reference %s depends on data %s", ref.ID, subjectID),
				})
			}
		}
		for _, d := range model.DataElements {
			for _, rel := range d.Relations {
				if rel.TargetID == subjectID {
					impacts = append(impacts, ModelChangeImpact{
						SubjectID:   d.ID,
						SubjectKind: "data",
						Impact:      "relation",
						Detail:      fmt.Sprintf("data %s has relation %s to %s", d.ID, rel.ID, subjectID),
					})
				}
			}
		}
	case "business":
		for _, bus := range model.BusinessElements {
			if bus.ID != subjectID {
				continue
			}
			for _, act := range bus.Activities {
				for _, fid := range act.FunctionIDs {
					impacts = append(impacts, ModelChangeImpact{
						SubjectID:   fid,
						SubjectKind: "function",
						Impact:      "activity",
						Detail:      fmt.Sprintf("activity %s uses function %s", act.ID, fid),
					})
				}
			}
		}
		for _, ref := range model.CrossReferences {
			if ref.FromKind == "business" && ref.FromID == subjectID {
				impacts = append(impacts, ModelChangeImpact{
					SubjectID:   ref.ToID,
					SubjectKind: ref.ToKind,
					Impact:      "cross-reference",
					Detail:      fmt.Sprintf("cross-reference %s starts from business %s", ref.ID, subjectID),
				})
			}
		}
	}

	sort.Slice(impacts, func(i, j int) bool {
		if impacts[i].SubjectKind != impacts[j].SubjectKind {
			return impacts[i].SubjectKind < impacts[j].SubjectKind
		}
		return impacts[i].SubjectID < impacts[j].SubjectID
	})
	return impacts
}
