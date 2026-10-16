// Package scenariodiff validates a typed Dataset/Snapshot against a model, derives a
// Business State Diff between snapshots, and compares a Scenario's expected invariants
// with its observations. Steps are analysed in their declared order; presentation order
// is metadata and is never used as time. An expectation whose only observation is an
// unobserved manual or external act stays unverified instead of passing.
//
// This is a read-only contract reference. It is not a simulator, a Driver, or a slide
// renderer, and it does not write a Source.
package scenariodiff

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const Schema = "tei.reference.scenario-state-diff/v0"

// Field is one typed field of a data definition.
type Field struct {
	Name string `json:"name"`
	Type string `json:"type"` // string | int | decimal | bool
	Key  bool   `json:"key"`
}

// DataDef is one typed business data definition.
type DataDef struct {
	ID     string  `json:"id"`
	Fields []Field `json:"fields"`
}

// Relation is one typed relation between two fields.
type Relation struct {
	ID   string `json:"id"`
	From string `json:"from"` // "Data.field"
	To   string `json:"to"`   // "Data.field"
}

// Model is the typed business model.
type Model struct {
	Data      []DataDef  `json:"data"`
	Relations []Relation `json:"relations"`
}

// DataSet is one typed set of rows for a data definition.
type DataSet struct {
	Data string           `json:"data"`
	Rows []map[string]any `json:"rows"`
}

// Snapshot is a bounded business state.
type Snapshot struct {
	ID      string    `json:"id"`
	Dataset []DataSet `json:"dataset"`
}

// Invariant is an expected or observed fact about one field.
type Invariant struct {
	Data  string `json:"data"`
	Field string `json:"field"`
	Op    string `json:"op"` // eq | ne | gt | lt | exists | absent
	Value any    `json:"value"`
}

// Observation is a raw observation. Manual and external acts are observed only when the
// source explicitly reports them; otherwise they remain unobserved.
type Observation struct {
	Source   string `json:"source"` // internal | manual | external
	Data     string `json:"data"`
	Field    string `json:"field"`
	Observed bool   `json:"observed"`
}

// Step is one scenario step.
type Step struct {
	ID           string        `json:"id"`
	Action       string        `json:"action"`
	Transition   string        `json:"transition"`
	Expected     []Invariant   `json:"expected"`
	Observed     []Invariant   `json:"observed"`
	Observations []Observation `json:"observations"`
	// PresentationOrder only orders material for a person. It is never timestamps
	// or step order.
	PresentationOrder int `json:"presentationOrder"`
}

// Scenario is one ordered set of steps.
type Scenario struct {
	ID    string `json:"id"`
	Steps []Step `json:"steps"`
}

// Input is one read-only analysis request.
type Input struct {
	Model     Model      `json:"model"`
	Snapshots []Snapshot `json:"snapshots"`
	Scenarios []Scenario `json:"scenarios"`
}

// Finding is one diagnostic.
type Finding struct {
	Code   string `json:"code"`
	Data   string `json:"data,omitempty"`
	Field  string `json:"field,omitempty"`
	Step   string `json:"step,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// Cell is one changed field of one row.
type Cell struct {
	Data   string `json:"data"`
	Key    string `json:"key"`
	Field  string `json:"field"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

// SnapshotDiff is the business state diff between two snapshots.
type SnapshotDiff struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Added   []Cell `json:"added"`
	Removed []Cell `json:"removed"`
	Changed []Cell `json:"changed"`
}

// ExpectationResult is the result of one expected invariant.
type ExpectationResult struct {
	Invariant Invariant `json:"invariant"`
	Status    string    `json:"status"` // passed | failed | unverified
	Actual    any       `json:"actual,omitempty"`
}

// StepResult is the result of one step.
type StepResult struct {
	Step     string              `json:"step"`
	Status   string              `json:"status"` // passed | failed | unverified
	Results  []ExpectationResult `json:"results"`
	Findings []Finding           `json:"findings"`
}

