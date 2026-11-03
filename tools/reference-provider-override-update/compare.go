// Package provideroverrideupdate is a read-only reference for #2091.
//
// It compares a provider definition at two versions (base and next) with the
// overrides and additions a case keeps against the base version, and classifies
// each element as unchanged, applicable, conflict, or unresolved. It never
// writes a definition, never applies an update on its own, and never decides
// who may accept the result.
package provideroverrideupdate

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

const (
	InputSchema  = "tei.reference.provider-override-update-input/v0"
	ReportSchema = "tei.reference.provider-override-update-report/v0"

	// ClassificationUnchanged: no semantic change that touches the case.
	ClassificationUnchanged = "unchanged"
	// ClassificationApplicable: the provider change does not touch the case's
	// overrides or additions, so the update can be proposed.
	ClassificationApplicable = "applicable"
	// ClassificationConflict: the provider change touches a field the case
	// overrides, removes an overridden element, or collides with a name.
	ClassificationConflict = "conflict"
	// ClassificationUnresolved: the comparison lacks information, for example
	// a version mismatch or an override that names a missing element.
	ClassificationUnresolved = "unresolved"

	ProviderUnchanged = "unchanged"
	ProviderChanged   = "changed"
	ProviderAdded     = "added"
	ProviderRemoved   = "removed"
	ProviderMissing   = "missing"
)

// Element is one definition item of the provider or of the case.
type Element struct {
	ID     string            `json:"id"`
	Kind   string            `json:"kind"`
	Fields map[string]string `json:"fields"`
}

// Provider is the reused definition at the version the case fixed (Base) and at
// the new version (Next).
type Provider struct {
	ID          string    `json:"id"`
	FromVersion string    `json:"fromVersion"`
	ToVersion   string    `json:"toVersion"`
	Base        []Element `json:"base"`
	Next        []Element `json:"next"`
}

// Override is a case-side field value kept against the base provider version.
type Override struct {
	Element string            `json:"element"`
	Fields  map[string]string `json:"fields"`
}

// Case is what the case added or changed against the base version.
type Case struct {
	BaseVersion string     `json:"baseVersion"`
	Overrides   []Override `json:"overrides"`
	Additions   []Element  `json:"additions"`
}

// Input is the model the comparison reads.
type Input struct {
	Schema            string   `json:"schema"`
	Provider          Provider `json:"provider"`
	Case              Case     `json:"case"`
	DisplayOnlyFields []string `json:"displayOnlyFields,omitempty"`
}

// ElementResult is the classification for one canonical element identifier.
type ElementResult struct {
	ID             string   `json:"id"`
	Classification string   `json:"classification"`
	ProviderChange string   `json:"providerChange"`
	ChangedFields  []string `json:"changedFields,omitempty"`
	OverrideFields []string `json:"overrideFields,omitempty"`
	Reasons        []string `json:"reasons,omitempty"`
}

// NameCollision is a case addition whose identifier the provider also defines.
type NameCollision struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

// State is a normalized snapshot of one side of the update.
type State struct {
	ProviderVersion string    `json:"providerVersion"`
	BaseVersion     string    `json:"baseVersion"`
	Elements        []Element `json:"elements"`
}

// Report is the comparison. Status is the worst classification found.
type Report struct {
	Schema         string          `json:"schema"`
	Status         string          `json:"status"`
	ProviderID     string          `json:"providerId"`
	FromVersion    string          `json:"fromVersion"`
	ToVersion      string          `json:"toVersion"`
	Elements       []ElementResult `json:"elements"`
	NameCollisions []NameCollision `json:"nameCollisions,omitempty"`
	Problems       []string        `json:"problems,omitempty"`
	Blocked        bool            `json:"blocked"`
	Note           string          `json:"note"`
}

// Result carries the report and the two states. After is nil when the update is
// blocked. Rollback returns Before and never changes the input.
type Result struct {
	Report Report `json:"report"`
	Before State  `json:"before"`
	After  *State `json:"after,omitempty"`
}

// LoadInput reads an input file.
func LoadInput(path string) (Input, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Input{}, err
	}
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		return Input{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return input, nil
}

