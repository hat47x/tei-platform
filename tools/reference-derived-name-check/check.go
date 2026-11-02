// Package derivednamecheck is a read-only reference for #2094. It compares the
// names of definition elements, and the derived names a generator or provider
// would build from them, under rules the layers declare. It reports collisions
// and single-element rule violations before a definition is accepted.
//
// The rules in testdata are reference-local examples. They are not the
// declarations of any real provider or generator, and the check never changes
// an element identifier or proposes a new name.
package derivednamecheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

const (
	RulesSchema  = "tei.reference.derived-name-rules/v0"
	NamesSchema  = "tei.reference.derived-name-input/v0"
	ReportSchema = "tei.reference.derived-name-report/v0"
)

// Derivation is a name a layer builds from an element name: Prefix + name + Suffix.
type Derivation struct {
	ID     string `json:"id"`
	Prefix string `json:"prefix,omitempty"`
	Suffix string `json:"suffix,omitempty"`
	// SkipLength leaves the derived name out of the MaxBytes check, for a
	// layer that shortens the name itself. Collisions are still compared on
	// the full name, so a shortened-name collision is not evaluated.
	SkipLength bool `json:"skipLength,omitempty"`
}

// Layer is one place that turns a name into a physical or generated name.
type Layer struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	// Target is the realization target the layer belongs to. A layer is
	// evaluated only when its target is selected.
	Target string `json:"target"`
	// FoldCase compares names without regard to upper and lower case.
	FoldCase bool `json:"foldCase,omitempty"`
	// IgnoreChars lists characters removed before comparison.
	IgnoreChars string `json:"ignoreChars,omitempty"`
	// MaxBytes limits the UTF-8 length of a name. Zero means no limit.
	MaxBytes int `json:"maxBytes,omitempty"`
	// Pattern is a regular expression every element name must match.
	Pattern string `json:"pattern,omitempty"`
	// Reserved lists names that may not be used, compared after normalization.
	Reserved    []string     `json:"reserved,omitempty"`
	Derivations []Derivation `json:"derivations,omitempty"`
}

type Rules struct {
	Schema  string  `json:"schema"`
	ID      string  `json:"id"`
	Version string  `json:"version"`
	Layers  []Layer `json:"layers"`
}

// Element is a definition element. ID is the canonical identifier; Name is the
// label the rules are applied to. The check never uses Name as an identity.
type Element struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Input lists the elements. Renames give the name an element would have after
// a rename (#1935); the same rules run on the renamed names.
type Input struct {
	Schema   string    `json:"schema"`
	Elements []Element `json:"elements"`
	Renames  []Element `json:"renames,omitempty"`
}

const (
	KindCollision    = "collision"
	KindReserved     = "reserved"
	KindTooLong      = "too-long"
	KindInvalidChars = "invalid-chars"
)

const (
	StatusUnevaluated = "unevaluated"
	StatusPassed      = "passed"
	StatusRejected    = "rejected"
)

// Occurrence is one name that took part in a violation.
type Occurrence struct {
	ElementID string `json:"elementId"`
	Name      string `json:"name"`
	// Derivation is empty for the element's own name.
	Derivation string `json:"derivation,omitempty"`
	Derived    string `json:"derivedName"`
}

type Violation struct {
	Kind         string       `json:"kind"`
	Layer        string       `json:"layer"`
	LayerVersion string       `json:"layerVersion"`
	Target       string       `json:"target"`
	Rule         string       `json:"rule"`
	Occurrences  []Occurrence `json:"occurrences"`
}

type Report struct {
	Schema       string      `json:"schema"`
	RulesID      string      `json:"rulesId"`
	RulesVersion string      `json:"rulesVersion"`
	Targets      []string    `json:"targets"`
	Status       string      `json:"status"`
	Evaluated    []string    `json:"evaluatedLayers"`
	Violations   []Violation `json:"violations"`
	Renamed      []string    `json:"renamedElements,omitempty"`
	Note         string      `json:"note"`
}

const reportNote = "要素の識別子は変更しない。名前の自動変更や代替名の提案はしない。" +
	"宣言された規則と版での検査結果であり、実在のProvider・生成器の規則との一致は確認していない。"