// ScenarioResult is the result of one scenario. Steps keep their declared analysis
// order. Presentation lists the step IDs in presentation order, kept apart from it.
type ScenarioResult struct {
	Scenario     string       `json:"scenario"`
	Status       string       `json:"status"`
	Steps        []StepResult `json:"steps"`
	Presentation []string     `json:"presentation"`
}

// Summary counts the results.
type Summary struct {
	ModelFindings int `json:"modelFindings"`
	DataFindings  int `json:"dataFindings"`
	Passed        int `json:"passed"`
	Failed        int `json:"failed"`
	Unverified    int `json:"unverified"`
}

// Report is the read-only result.
type Report struct {
	Schema        string           `json:"schema"`
	ModelFindings []Finding        `json:"modelFindings"`
	DataFindings  []Finding        `json:"dataFindings"`
	SnapshotDiffs []SnapshotDiff   `json:"snapshotDiffs"`
	Scenarios     []ScenarioResult `json:"scenarios"`
	Summary       Summary          `json:"summary"`
	Fingerprint   string           `json:"fingerprint"`
}

var knownTypes = map[string]bool{"string": true, "int": true, "decimal": true, "bool": true}

func splitRef(ref string) (string, string) {
	i := strings.LastIndexByte(ref, '.')
	if i < 0 {
		return ref, ""
	}
	return ref[:i], ref[i+1:]
}

func (m Model) field(data, field string) (Field, DataDef, bool) {
	for _, d := range m.Data {
		if d.ID != data {
			continue
		}
		for _, f := range d.Fields {
			if f.Name == field {
				return f, d, true
			}
		}
		return Field{}, d, false
	}
	return Field{}, DataDef{}, false
}

func validateModel(m Model) []Finding {
	findings := []Finding{}
	seenData := map[string]bool{}
	for _, d := range m.Data {
		if d.ID == "" {
			findings = append(findings, Finding{Code: "data-id-missing"})
			continue
		}
		if seenData[d.ID] {
			findings = append(findings, Finding{Code: "data-duplicate", Data: d.ID})
		}
		seenData[d.ID] = true
		seenField := map[string]bool{}
		keys := 0
		for _, f := range d.Fields {
			if seenField[f.Name] {
				findings = append(findings, Finding{Code: "field-duplicate", Data: d.ID, Field: f.Name})
			}
			seenField[f.Name] = true
			if !knownTypes[f.Type] {
				findings = append(findings, Finding{Code: "unknown-field-type", Data: d.ID, Field: f.Name, Detail: f.Type})
			}
			if f.Key {
				keys++
			}
		}
		if keys == 0 {
			findings = append(findings, Finding{Code: "data-key-missing", Data: d.ID})
		}
	}
	for _, r := range m.Relations {
		fromData, fromField := splitRef(r.From)
		toData, toField := splitRef(r.To)
		if _, _, ok := m.field(fromData, fromField); !ok {
			findings = append(findings, Finding{Code: "relation-source-missing", Detail: r.From})
		}
		if _, _, ok := m.field(toData, toField); !ok {
			findings = append(findings, Finding{Code: "relation-target-missing", Detail: r.To})
		}
	}
	return findings
}

func matchesType(kind string, v any) bool {
	switch kind {
	case "string":
		_, ok := v.(string)
		return ok
	case "bool":
		_, ok := v.(bool)
		return ok
	case "int":
		f, ok := v.(float64)
		return ok && f == float64(int64(f))
	case "decimal":
		_, ok := v.(float64)
		return ok
	}
	return false
}

func keyOf(d DataDef, row map[string]any) (string, bool) {
	parts := []string{}
	for _, f := range d.Fields {
		if !f.Key {
			continue
		}
		v, ok := row[f.Name]
		if !ok || v == nil {
			return "", false
		}
		b, _ := json.Marshal(v)
		parts = append(parts, string(b))
	}
	return strings.Join(parts, "\x00"), true
}

