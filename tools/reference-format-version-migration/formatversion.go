// Package formatversion migrates a definition between format versions through a single
// conversion table, so each reader only handles the current version. It never reads a
// definition without a recorded format version as an implicit default, never treats a
// content version as a format version, raises the version value only on a successful
// conversion, and reports every removed or dropped attribute instead of skipping it.
// It is read-only: it does not write a Source.
package formatversion

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const Schema = "tei.reference.format-version-migration/v0"

// Attr is one attribute value. Value is nil when the attribute is absent, an empty
// string when it is present and empty, and a value when it is present.
type Attr struct {
	Name  string  `json:"name"`
	Value *string `json:"value"`
}

// Definition is one definition with its declared format version.
type Definition struct {
	ID string `json:"id"`
	// Version is the definition format version. An empty string means it is not recorded.
	Version string `json:"version"`
	// Kind is "format" for a definition format version. "content" marks a content
	// version that must not be used as a format version.
	Kind string `json:"kind"`
	// ContentVersion is a separate content version, when the definition declares one.
	ContentVersion string `json:"contentVersion"`
	Attrs          []Attr `json:"attrs"`
}

// SetDefault writes an implicit default of an older version as an explicit value.
type SetDefault struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Rename carries a value from an old attribute name to a new one.
type Rename struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Rule is one conversion step from one format version to the next.
type Rule struct {
	From       string       `json:"from"`
	To         string       `json:"to"`
	Renames    []Rename     `json:"renames"`
	SetDefault []SetDefault `json:"setDefaults"`
	// Removed lists attributes that stay defined but are no longer effective. Their
	// presence is reported as a warning, never skipped silently.
	Removed []string `json:"removed"`
	// DropBySilently is an anti-pattern fixture: a rule that would drop an attribute
	// without recording it. The reference must reject it.
	DropBySilently []string `json:"dropBySilently"`
}

// Input is one read-only migration request.
type Input struct {
	// Current is the format version the core handles.
	Current string `json:"current"`
	Rules   []Rule `json:"rules"`
	// NeedRoundTrip marks that a conversion back to an older version is required.
	NeedRoundTrip bool         `json:"needRoundTrip"`
	Documents     []Definition `json:"documents"`
}

