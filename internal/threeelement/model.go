package threeelement

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const schemaVersion = "tei.three-element.model/v0"

// EffectReadOnly marks a projection that does not authorize source writes.
const EffectReadOnly = "read-only"

// DataElement is a stable Identity for a data entity in the Data Model.
type DataElement struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	Attributes      []DataAttribute `json:"attributes,omitempty"`
	Relations       []DataRelation  `json:"relations,omitempty"`
	CompositeKey    []string        `json:"compositeKey,omitempty"`
	AuthorityLimits []string        `json:"authorityLimits,omitempty"`
}

// DataAttribute describes one field of a DataElement.
type DataAttribute struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable,omitempty"`
	Key      bool   `json:"key,omitempty"`
}

// DataRelation describes a directed or undirected relation between data entities.
type DataRelation struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	TargetID string `json:"targetId"`
	Kind     string `json:"kind"` // one-to-many | many-to-many | one-to-one | lookup
	Directed bool   `json:"directed,omitempty"`
	Role     string `json:"role,omitempty"`
}

// FunctionElement is a stable Identity for a function in the Function Model.
type FunctionElement struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Description     string              `json:"description,omitempty"`
	Type            string              `json:"type,omitempty"` // e.g. XF200, batch, report, integration
	Operations      []FunctionOperation `json:"operations,omitempty"`
	Invokes         []string            `json:"invokes,omitempty"` // stable function IDs
	AuthorityLimits []string            `json:"authorityLimits,omitempty"`
}

// FunctionOperation records declared or derived CRUD intent for one data target.
type FunctionOperation struct {
	DataID     string `json:"dataId"`
	Operations string `json:"operations"` // subset of C,R,U,D
}

// BusinessElement is a stable Identity for a business element in the Business Model.
type BusinessElement struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Description     string             `json:"description,omitempty"`
	Role            string             `json:"role,omitempty"` // actor, role, organization
	Activities      []BusinessActivity `json:"activities,omitempty"`
	AuthorityLimits []string           `json:"authorityLimits,omitempty"`
}

// BusinessActivity describes one business activity and the functions it uses.
type BusinessActivity struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	FunctionIDs []string `json:"functionIds,omitempty"`
}

// CrossReference is a typed, directed or undirected relation between elements.
type CrossReference struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"` // business-to-function | function-to-data | data-to-data | business-to-data
	FromID   string `json:"fromId"`
	FromKind string `json:"fromKind"` // business | function | data
	ToID     string `json:"toId"`
	ToKind   string `json:"toKind"` // business | function | data
	Role     string `json:"role,omitempty"`
	Directed bool   `json:"directed,omitempty"`
}

// CRUDDefinition records one Function→Data CRUD mapping.
type CRUDDefinition struct {
	ID         string `json:"id"`
	FunctionID string `json:"functionId"`
	DataID     string `json:"dataId"`
	Create     bool   `json:"create,omitempty"`
	Read       bool   `json:"read,omitempty"`
	Update     bool   `json:"update,omitempty"`
	Delete     bool   `json:"delete,omitempty"`
	Source     string `json:"source,omitempty"` // declared | derived | inferred
}

// Diagnostic is a finding from validation or derivation.
type Diagnostic struct {
	Code       string   `json:"code"`
	Severity   string   `json:"severity"` // error | warning | info
	Message    string   `json:"message"`
	SubjectIDs []string `json:"subjectIds,omitempty"`
}