func validateDataset(m Model, ds DataSet, all []DataSet) []Finding {
	findings := []Finding{}
	def, ok := dataDef(m, ds.Data)
	if !ok {
		return []Finding{{Code: "unknown-data", Data: ds.Data}}
	}
	seenKeys := map[string]bool{}
	for _, row := range ds.Rows {
		for name, v := range row {
			f, _, ok := m.field(ds.Data, name)
			if !ok {
				findings = append(findings, Finding{Code: "unknown-field", Data: ds.Data, Field: name})
				continue
			}
			if v != nil && !matchesType(f.Type, v) {
				findings = append(findings, Finding{Code: "field-type-mismatch", Data: ds.Data, Field: name})
			}
		}
		for _, f := range def.Fields {
			if !f.Key {
				continue
			}
			if _, ok := row[f.Name]; !ok {
				findings = append(findings, Finding{Code: "missing-key", Data: ds.Data, Field: f.Name})
			}
		}
		if key, ok := keyOf(def, row); ok {
			if seenKeys[key] {
				findings = append(findings, Finding{Code: "duplicate-key", Data: ds.Data, Detail: key})
			}
			seenKeys[key] = true
		}
	}
	// Typed relation references must not dangle within the same snapshot.
	for _, r := range m.Relations {
		fromData, fromField := splitRef(r.From)
		toData, toField := splitRef(r.To)
		if fromData != ds.Data {
			continue
		}
		target, ok := datasetOf(all, toData)
		if !ok {
			continue
		}
		for _, row := range ds.Rows {
			ref, ok := row[fromField]
			if !ok || ref == nil || ref == "" {
				continue
			}
			if !targetHasKey(m, target, toField, ref) {
				findings = append(findings, Finding{Code: "dangling-reference", Data: ds.Data, Field: fromField, Detail: r.ID})
			}
		}
	}
	return findings
}

func dataDef(m Model, id string) (DataDef, bool) {
	for _, d := range m.Data {
		if d.ID == id {
			return d, true
		}
	}
	return DataDef{}, false
}

func datasetOf(datasets []DataSet, data string) (DataSet, bool) {
	for _, ds := range datasets {
		if ds.Data == data {
			return ds, true
		}
	}
	return DataSet{}, false
}

func targetHasKey(m Model, target DataSet, field string, ref any) bool {
	for _, row := range target.Rows {
		if v, ok := row[field]; ok && v == ref {
			return true
		}
	}
	return false
}

func validateSnapshots(m Model, snaps []Snapshot) []Finding {
	findings := []Finding{}
	for _, s := range snaps {
		for _, ds := range s.Dataset {
			for _, f := range validateDataset(m, ds, s.Dataset) {
				f.Step = s.ID
				findings = append(findings, f)
			}
		}
	}
	return findings
}

func indexRows(m Model, ds DataSet) (map[string]map[string]any, DataDef, bool) {
	def, ok := dataDef(m, ds.Data)
	if !ok {
		return nil, DataDef{}, false
	}
	out := map[string]map[string]any{}
	for _, row := range ds.Rows {
		if key, ok := keyOf(def, row); ok {
			out[key] = row
		}
	}
	return out, def, true
}