// Item is one warning or failure with the element and attribute it concerns.
type Item struct {
	Code      string `json:"code"`
	Element   string `json:"element"`
	Attribute string `json:"attribute,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

// Outcome is the migration result of one definition.
type Outcome struct {
	Document      string   `json:"document"`
	Status        string   `json:"status"` // success | success-with-warning | failure
	From          string   `json:"from"`
	To            string   `json:"to"`
	VersionRaised bool     `json:"versionRaised"`
	Path          []string `json:"path"`
	Attrs         []Attr   `json:"attrs"`
	Warnings      []Item   `json:"warnings"`
	Failures      []Item   `json:"failures"`
}

// Strategy is the chosen conversion mode, with the reason.
type Strategy struct {
	Mode   string `json:"mode"` // chain | hub
	Reason string `json:"reason"`
}

// Summary counts the outcomes.
type Summary struct {
	Total    int `json:"total"`
	Success  int `json:"success"`
	Warnings int `json:"warnings"`
	Failures int `json:"failures"`
}

// Report is the read-only result.
type Report struct {
	Schema      string    `json:"schema"`
	Current     string    `json:"current"`
	Strategy    Strategy  `json:"strategy"`
	Versions    []string  `json:"versions"`
	Outcomes    []Outcome `json:"outcomes"`
	Summary     Summary   `json:"summary"`
	Fingerprint string    `json:"fingerprint"`
}

func compareVersions(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		av, bv := 0, 0
		if i < len(as) {
			av, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bv, _ = strconv.Atoi(bs[i])
		}
		if av != bv {
			if av < bv {
				return -1
			}
			return 1
		}
	}
	return 0
}

// PlanStrategy decides between a stepwise chain and a hub representation from the number
// of versions and whether a round-trip is required, and records the reason.
func PlanStrategy(versionCount int, needRoundTrip bool) Strategy {
	if needRoundTrip {
		return Strategy{Mode: "hub", Reason: "往復が必要なため、中心の表現を経由する方式を既定にする"}
	}
	if versionCount <= 3 {
		return Strategy{Mode: "chain", Reason: "版が少なく往復が不要なため、一段ずつ変換する方式を既定にする"}
	}
	return Strategy{Mode: "hub", Reason: "版が多く組み合わせが増えるため、中心の表現を経由する方式を既定にする"}
}

func attrMap(attrs []Attr) map[string]*string {
	m := make(map[string]*string, len(attrs))
	for _, a := range attrs {
		m[a.Name] = copyValue(a.Value)
	}
	return m
}

func copyValue(v *string) *string {
	if v == nil {
		return nil
	}
	s := *v
	return &s
}

func attrsFromMap(m map[string]*string) []Attr {
	out := make([]Attr, 0, len(m))
	for k, v := range m {
		out = append(out, Attr{Name: k, Value: copyValue(v)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func cloneAttrs(attrs []Attr) []Attr {
	return attrsFromMap(attrMap(attrs))
}

func findPath(rules []Rule, from, to string) ([]Rule, []string, bool) {
	adj := map[string][]Rule{}
	for _, r := range rules {
		adj[r.From] = append(adj[r.From], r)
	}
	type node struct {
		v        string
		path     []Rule
		versions []string
	}
	queue := []node{{v: from, versions: []string{from}}}
	seen := map[string]bool{from: true}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.v == to {
			return cur.path, cur.versions, true
		}
		next := append([]Rule(nil), adj[cur.v]...)
		sort.Slice(next, func(i, j int) bool { return next[i].To < next[j].To })
		for _, r := range next {
			if seen[r.To] {
				continue
			}
			seen[r.To] = true
			queue = append(queue, node{
				v:        r.To,
				path:     append(append([]Rule(nil), cur.path...), r),
				versions: append(append([]string(nil), cur.versions...), r.To),
			})
		}
	}
	return nil, nil, false
}

func applyPath(d Definition, path []Rule) ([]Attr, []Item, []Item, []string) {
	m := attrMap(d.Attrs)
	warnings := []Item{}
	failures := []Item{}
	versions := []string{d.Version}
	for _, r := range path {
		step := r.From + "->" + r.To
		for _, rn := range r.Renames {
			if v, ok := m[rn.From]; ok {
				m[rn.To] = v
				delete(m, rn.From)
			}
		}
		for _, name := range r.DropBySilently {
			if _, ok := m[name]; ok {
				failures = append(failures, Item{Code: "attribute-silently-dropped", Element: d.ID, Attribute: name, Detail: step})
				delete(m, name)
			}
		}
		for _, name := range r.Removed {
			if _, ok := m[name]; ok {
				warnings = append(warnings, Item{Code: "attribute-removed", Element: d.ID, Attribute: name, Detail: step})
				delete(m, name)
			}
		}
		for _, sd := range r.SetDefault {
			if _, ok := m[sd.Name]; !ok {
				v := sd.Value
				m[sd.Name] = &v
			}
		}
		versions = append(versions, r.To)
	}
	return attrsFromMap(m), warnings, failures, versions
}

func sortItems(items []Item) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Code != items[j].Code {
			return items[i].Code < items[j].Code
		}
		if items[i].Attribute != items[j].Attribute {
			return items[i].Attribute < items[j].Attribute
		}
		return items[i].Detail < items[j].Detail
	})
}

func migrateOne(in Input, d Definition) Outcome {
	o := Outcome{
		Document: d.ID,
		From:     d.Version,
		To:       d.Version,
		Attrs:    cloneAttrs(d.Attrs),
		Path:     []string{d.Version},
	}
	fail := func(code, attribute, detail string) Outcome {
		o.Status = "failure"
		o.Failures = []Item{{Code: code, Element: d.ID, Attribute: attribute, Detail: detail}}
		return o
	}

	switch {
	case d.Version == "":
		return fail("missing-format-version", "", "形式の版が記録されていない")
	case d.Kind == "content":
		return fail("content-version-not-format", "", "内容の版を形式の版として扱っている")
	case d.ContentVersion != "" && d.ContentVersion != d.Version:
		return fail("version-content-mismatch", "", d.Version+" と "+d.ContentVersion)
	case compareVersions(d.Version, in.Current) > 0:
		return fail("unsupported-newer-version", "", "必要な道具の版: "+d.Version)
	}

	if d.Version == in.Current {
		o.Status = "success"
		return o
	}

	path, versions, ok := findPath(in.Rules, d.Version, in.Current)
	if !ok {
		return fail("no-conversion-path", "", d.Version+" -> "+in.Current)
	}
	attrs, warnings, failures, _ := applyPath(d, path)
	if len(failures) > 0 {
		o.Status = "failure"
		o.Failures = failures
		sortItems(o.Failures)
		return o
	}
	o.Attrs = attrs
	o.Warnings = warnings
	o.Path = versions
	o.To = in.Current
	o.VersionRaised = true
	if len(warnings) > 0 {
		o.Status = "success-with-warning"
	} else {
		o.Status = "success"
	}
	sortItems(o.Warnings)
	return o
}

func collectVersions(in Input) []string {
	set := map[string]bool{in.Current: true}
	for _, r := range in.Rules {
		set[r.From] = true
		set[r.To] = true
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// Plan migrates every definition through the single conversion table.
func Plan(in Input) Report {
	docs := append([]Definition(nil), in.Documents...)
	sort.Slice(docs, func(i, j int) bool { return docs[i].ID < docs[j].ID })

	outcomes := make([]Outcome, 0, len(docs))
	summary := Summary{Total: len(docs)}
	for _, d := range docs {
		o := migrateOne(in, d)
		outcomes = append(outcomes, o)
		switch o.Status {
		case "success":
			summary.Success++
		case "success-with-warning":
			summary.Warnings++
		default:
			summary.Failures++
		}
	}

	versions := collectVersions(in)
	return Report{
		Schema:      Schema,
		Current:     in.Current,
		Strategy:    PlanStrategy(len(versions), in.NeedRoundTrip),
		Versions:    versions,
		Outcomes:    outcomes,
		Summary:     summary,
		Fingerprint: fingerprint(outcomes),
	}
}

func fingerprint(outcomes []Outcome) string {
	raw, err := json.Marshal(outcomes)
	if err != nil {
		raw = []byte(fmt.Sprintf("%v", outcomes))
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("sha256:%x", sum)
}