// Plan compares the two provider versions with the case's overrides and
// additions. A blocked plan has no After state, so a caller cannot apply it by
// accident.
func Plan(input Input) Result {
	report := Report{
		Schema:      ReportSchema,
		ProviderID:  input.Provider.ID,
		FromVersion: input.Provider.FromVersion,
		ToVersion:   input.Provider.ToVersion,
		Note:        "読取り専用の比較です。適用・受理は行いません。conflictとunresolvedではAfterを返しません。",
	}

	var problems []string
	if input.Schema != InputSchema {
		problems = append(problems, fmt.Sprintf("schema is %q, want %q", input.Schema, InputSchema))
	}
	if input.Provider.ID == "" {
		problems = append(problems, "provider id is required")
	}
	if input.Provider.FromVersion == "" || input.Provider.ToVersion == "" {
		problems = append(problems, "provider fromVersion and toVersion are required")
	}
	if input.Case.BaseVersion != input.Provider.FromVersion {
		problems = append(problems, fmt.Sprintf("case baseVersion %q does not match provider fromVersion %q", input.Case.BaseVersion, input.Provider.FromVersion))
	}

	base, baseProblems := elementMap("provider base", input.Provider.Base)
	next, nextProblems := elementMap("provider next", input.Provider.Next)
	problems = append(problems, baseProblems...)
	problems = append(problems, nextProblems...)

	additions, additionProblems := elementMap("case additions", input.Case.Additions)
	problems = append(problems, additionProblems...)

	displayOnly := make(map[string]bool, len(input.DisplayOnlyFields))
	for _, field := range input.DisplayOnlyFields {
		displayOnly[field] = true
	}

	overridesByElement := map[string]map[string]string{}
	for _, override := range input.Case.Overrides {
		if override.Element == "" {
			problems = append(problems, "override with empty element id")
			continue
		}
		target, exists := base[override.Element]
		if !exists {
			problems = append(problems, fmt.Sprintf("override target %q is not in the provider base", override.Element))
			continue
		}
		if len(override.Fields) == 0 {
			problems = append(problems, fmt.Sprintf("override for %q has no fields", override.Element))
			continue
		}
		accumulated := overridesByElement[override.Element]
		if accumulated == nil {
			accumulated = map[string]string{}
			overridesByElement[override.Element] = accumulated
		}
		for field := range override.Fields {
			if _, exists := target.Fields[field]; !exists {
				problems = append(problems, fmt.Sprintf("override for %q names missing field %q", override.Element, field))
				continue
			}
			accumulated[field] = override.Fields[field]
		}
	}

	ids := map[string]bool{}
	for id := range base {
		ids[id] = true
	}
	for id := range next {
		ids[id] = true
	}
	for id := range additions {
		ids[id] = true
	}
	for id := range overridesByElement {
		ids[id] = true
	}

	var collisions []NameCollision
	for id := range additions {
		if _, exists := next[id]; exists {
			if _, existedInBase := base[id]; existedInBase {
				collisions = append(collisions, NameCollision{ID: id, Kind: "case-addition-vs-provider-element"})
			} else {
				collisions = append(collisions, NameCollision{ID: id, Kind: "case-addition-vs-provider-addition"})
			}
			continue
		}
		if _, exists := base[id]; exists {
			collisions = append(collisions, NameCollision{ID: id, Kind: "case-addition-vs-provider-element"})
		}
	}
	sort.Slice(collisions, func(i, j int) bool { return collisions[i].ID < collisions[j].ID })
	report.NameCollisions = collisions

	var results []ElementResult
	for id := range ids {
		result := classify(id, base[id], next[id], additions[id], overridesByElement[id], displayOnly, collisions)
		results = append(results, result)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].ID < results[j].ID })
	report.Elements = results

	if len(problems) > 0 {
		report.Problems = problems
		report.Status = ClassificationUnresolved
		report.Blocked = true
		return Result{
			Report: report,
			Before: buildState(input.Provider.FromVersion, input.Case.BaseVersion, input.Provider.Base, overridesByElement, input.Case.Additions),
		}
	}

	report.Status = worstStatus(results)
	report.Blocked = report.Status == ClassificationConflict || report.Status == ClassificationUnresolved
	result := Result{
		Report: report,
		Before: buildState(input.Provider.FromVersion, input.Case.BaseVersion, input.Provider.Base, overridesByElement, input.Case.Additions),
	}
	if !report.Blocked {
		after := buildState(input.Provider.ToVersion, input.Provider.ToVersion, input.Provider.Next, overridesByElement, input.Case.Additions)
		result.After = &after
	}
	return result
}