// ModelFingerprint returns a stable digest for the model content without
// changing the caller's model. The derived fingerprint must not hash itself.
func ModelFingerprint(model ThreeElementModel) string {
	// These fields describe observation and authority, not the structural
	// model content. Clear them only on the local projection being hashed.
	model.ModelFingerprint = ""
	model.Diagnostics = nil
	model.AuthorityLimits = nil
	if len(model.DataElements) > 0 {
		model.DataElements = append([]DataElement(nil), model.DataElements...)
		for i := range model.DataElements {
			model.DataElements[i].AuthorityLimits = nil
		}
	}
	if len(model.FunctionElements) > 0 {
		model.FunctionElements = append([]FunctionElement(nil), model.FunctionElements...)
		for i := range model.FunctionElements {
			model.FunctionElements[i].AuthorityLimits = nil
		}
	}
	if len(model.BusinessElements) > 0 {
		model.BusinessElements = append([]BusinessElement(nil), model.BusinessElements...)
		for i := range model.BusinessElements {
			model.BusinessElements[i].AuthorityLimits = nil
		}
	}
	b, _ := json.Marshal(model)
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ThreeElementModel is the unified three-element analysis model.
type ThreeElementModel struct {
	SchemaVersion    string            `json:"schemaVersion"`
	Effect           string            `json:"effect"`
	ModelID          string            `json:"modelId"`
	Name             string            `json:"name"`
	Description      string            `json:"description,omitempty"`
	ModelFingerprint string            `json:"modelFingerprint"`
	DataElements     []DataElement     `json:"dataElements"`
	FunctionElements []FunctionElement `json:"functionElements"`
	BusinessElements []BusinessElement `json:"businessElements"`
	CrossReferences  []CrossReference  `json:"crossReferences,omitempty"`
	CRUDDefinitions  []CRUDDefinition  `json:"crudDefinitions,omitempty"`
	Diagnostics      []Diagnostic      `json:"diagnostics,omitempty"`
	AuthorityLimits  []string          `json:"authorityLimits,omitempty"`
}

// NewThreeElementModel creates a minimal valid model.
func NewThreeElementModel(modelID, name string) ThreeElementModel {
	return ThreeElementModel{
		SchemaVersion:   schemaVersion,
		Effect:          EffectReadOnly,
		ModelID:         strings.TrimSpace(modelID),
		Name:            strings.TrimSpace(name),
		AuthorityLimits: []string{"derivation-does-not-authorize-writes", "owner-acceptance-may-be-required"},
	}
}

// Index builds lookup maps by stable ID for each element kind.
// A duplicate ID WITHIN a kind is ambiguous: leave it out of the lookup
// instead of silently selecting the last definition. Validate reports each
// collision as an error. Equal IDs across different kinds remain distinct.
func (m ThreeElementModel) Index() (map[string]DataElement, map[string]FunctionElement, map[string]BusinessElement) {
	data := map[string]DataElement{}
	ambiguousData := map[string]bool{}
	for _, e := range m.DataElements {
		if ambiguousData[e.ID] {
			continue
		}
		if _, exists := data[e.ID]; exists {
			delete(data, e.ID)
			ambiguousData[e.ID] = true
			continue
		}
		data[e.ID] = e
	}
	fn := map[string]FunctionElement{}
	ambiguousFunction := map[string]bool{}
	for _, e := range m.FunctionElements {
		if ambiguousFunction[e.ID] {
			continue
		}
		if _, exists := fn[e.ID]; exists {
			delete(fn, e.ID)
			ambiguousFunction[e.ID] = true
			continue
		}
		fn[e.ID] = e
	}
	bus := map[string]BusinessElement{}
	ambiguousBusiness := map[string]bool{}
	for _, e := range m.BusinessElements {
		if ambiguousBusiness[e.ID] {
			continue
		}
		if _, exists := bus[e.ID]; exists {
			delete(bus, e.ID)
			ambiguousBusiness[e.ID] = true
			continue
		}
		bus[e.ID] = e
	}
	return data, fn, bus
}

// Validate runs structural checks and returns diagnostics.
func (m ThreeElementModel) Validate() []Diagnostic {
	var diags []Diagnostic
	// Detect collisions against the source arrays before looking anything up.
	// Code values are literal so repository-wide diagnostic coverage verifies
	// each public diagnostic contract against its independent test.
	dataPositions := map[string]int{}
	for i, e := range m.DataElements {
		if e.ID == "" {
			continue // EMPTY_DATA_ID is separately diagnosed below.
		}
		if first, exists := dataPositions[e.ID]; exists {
			diags = append(diags, Diagnostic{
				Code:       "DUPLICATE_DATA_ID",
				Severity:   "error",
				Message:    fmt.Sprintf("data id %q is duplicated at positions %d and %d", e.ID, first, i+1),
				SubjectIDs: []string{e.ID},
			})
		} else {
			dataPositions[e.ID] = i + 1
		}
	}
	functionPositions := map[string]int{}
	for i, e := range m.FunctionElements {
		if e.ID == "" {
			continue // EMPTY_FUNCTION_ID is separately diagnosed below.
		}
		if first, exists := functionPositions[e.ID]; exists {
			diags = append(diags, Diagnostic{
				Code:       "DUPLICATE_FUNCTION_ID",
				Severity:   "error",
				Message:    fmt.Sprintf("function id %q is duplicated at positions %d and %d", e.ID, first, i+1),
				SubjectIDs: []string{e.ID},
			})
		} else {
			functionPositions[e.ID] = i + 1
		}
	}
	businessPositions := map[string]int{}
	for i, e := range m.BusinessElements {
		if e.ID == "" {
			continue // EMPTY_BUSINESS_ID is separately diagnosed below.
		}
		if first, exists := businessPositions[e.ID]; exists {
			diags = append(diags, Diagnostic{
				Code:       "DUPLICATE_BUSINESS_ID",
				Severity:   "error",
				Message:    fmt.Sprintf("business id %q is duplicated at positions %d and %d", e.ID, first, i+1),
				SubjectIDs: []string{e.ID},
			})
		} else {
			businessPositions[e.ID] = i + 1
		}
	}
	dataByID, fnByID, busByID := m.Index()

	for _, e := range m.DataElements {
		if e.ID == "" {
			diags = append(diags, Diagnostic{Code: "EMPTY_DATA_ID", Severity: "error", Message: "data element has empty id"})
		}
		if e.Name == "" {
			diags = append(diags, Diagnostic{Code: "EMPTY_DATA_NAME", Severity: "warning", Message: fmt.Sprintf("data element %q has empty name", e.ID)})
		}
		for _, attr := range e.Attributes {
			if attr.ID == "" {
				diags = append(diags, Diagnostic{Code: "EMPTY_ATTRIBUTE_ID", Severity: "error", Message: fmt.Sprintf("data %q has attribute with empty id", e.ID), SubjectIDs: []string{e.ID}})
			}
		}
		for _, rel := range e.Relations {
			if rel.TargetID == "" {
				diags = append(diags, Diagnostic{Code: "EMPTY_RELATION_TARGET", Severity: "error", Message: fmt.Sprintf("data relation %q has empty target", rel.ID), SubjectIDs: []string{e.ID, rel.ID}})
			} else if _, ok := dataByID[rel.TargetID]; !ok {
				diags = append(diags, Diagnostic{Code: "UNKNOWN_RELATION_TARGET", Severity: "warning", Message: fmt.Sprintf("data relation %q targets unknown data %q", rel.ID, rel.TargetID), SubjectIDs: []string{e.ID, rel.TargetID}})
			}
		}
		if len(e.CompositeKey) > 0 {
			for _, k := range e.CompositeKey {
				found := false
				for _, attr := range e.Attributes {
					if attr.ID == k {
						found = true
						break
					}
				}
				if !found {
					diags = append(diags, Diagnostic{Code: "COMPOSITE_KEY_MISSING_ATTRIBUTE", Severity: "error", Message: fmt.Sprintf("composite key %q on data %q does not match any attribute", k, e.ID), SubjectIDs: []string{e.ID}})
				}
			}
		}
	}

	for _, e := range m.FunctionElements {
		if e.ID == "" {
			diags = append(diags, Diagnostic{Code: "EMPTY_FUNCTION_ID", Severity: "error", Message: "function element has empty id"})
		}
		if e.Name == "" {
			diags = append(diags, Diagnostic{Code: "EMPTY_FUNCTION_NAME", Severity: "warning", Message: fmt.Sprintf("function element %q has empty name", e.ID)})
		}
		for _, op := range e.Operations {
			if _, ok := dataByID[op.DataID]; !ok {
				diags = append(diags, Diagnostic{Code: "UNKNOWN_OPERATION_TARGET", Severity: "error", Message: fmt.Sprintf("function %q operates on unknown data %q", e.ID, op.DataID), SubjectIDs: []string{e.ID, op.DataID}})
			}
			for _, ch := range op.Operations {
				if !strings.Contains("CRUD", strings.ToUpper(string(ch))) {
					diags = append(diags, Diagnostic{Code: "UNKNOWN_OPERATION_LETTER", Severity: "error", Message: fmt.Sprintf("function %q has unknown operation %q on data %q", e.ID, string(ch), op.DataID), SubjectIDs: []string{e.ID, op.DataID}})
				}
			}
		}
		for _, inv := range e.Invokes {
			if _, ok := fnByID[inv]; !ok {
				diags = append(diags, Diagnostic{Code: "UNKNOWN_INVOKE_TARGET", Severity: "warning", Message: fmt.Sprintf("function %q invokes unknown function %q", e.ID, inv), SubjectIDs: []string{e.ID, inv}})
			}
		}
	}

	for _, e := range m.BusinessElements {
		if e.ID == "" {
			diags = append(diags, Diagnostic{Code: "EMPTY_BUSINESS_ID", Severity: "error", Message: "business element has empty id"})
		}
		if e.Name == "" {
			diags = append(diags, Diagnostic{Code: "EMPTY_BUSINESS_NAME", Severity: "warning", Message: fmt.Sprintf("business element %q has empty name", e.ID)})
		}
		for _, act := range e.Activities {
			if act.ID == "" {
				diags = append(diags, Diagnostic{Code: "EMPTY_ACTIVITY_ID", Severity: "error", Message: fmt.Sprintf("business element %q has activity with empty id", e.ID), SubjectIDs: []string{e.ID}})
			}
			for _, fid := range act.FunctionIDs {
				if _, ok := fnByID[fid]; !ok {
					diags = append(diags, Diagnostic{Code: "UNKNOWN_ACTIVITY_FUNCTION", Severity: "warning", Message: fmt.Sprintf("business activity %q references unknown function %q", act.ID, fid), SubjectIDs: []string{e.ID, act.ID, fid}})
				}
			}
		}
	}

	for _, ref := range m.CrossReferences {
		if ref.ID == "" {
			diags = append(diags, Diagnostic{Code: "EMPTY_CROSS_REF_ID", Severity: "error", Message: "cross-reference has empty id"})
		}
		if _, ok := resolveSubject(ref.FromKind, ref.FromID, dataByID, fnByID, busByID); !ok {
			diags = append(diags, Diagnostic{Code: "UNKNOWN_CROSS_REF_FROM", Severity: "warning", Message: fmt.Sprintf("cross-reference %q from unknown %s/%s", ref.ID, ref.FromKind, ref.FromID), SubjectIDs: []string{ref.ID, ref.FromID}})
		}
		if _, ok := resolveSubject(ref.ToKind, ref.ToID, dataByID, fnByID, busByID); !ok {
			diags = append(diags, Diagnostic{Code: "UNKNOWN_CROSS_REF_TO", Severity: "warning", Message: fmt.Sprintf("cross-reference %q to unknown %s/%s", ref.ID, ref.ToKind, ref.ToID), SubjectIDs: []string{ref.ID, ref.ToID}})
		}
	}

	for _, crud := range m.CRUDDefinitions {
		if _, ok := fnByID[crud.FunctionID]; !ok {
			diags = append(diags, Diagnostic{Code: "UNKNOWN_CRUD_FUNCTION", Severity: "error", Message: fmt.Sprintf("CRUD %q references unknown function %q", crud.ID, crud.FunctionID), SubjectIDs: []string{crud.ID, crud.FunctionID}})
		}
		if _, ok := dataByID[crud.DataID]; !ok {
			diags = append(diags, Diagnostic{Code: "UNKNOWN_CRUD_DATA", Severity: "error", Message: fmt.Sprintf("CRUD %q references unknown data %q", crud.ID, crud.DataID), SubjectIDs: []string{crud.ID, crud.DataID}})
		}
		if !crud.Create && !crud.Read && !crud.Update && !crud.Delete {
			diags = append(diags, Diagnostic{Code: "EMPTY_CRUD_DEFINITION", Severity: "warning", Message: fmt.Sprintf("CRUD %q has no operations", crud.ID), SubjectIDs: []string{crud.ID}})
		}
	}

	sort.Slice(diags, func(i, j int) bool {
		if diags[i].Severity != diags[j].Severity {
			return severityRank(diags[i].Severity) < severityRank(diags[j].Severity)
		}
		return diags[i].Code < diags[j].Code
	})
	return diags
}

func severityRank(s string) int {
	switch s {
	case "error":
		return 0
	case "warning":
		return 1
	case "info":
		return 2
	}
	return 3
}

func resolveSubject(kind, id string, data map[string]DataElement, fn map[string]FunctionElement, bus map[string]BusinessElement) (string, bool) {
	switch kind {
	case "data":
		_, ok := data[id]
		return "data", ok
	case "function":
		_, ok := fn[id]
		return "function", ok
	case "business":
		_, ok := bus[id]
		return "business", ok
	}
	return kind, false
}

// CopyDiagnostics returns a model with diagnostics replaced.
func (m ThreeElementModel) CopyDiagnostics(diags []Diagnostic) ThreeElementModel {
	m.Diagnostics = diags
	return m
}

// UnusedData returns data IDs that have no CRUD references from any function.
func (m ThreeElementModel) UnusedData() []string {
	_, fnByID, _ := m.Index()
	used := map[string]bool{}
	for _, fn := range fnByID {
		for _, op := range fn.Operations {
			used[op.DataID] = true
		}
	}
	// An explicit-only CRUD mapping is still a Function→Data reference.
	// Only a uniquely resolved Function proves usage; unknown/ambiguous
	// Function IDs must not hide an otherwise unused Data element.
	for _, crud := range m.CRUDDefinitions {
		if _, ok := fnByID[crud.FunctionID]; ok {
			used[crud.DataID] = true
		}
	}
	var out []string
	for _, d := range m.DataElements {
		if !used[d.ID] {
			out = append(out, d.ID)
		}
	}
	sort.Strings(out)
	return out
}

// UnreferencedFunctions returns function IDs not invoked by any business activity.
func (m ThreeElementModel) UnreferencedFunctions() []string {
	_, fnByID, busByID := m.Index()
	referenced := map[string]bool{}
	for _, bus := range busByID {
		for _, act := range bus.Activities {
			for _, fid := range act.FunctionIDs {
				referenced[fid] = true
			}
		}
	}
	var out []string
	for id := range fnByID {
		if !referenced[id] {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// MissingBusinessFunctionMapping returns business activities with empty function lists.
func (m ThreeElementModel) MissingBusinessFunctionMapping() []string {
	_, _, busByID := m.Index()
	var out []string
	for _, bus := range busByID {
		for _, act := range bus.Activities {
			if len(act.FunctionIDs) == 0 {
				out = append(out, act.ID)
			}
		}
	}
	sort.Strings(out)
	return out
}

// ContradictoryCRUDs returns every later CRUD definition whose operation
// flags conflict with the preceding definition for the same Function/Data pair.
// Keep source order and the source definition intact; never silently choose a
// winning definition when constructing validation evidence.
func (m ThreeElementModel) ContradictoryCRUDs() []CRUDDefinition {
	type pair struct{ fn, data string }
	best := map[pair]CRUDDefinition{}
	var conflicts []CRUDDefinition
	for _, crud := range m.CRUDDefinitions {
		key := pair{crud.FunctionID, crud.DataID}
		if prev, ok := best[key]; ok {
			if prev.Create != crud.Create || prev.Read != crud.Read || prev.Update != crud.Update || prev.Delete != crud.Delete {
				conflicts = append(conflicts, crud)
			}
		}
		best[key] = crud
	}
	return conflicts
}

// UnknownDependencies returns function invokes that reference nonexistent functions.
func (m ThreeElementModel) UnknownDependencies() [][2]string {
	_, fnByID, _ := m.Index()
	var out [][2]string
	for _, fn := range fnByID {
		for _, inv := range fn.Invokes {
			if _, ok := fnByID[inv]; !ok {
				out = append(out, [2]string{fn.ID, inv})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i][0] != out[j][0] {
			return out[i][0] < out[j][0]
		}
		return out[i][1] < out[j][1]
	})
	return out
}

// SameNameConflicts returns IDs that share a name across different elements of the same kind.
func (m ThreeElementModel) SameNameConflicts() map[string][]string {
	byName := map[string][]string{}
	add := func(kind string, elements []struct{ id, name string }) {
		for _, e := range elements {
			if e.id == "" || e.name == "" {
				continue
			}
			key := kind + ":" + strings.ToLower(e.name)
			byName[key] = append(byName[key], e.id)
		}
	}
	var dataNames []struct{ id, name string }
	for _, d := range m.DataElements {
		dataNames = append(dataNames, struct{ id, name string }{d.ID, d.Name})
	}
	var fnNames []struct{ id, name string }
	for _, f := range m.FunctionElements {
		fnNames = append(fnNames, struct{ id, name string }{f.ID, f.Name})
	}
	var busNames []struct{ id, name string }
	for _, b := range m.BusinessElements {
		busNames = append(busNames, struct{ id, name string }{b.ID, b.Name})
	}
	add("data", dataNames)
	add("function", fnNames)
	add("business", busNames)
	out := map[string][]string{}
	for key, ids := range byName {
		if len(ids) > 1 {
			out[key] = ids
		}
	}
	return out
}