func diffSnapshots(m Model, from, to Snapshot) SnapshotDiff {
	d := SnapshotDiff{From: from.ID, To: to.ID, Added: []Cell{}, Removed: []Cell{}, Changed: []Cell{}}
	names := map[string]bool{}
	for _, ds := range from.Dataset {
		names[ds.Data] = true
	}
	for _, ds := range to.Dataset {
		names[ds.Data] = true
	}
	ordered := make([]string, 0, len(names))
	for n := range names {
		ordered = append(ordered, n)
	}
	sort.Strings(ordered)
	for _, name := range ordered {
		beforeDs, _ := datasetOf(from.Dataset, name)
		afterDs, _ := datasetOf(to.Dataset, name)
		before, def, ok := indexRows(m, DataSet{Data: name, Rows: beforeDs.Rows})
		if !ok {
			continue
		}
		after, _, _ := indexRows(m, DataSet{Data: name, Rows: afterDs.Rows})
		keys := map[string]bool{}
		for k := range before {
			keys[k] = true
		}
		for k := range after {
			keys[k] = true
		}
		sorted := make([]string, 0, len(keys))
		for k := range keys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			b, inB := before[k]
			a, inA := after[k]
			switch {
			case !inB && inA:
				for _, f := range def.Fields {
					d.Added = append(d.Added, Cell{Data: def.ID, Key: k, Field: f.Name, After: a[f.Name]})
				}
			case inB && !inA:
				for _, f := range def.Fields {
					d.Removed = append(d.Removed, Cell{Data: def.ID, Key: k, Field: f.Name, Before: b[f.Name]})
				}
			case inB && inA:
				for _, f := range def.Fields {
					if !valueEqual(b[f.Name], a[f.Name]) {
						d.Changed = append(d.Changed, Cell{Data: def.ID, Key: k, Field: f.Name, Before: b[f.Name], After: a[f.Name]})
					}
				}
			}
		}
	}
	sortCells(d.Added)
	sortCells(d.Removed)
	sortCells(d.Changed)
	return d
}

func sortCells(cells []Cell) {
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].Data != cells[j].Data {
			return cells[i].Data < cells[j].Data
		}
		if cells[i].Key != cells[j].Key {
			return cells[i].Key < cells[j].Key
		}
		return cells[i].Field < cells[j].Field
	})
}

func valueEqual(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}

func numeric(v any) (float64, bool) {
	f, ok := v.(float64)
	return f, ok
}

func evaluate(inv Invariant, actual any) bool {
	switch inv.Op {
	case "eq":
		return valueEqual(inv.Value, actual)
	case "ne":
		return !valueEqual(inv.Value, actual)
	case "exists":
		return actual != nil
	case "absent":
		return actual == nil
	case "gt", "lt":
		af, aok := numeric(actual)
		bf, bok := numeric(inv.Value)
		if aok && bok {
			if inv.Op == "gt" {
				return af > bf
			}
			return af < bf
		}
		as, _ := actual.(string)
		bs, _ := inv.Value.(string)
		if inv.Op == "gt" {
			return as > bs
		}
		return as < bs
	}
	return false
}

func analyzeScenario(s Scenario) ScenarioResult {
	r := ScenarioResult{Scenario: s.ID, Status: "passed", Steps: []StepResult{}, Presentation: []string{}}
	type ordered struct {
		id  string
		ord int
	}
	pres := []ordered{}
	for _, step := range s.Steps {
		res := StepResult{Step: step.ID, Status: "passed", Results: []ExpectationResult{}, Findings: []Finding{}}
		observed := map[string]Invariant{}
		for _, o := range step.Observed {
			observed[o.Data+"\x00"+o.Field+"\x00"+o.Op] = o
		}
		unobserved := map[string]bool{}
		for _, o := range step.Observations {
			if o.Source == "manual" || o.Source == "external" {
				if !o.Observed {
					unobserved[o.Data+"\x00"+o.Field] = true
				}
			}
		}
		failed, unverified := false, false
		for _, exp := range step.Expected {
			er := ExpectationResult{Invariant: exp}
			key := exp.Data + "\x00" + exp.Field + "\x00" + exp.Op
			if o, ok := observed[key]; ok {
				er.Actual = o.Value
				if evaluate(exp, o.Value) {
					er.Status = "passed"
				} else {
					er.Status = "failed"
					failed = true
				}
			} else if unobserved[exp.Data+"\x00"+exp.Field] {
				er.Status = "unverified"
				unverified = true
			} else {
				er.Status = "failed"
				failed = true
				res.Findings = append(res.Findings, Finding{Code: "expected-not-observed", Data: exp.Data, Field: exp.Field, Step: step.ID})
			}
			res.Results = append(res.Results, er)
		}
		sort.Slice(res.Results, func(i, j int) bool {
			a, b := res.Results[i].Invariant, res.Results[j].Invariant
			if a.Data != b.Data {
				return a.Data < b.Data
			}
			if a.Field != b.Field {
				return a.Field < b.Field
			}
			return a.Op < b.Op
		})
		switch {
		case failed:
			res.Status = "failed"
			r.Status = "failed"
		case unverified:
			res.Status = "unverified"
			if r.Status == "passed" {
				r.Status = "unverified"
			}
		}
		r.Steps = append(r.Steps, res)
		pres = append(pres, ordered{id: step.ID, ord: step.PresentationOrder})
	}
	sort.SliceStable(pres, func(i, j int) bool { return pres[i].ord < pres[j].ord })
	for _, p := range pres {
		r.Presentation = append(r.Presentation, p.id)
	}
	return r
}