func classify(id string, baseElement, nextElement *Element, addition *Element, overrides map[string]string, displayOnly map[string]bool, collisions []NameCollision) ElementResult {
	result := ElementResult{ID: id}
	result.OverrideFields = sortedKeys(overrides)

	if addition != nil {
		for _, collision := range collisions {
			if collision.ID == id {
				result.Classification = ClassificationConflict
				result.ProviderChange = ProviderAdded
				result.Reasons = append(result.Reasons, collision.Kind)
				return result
			}
		}
		result.Classification = ClassificationUnchanged
		result.ProviderChange = ProviderUnchanged
		result.Reasons = append(result.Reasons, "case-only-addition")
		return result
	}

	switch {
	case baseElement == nil && nextElement == nil:
		result.Classification = ClassificationUnresolved
		result.ProviderChange = ProviderMissing
		result.Reasons = append(result.Reasons, "element-not-in-provider")
		return result
	case baseElement == nil:
		result.Classification = ClassificationApplicable
		result.ProviderChange = ProviderAdded
		result.Reasons = append(result.Reasons, "provider-added-element")
		return result
	case nextElement == nil:
		result.ProviderChange = ProviderRemoved
		if len(overrides) > 0 {
			result.Classification = ClassificationConflict
			result.Reasons = append(result.Reasons, "provider-removed-overridden-element")
			return result
		}
		result.Classification = ClassificationApplicable
		result.Reasons = append(result.Reasons, "provider-removed-element")
		return result
	}

	changed := changedFields(baseElement.Fields, nextElement.Fields)
	result.ChangedFields = changed
	if len(changed) == 0 {
		result.Classification = ClassificationUnchanged
		result.ProviderChange = ProviderUnchanged
		result.Reasons = append(result.Reasons, "provider-unchanged-element")
		return result
	}

	result.ProviderChange = ProviderChanged
	semantic := make([]string, 0, len(changed))
	for _, field := range changed {
		if !displayOnly[field] {
			semantic = append(semantic, field)
		}
	}
	if len(semantic) == 0 {
		result.Classification = ClassificationUnchanged
		result.Reasons = append(result.Reasons, "display-only-change")
		return result
	}

	for _, field := range semantic {
		if _, exists := overrides[field]; exists {
			result.Classification = ClassificationConflict
			result.Reasons = append(result.Reasons, fmt.Sprintf("provider-and-case-changed-field:%s", field))
			return result
		}
	}
	result.Classification = ClassificationApplicable
	if len(overrides) > 0 {
		result.Reasons = append(result.Reasons, "override-kept-on-other-field")
	} else {
		result.Reasons = append(result.Reasons, "provider-changed-element")
	}
	return result
}

func changedFields(base, next map[string]string) []string {
	seen := map[string]bool{}
	for field := range base {
		seen[field] = true
	}
	for field := range next {
		seen[field] = true
	}
	var changed []string
	for field := range seen {
		if base[field] != next[field] {
			changed = append(changed, field)
		}
	}
	sort.Strings(changed)
	return changed
}

func worstStatus(results []ElementResult) string {
	rank := map[string]int{
		ClassificationUnchanged:  0,
		ClassificationApplicable: 1,
		ClassificationConflict:   2,
		ClassificationUnresolved: 3,
	}
	worst := ClassificationUnchanged
	for _, result := range results {
		if rank[result.Classification] > rank[worst] {
			worst = result.Classification
		}
	}
	return worst
}

func buildState(providerVersion, baseVersion string, elements []Element, overrides map[string]map[string]string, additions []Element) State {
	state := State{ProviderVersion: providerVersion, BaseVersion: baseVersion}
	seen := map[string]bool{}
	for _, element := range elements {
		if seen[element.ID] {
			continue
		}
		seen[element.ID] = true
		state.Elements = append(state.Elements, applyOverrides(element, overrides[element.ID]))
	}
	for _, addition := range additions {
		if seen[addition.ID] {
			continue
		}
		seen[addition.ID] = true
		state.Elements = append(state.Elements, applyOverrides(addition, nil))
	}
	sort.Slice(state.Elements, func(i, j int) bool { return state.Elements[i].ID < state.Elements[j].ID })
	return state
}

func applyOverrides(element Element, overrides map[string]string) Element {
	if len(overrides) == 0 {
		return element
	}
	fields := make(map[string]string, len(element.Fields))
	for field, value := range element.Fields {
		fields[field] = value
	}
	for field, value := range overrides {
		fields[field] = value
	}
	return Element{ID: element.ID, Kind: element.Kind, Fields: fields}
}

// Rollback returns the pre-update state. It is a pure function of the plan and
// never changes the input.
func Rollback(result Result) State {
	return result.Before
}

func elementMap(label string, elements []Element) (map[string]*Element, []string) {
	out := make(map[string]*Element, len(elements))
	var problems []string
	for index := range elements {
		element := elements[index]
		if element.ID == "" {
			problems = append(problems, fmt.Sprintf("%s has an element with empty id", label))
			continue
		}
		if _, exists := out[element.ID]; exists {
			problems = append(problems, fmt.Sprintf("%s has duplicate element id %q", label, element.ID))
			continue
		}
		copied := element
		out[element.ID] = &copied
	}
	return out, problems
}

func sortedKeys(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
