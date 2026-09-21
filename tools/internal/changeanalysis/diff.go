// Package changeanalysisは、二つのCanonical Worldの間のread-onlyな差分と、
// その差分から辿れるexact identityに基づく依存影響を計算する。
//
// authoritative sourceを読み書きせず、採用判断もしない。差分・影響の解析（ISSUE-000039）と、
// 受理・source反映（ISSUE-000027）を混ぜないための境界であり、Reference専用のtool内部packageである。
package changeanalysis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// SubjectChangeは、Canonical Worldのsubject単位の差分である。
// file pathやsource上の順序は含まない。
type SubjectChange struct {
	Kind   string          `json:"kind"`
	ID     string          `json:"id"`
	Change string          `json:"change"`
	Before json.RawMessage `json:"before,omitempty"`
	After  json.RawMessage `json:"after,omitempty"`
}

// Keyは"kind:id"形式のsubject keyを返す。
func (c SubjectChange) Key() string { return c.Kind + ":" + c.ID }

type subjectValue struct {
	kind string
	id   string
	raw  json.RawMessage
}

type subjectSet map[string]subjectValue

func (set subjectSet) add(kind string, id core.CanonicalID, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	key := kind + ":" + string(id)
	if _, exists := set[key]; exists {
		return fmt.Errorf("%s が複数回宣言されています", key)
	}
	set[key] = subjectValue{kind: kind, id: string(id), raw: raw}
	return nil
}

