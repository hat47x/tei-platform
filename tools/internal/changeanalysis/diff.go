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

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// SubjectChangeは、Canonical Worldのsemantic subject単位の差分である。
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

func flattenSemantics(namespaces []semantic.Namespace) (map[string]subjectValue, error) {
	out := map[string]subjectValue{}
	add := func(kind string, id core.CanonicalID, value any) error {
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		key := kind + ":" + string(id)
		if _, exists := out[key]; exists {
			return fmt.Errorf("%s が複数回宣言されています", key)
		}
		out[key] = subjectValue{kind: kind, id: string(id), raw: raw}
		return nil
	}
	type identity struct {
		ID     core.CanonicalID `json:"id"`
		Symbol string           `json:"symbol,omitempty"`
	}
	for _, ns := range namespaces {
		if err := add("namespace", ns.ID, identity{ID: ns.ID}); err != nil {
			return nil, err
		}
		for _, t := range ns.Types {
			if err := add("type", t.ID, t); err != nil {
				return nil, err
			}
		}
		for _, d := range ns.Data {
			if err := add("data", d.ID, identity{ID: d.ID, Symbol: d.Symbol}); err != nil {
				return nil, err
			}
			for _, a := range d.Attributes {
				if err := add("attribute", a.ID, a); err != nil {
					return nil, err
				}
			}
			for _, r := range d.Relations {
				if err := add("relation", r.ID, r); err != nil {
					return nil, err
				}
			}
		}
		for _, c := range ns.Capabilities {
			if err := add("capability", c.ID, c); err != nil {
				return nil, err
			}
		}
		for _, v := range ns.Validations {
			if err := add("validation", v.ID, v); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
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
	return out, nil
}

// CollectionDiffは、World直下のcollectionのうち内容が変わったものと変わらないものを返す。
// semantic層以外が変わっていないことは、persistence・binding・Target入力へ副作用が及ばない根拠になる。
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