func sortFindings(f []Finding) {
	sort.Slice(f, func(i, j int) bool {
		key := func(x Finding) string {
			return strings.Join([]string{x.Code, x.Data, x.Field, x.Step, x.Detail}, "\x00")
		}
		return key(f[i]) < key(f[j])
	})
}

// Plan validates the model and data, diffs consecutive snapshots, and checks the
// scenarios. Steps are analysed in their declared order.
func Plan(in Input) Report {
	modelFindings := validateModel(in.Model)
	sortFindings(modelFindings)

	dataFindings := validateSnapshots(in.Model, in.Snapshots)
	sortFindings(dataFindings)

	diffs := []SnapshotDiff{}
	for i := 1; i < len(in.Snapshots); i++ {
		diffs = append(diffs, diffSnapshots(in.Model, in.Snapshots[i-1], in.Snapshots[i]))
	}

	scenarios := []ScenarioResult{}
	summary := Summary{ModelFindings: len(modelFindings), DataFindings: len(dataFindings)}
	for _, s := range in.Scenarios {
		sr := analyzeScenario(s)
		for _, step := range sr.Steps {
			switch step.Status {
			case "passed":
				summary.Passed++
			case "unverified":
				summary.Unverified++
			default:
				summary.Failed++
			}
		}
		scenarios = append(scenarios, sr)
	}

	report := Report{
		Schema:        Schema,
		ModelFindings: modelFindings,
		DataFindings:  dataFindings,
		SnapshotDiffs: diffs,
		Scenarios:     scenarios,
		Summary:       summary,
	}
	report.Fingerprint = fingerprint(scenarios, summary)
	return report
}

// fingerprint is computed over step outcomes keyed by scenario and step ID, so it is
// independent of step array order and presentation order.
func fingerprint(scenarios []ScenarioResult, summary Summary) string {
	type stepOutcome struct {
		Scenario string `json:"scenario"`
		Step     string `json:"step"`
		Status   string `json:"status"`
	}
	outcomes := []stepOutcome{}
	for _, s := range scenarios {
		for _, st := range s.Steps {
			outcomes = append(outcomes, stepOutcome{Scenario: s.Scenario, Step: st.Step, Status: st.Status})
		}
	}
	sort.Slice(outcomes, func(i, j int) bool {
		if outcomes[i].Scenario != outcomes[j].Scenario {
			return outcomes[i].Scenario < outcomes[j].Scenario
		}
		return outcomes[i].Step < outcomes[j].Step
	})
	raw, err := json.Marshal(struct {
		Outcomes []stepOutcome `json:"outcomes"`
		Summary  Summary       `json:"summary"`
	}{outcomes, summary})
	if err != nil {
		raw = []byte(fmt.Sprintf("%v", outcomes))
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("sha256:%x", sum)
}