func flattenSemantics(namespaces []semantic.Namespace) (subjectSet, error) {
	out := subjectSet{}
	type identity struct {
		ID     core.CanonicalID `json:"id"`
		Symbol string           `json:"symbol,omitempty"`
	}
	for _, ns := range namespaces {
		if err := out.add("namespace", ns.ID, identity{ID: ns.ID}); err != nil {
			return nil, err
		}
		for _, t := range ns.Types {
			if err := out.add("type", t.ID, t); err != nil {
				return nil, err
			}
		}
		for _, d := range ns.Data {
			if err := out.add("data", d.ID, identity{ID: d.ID, Symbol: d.Symbol}); err != nil {
				return nil, err
			}
			for _, a := range d.Attributes {
				if err := out.add("attribute", a.ID, a); err != nil {
					return nil, err
				}
			}
			for _, r := range d.Relations {
				if err := out.add("relation", r.ID, r); err != nil {
					return nil, err
				}
			}
		}
		for _, c := range ns.Capabilities {
			if err := out.add("capability", c.ID, c); err != nil {
				return nil, err
			}
		}
		for _, v := range ns.Validations {
			if err := out.add("validation", v.ID, v); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// flattenApplicationsは、Applicationの本体とPatternを別のsubjectとして並べる。
// Application本体の比較にはPatternを含めないため、Patternだけの変更でApplication本体まで
// 二重にchangedとは扱わない。datasource aliasとContract requirementはApplication本体の意味に含める。
func flattenApplications(apps []application.Application) (subjectSet, error) {
	out := subjectSet{}
	type applicationIdentity struct {
		ID           core.CanonicalID                  `json:"id"`
		SemanticUses []core.CanonicalID                `json:"semanticUses,omitempty"`
		Validations  []core.CanonicalID                `json:"validations,omitempty"`
		Databases    []application.DatabaseRequirement `json:"databases,omitempty"`
	}
	for _, app := range apps {
		identity := applicationIdentity{ID: app.ID, SemanticUses: app.SemanticUses, Validations: app.Validations, Databases: app.Databases}
		if err := out.add("application", app.ID, identity); err != nil {
			return nil, err
		}
		for _, p := range app.Patterns {
			if err := out.add("application-pattern", p.ID, p); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

func diffSubjects(b, a subjectSet) []SubjectChange {
	var out []SubjectChange
	for key, bv := range b {
		av, ok := a[key]
		switch {
		case !ok:
			out = append(out, SubjectChange{Kind: bv.kind, ID: bv.id, Change: "removed", Before: bv.raw})
		case !bytes.Equal(bv.raw, av.raw):
			out = append(out, SubjectChange{Kind: bv.kind, ID: bv.id, Change: "changed", Before: bv.raw, After: av.raw})
		}
	}
	for key, av := range a {
		if _, ok := b[key]; !ok {
			out = append(out, SubjectChange{Kind: av.kind, ID: av.id, Change: "added", After: av.raw})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Diffは、二つのCanonical Worldのsemantic層の差をsubject単位で返す。
// 同じ意味を別fileへ移しただけの変更は、Worldが同一になるため差を生まない。
// 論理的な改名は、旧subjectのremovedと新subjectのaddedとして現れ、暗黙にno-opとは扱わない。
func Diff(before, after ir.World) ([]SubjectChange, error) {
	b, err := flattenSemantics(before.Semantics)
	if err != nil {
		return nil, fmt.Errorf("変更前のsemantic層: %w", err)
	}
	a, err := flattenSemantics(after.Semantics)
	if err != nil {
		return nil, fmt.Errorf("変更後のsemantic層: %w", err)
	}
	return diffSubjects(b, a), nil
}

// DiffApplicationsは、二つのCanonical WorldのApplication層の差をsubject単位で返す。
// subject kindは application（本体）と application-pattern（Pattern）の二つである。
func DiffApplications(before, after ir.World) ([]SubjectChange, error) {
	b, err := flattenApplications(before.Applications)
	if err != nil {
		return nil, fmt.Errorf("変更前のApplication層: %w", err)
	}
	a, err := flattenApplications(after.Applications)
	if err != nil {
		return nil, fmt.Errorf("変更後のApplication層: %w", err)
	}
	return diffSubjects(b, a), nil
}

// DiffEnvironmentsは、Environment層の差をsubject単位で返す。
// Environment bindingはApplicationが持つ意味ではなくEnvironmentが下した決定なので、semantic / application差分とは分けて扱う。
func DiffEnvironments(before, after ir.World) ([]SubjectChange, error) {
	b, err := flattenEnvironments(before.Environments)
	if err != nil {
		return nil, fmt.Errorf("変更前のEnvironment層: %w", err)
	}
	a, err := flattenEnvironments(after.Environments)
	if err != nil {
		return nil, fmt.Errorf("変更後のEnvironment層: %w", err)
	}
	return diffSubjects(b, a), nil
}

// EnvironmentBindingSubjectIDは、Environment binding一件のsubject IDを返す。
// Environmentを跨いで同じApplication / aliasが別instanceへ向くため、Environmentを含めて識別する。
func EnvironmentBindingSubjectID(env core.CanonicalID, application core.CanonicalID, alias core.LocalName) core.CanonicalID {
	return core.CanonicalID(string(env) + "/" + string(application) + "." + string(alias))
}

func flattenEnvironments(envs []environment.Environment) (subjectSet, error) {
	out := subjectSet{}
	for _, env := range envs {
		for _, binding := range env.Bindings {
			id := EnvironmentBindingSubjectID(env.ID, binding.Application, binding.Alias)
			if err := out.add("environment-binding", id, binding); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// CollectionDiffは、World直下のcollectionのうち内容が変わったものと変わらないものを返す。
// 期待したcollection以外が変わっていないことは、persistence・binding・Target入力へ副作用が及ばない根拠になる。
func CollectionDiff(before, after ir.World) (changed, unchanged []string, err error) {
	b, err := collections(before)
	if err != nil {
		return nil, nil, err
	}
	a, err := collections(after)
	if err != nil {
		return nil, nil, err
	}
	names := map[string]bool{}
	for name := range b {
		names[name] = true
	}
	for name := range a {
		names[name] = true
	}
	for name := range names {
		if bytes.Equal(b[name], a[name]) {
			unchanged = append(unchanged, name)
		} else {
			changed = append(changed, name)
		}
	}
	sort.Strings(changed)
	sort.Strings(unchanged)
	return changed, unchanged, nil
}

func collections(w ir.World) (map[string][]byte, error) {
	raw, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	out := make(map[string][]byte, len(fields))
	for name, value := range fields {
		out[name] = value
	}
	return out, nil
}