// LoadRules reads and validates a rules file.
func LoadRules(path string) (Rules, error) {
	var rules Rules
	if err := readJSON(path, &rules); err != nil {
		return rules, err
	}
	return rules, rules.Validate()
}

// LoadInput reads and validates an input file.
func LoadInput(path string) (Input, error) {
	var in Input
	if err := readJSON(path, &in); err != nil {
		return in, err
	}
	return in, in.Validate()
}

func readJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if decoder.More() {
		return fmt.Errorf("%s: trailing JSON", path)
	}
	return nil
}

// Validate rejects a rule set whose own declarations would make a comparison
// ambiguous.
func (r Rules) Validate() error {
	if r.Schema != RulesSchema {
		return fmt.Errorf("rules schema must be %s", RulesSchema)
	}
	if r.ID == "" || r.Version == "" {
		return fmt.Errorf("rules id and version are required")
	}
	if len(r.Layers) == 0 {
		return fmt.Errorf("rules have no layers")
	}
	seen := map[string]bool{}
	for _, layer := range r.Layers {
		if layer.ID == "" || layer.Version == "" || layer.Target == "" {
			return fmt.Errorf("layer id, version, and target are required")
		}
		if seen[layer.ID] {
			return fmt.Errorf("duplicate layer %q", layer.ID)
		}
		seen[layer.ID] = true
		if layer.MaxBytes < 0 {
			return fmt.Errorf("layer %q: maxBytes must not be negative", layer.ID)
		}
		if layer.Pattern != "" {
			if _, err := regexp.Compile(layer.Pattern); err != nil {
				return fmt.Errorf("layer %q: pattern: %w", layer.ID, err)
			}
		}
		derivations := map[string]bool{}
		forms := map[string]string{}
		for _, d := range layer.Derivations {
			if d.ID == "" || (d.Prefix == "" && d.Suffix == "") {
				return fmt.Errorf("layer %q: a derivation needs an id and a prefix or suffix", layer.ID)
			}
			if derivations[d.ID] {
				return fmt.Errorf("layer %q: duplicate derivation %q", layer.ID, d.ID)
			}
			derivations[d.ID] = true
			form := layer.normalize(d.Prefix + "\x00" + d.Suffix)
			if other, dup := forms[form]; dup {
				return fmt.Errorf("layer %q: derivations %q and %q build the same name", layer.ID, other, d.ID)
			}
			forms[form] = d.ID
		}
	}
	return nil
}

// Validate rejects an input that cannot be evaluated.
func (in Input) Validate() error {
	if in.Schema != NamesSchema {
		return fmt.Errorf("input schema must be %s", NamesSchema)
	}
	ids := map[string]bool{}
	for _, e := range in.Elements {
		if e.ID == "" || e.Name == "" {
			return fmt.Errorf("element id and name are required")
		}
		if ids[e.ID] {
			return fmt.Errorf("duplicate element id %q", e.ID)
		}
		ids[e.ID] = true
	}
	renamed := map[string]bool{}
	for _, e := range in.Renames {
		if e.Name == "" || !ids[e.ID] {
			return fmt.Errorf("rename must name an existing element and a non-empty name: %q", e.ID)
		}
		if renamed[e.ID] {
			return fmt.Errorf("duplicate rename for %q", e.ID)
		}
		renamed[e.ID] = true
	}
	return nil
}

func (l Layer) normalize(s string) string {
	if l.FoldCase {
		s = strings.ToLower(s)
	}
	if l.IgnoreChars != "" {
		s = strings.Map(func(r rune) rune {
			if strings.ContainsRune(l.IgnoreChars, r) {
				return -1
			}
			return r
		}, s)
	}
	return s
}

// Check evaluates the layers of the selected targets. With no target the
// result is "unevaluated": conformance to any realization target is unknown,
// which is not a pass.
func Check(rules Rules, in Input, targets []string) (Report, error) {
	if err := rules.Validate(); err != nil {
		return Report{}, err
	}
	if err := in.Validate(); err != nil {
		return Report{}, err
	}
	report := Report{
		Schema:       ReportSchema,
		RulesID:      rules.ID,
		RulesVersion: rules.Version,
		Targets:      append([]string{}, targets...),
		Evaluated:    []string{},
		Violations:   []Violation{},
		Note:         reportNote,
	}
	sort.Strings(report.Targets)
	if len(targets) == 0 {
		report.Status = StatusUnevaluated
		return report, nil
	}
	selected := map[string]bool{}
	for _, t := range targets {
		selected[t] = true
	}
	known := map[string]bool{}
	for _, layer := range rules.Layers {
		known[layer.Target] = true
	}
	for t := range selected {
		if !known[t] {
			return report, fmt.Errorf("no layer declares target %q", t)
		}
	}

	elements := append([]Element{}, in.Elements...)
	renamed := map[string]string{}
	for _, r := range in.Renames {
		renamed[r.ID] = r.Name
	}
	for i := range elements {
		if name, ok := renamed[elements[i].ID]; ok {
			elements[i].Name = name
			report.Renamed = append(report.Renamed, elements[i].ID)
		}
	}
	sort.Strings(report.Renamed)

	for _, layer := range rules.Layers {
		if !selected[layer.Target] {
			continue
		}
		report.Evaluated = append(report.Evaluated, layer.ID)
		report.Violations = append(report.Violations, checkLayer(layer, elements)...)
	}
	sort.SliceStable(report.Violations, func(i, j int) bool {
		a, b := report.Violations[i], report.Violations[j]
		if a.Layer != b.Layer {
			return a.Layer < b.Layer
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Occurrences[0].ElementID < b.Occurrences[0].ElementID
	})
	if len(report.Violations) == 0 {
		report.Status = StatusPassed
	} else {
		report.Status = StatusRejected
	}
	return report, nil
}

func checkLayer(layer Layer, elements []Element) []Violation {
	var out []Violation
	newViolation := func(kind, rule string, occ ...Occurrence) Violation {
		return Violation{
			Kind: kind, Layer: layer.ID, LayerVersion: layer.Version,
			Target: layer.Target, Rule: rule, Occurrences: occ,
		}
	}
	var pattern *regexp.Regexp
	if layer.Pattern != "" {
		pattern = regexp.MustCompile(layer.Pattern)
	}
	reserved := map[string]bool{}
	for _, word := range layer.Reserved {
		reserved[layer.normalize(word)] = true
	}

	type form struct {
		occ  Occurrence
		norm string
	}
	byNorm := map[string][]Occurrence{}
	for _, e := range elements {
		forms := []form{{Occurrence{ElementID: e.ID, Name: e.Name, Derived: e.Name}, layer.normalize(e.Name)}}
		for _, d := range layer.Derivations {
			derived := d.Prefix + e.Name + d.Suffix
			forms = append(forms, form{Occurrence{ElementID: e.ID, Name: e.Name, Derivation: d.ID, Derived: derived}, layer.normalize(derived)})
		}
		if pattern != nil && !pattern.MatchString(e.Name) {
			out = append(out, newViolation(KindInvalidChars, "pattern "+layer.Pattern, forms[0].occ))
		}
		for i, f := range forms {
			skipLength := i > 0 && layer.Derivations[i-1].SkipLength
			if layer.MaxBytes > 0 && !skipLength && len(f.occ.Derived) > layer.MaxBytes {
				out = append(out, newViolation(KindTooLong, fmt.Sprintf("maxBytes %d", layer.MaxBytes), f.occ))
			}
			if reserved[f.norm] {
				out = append(out, newViolation(KindReserved, "reserved word", f.occ))
			}
			byNorm[f.norm] = append(byNorm[f.norm], f.occ)
		}
	}
	norms := make([]string, 0, len(byNorm))
	for n := range byNorm {
		norms = append(norms, n)
	}
	sort.Strings(norms)
	for _, n := range norms {
		occs := byNorm[n]
		distinct := map[string]bool{}
		for _, o := range occs {
			distinct[o.ElementID] = true
		}
		if len(distinct) < 2 {
			continue
		}
		out = append(out, newViolation(KindCollision, collisionRule(layer), occs...))
	}
	return out
}

func collisionRule(layer Layer) string {
	var parts []string
	if layer.FoldCase {
		parts = append(parts, "case-insensitive")
	} else {
		parts = append(parts, "case-sensitive")
	}
	if layer.IgnoreChars != "" {
		parts = append(parts, fmt.Sprintf("ignoring %q", layer.IgnoreChars))
	}
	return "equal names, " + strings.Join(parts, ", ")
}
