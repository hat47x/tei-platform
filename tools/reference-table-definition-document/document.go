// Package tabledoc renders a table definition document from an accepted
// World. The document is a derived artifact: it records the structure
// revision it came from, and editing it never changes the model.
package tabledoc

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/semantic"
)

const Schema = "tei.reference.table-definition-document/v0"

type Document struct {
	Schema            string   `json:"schema"`
	StructureRevision string   `json:"structureRevision"`
	Digest            string   `json:"digest"`
	Unresolved        []string `json:"unresolved"`
	Markdown          string   `json:"-"`
}

// Render is deterministic: the same World gives the same bytes regardless
// of the order of namespaces, Data, or mappings in the input.
func Render(world ir.World) (Document, error) {
	raw, err := json.Marshal(struct {
		S, P interface{}
	}{world.Semantics, world.Persistence})
	if err != nil {
		return Document{}, err
	}
	rev := fmt.Sprintf("sha256:%x", sha256.Sum256(raw))

	types := map[string]string{}
	var data []semantic.Data
	for _, ns := range world.Semantics {
		for _, t := range ns.Types {
			types[string(t.ID)] = t.Symbol
		}
		data = append(data, ns.Data...)
	}
	sort.Slice(data, func(i, j int) bool { return data[i].ID < data[j].ID })
	known := map[string]bool{}
	for _, d := range data {
		known[string(d.ID)] = true
	}
	mappings := map[string][]persistence.Mapping{}
	var unresolved []string
	for _, m := range world.Persistence {
		if !known[string(m.SemanticData)] {
			unresolved = append(unresolved, fmt.Sprintf("物理対応 %s の対象Data %s が存在しない", m.ID, m.SemanticData))
			continue
		}
		mappings[string(m.SemanticData)] = append(mappings[string(m.SemanticData)], m)
	}
	for k := range mappings {
		ms := mappings[k]
		sort.Slice(ms, func(i, j int) bool { return ms[i].ID < ms[j].ID })
	}

	var b strings.Builder
	w := func(format string, a ...interface{}) { fmt.Fprintf(&b, format, a...) }
	w("# 表定義書\n\n")
	w("> 受理済みの意味から導出した文書です。この文書を編集しても、意味の正本には反映されません。\n\n")
	w("- 導出元の構造の版: `%s`\n", rev)
	w("- 対象のData: %d件\n\n", len(data))

	for _, d := range data {
		id := string(d.ID)
		w("## %s（`%s`）\n\n", cell(d.Symbol), id)

		cols := map[string]persistence.ColumnMapping{}
		for _, m := range mappings[id] {
			for _, c := range m.Columns {
				cols[string(c.Attribute)] = c
			}
		}
		w("### 属性\n\n| 識別子 | 名称 | 型 | 物理列 | 格納型 | NULL |\n| --- | --- | --- | --- | --- | --- |\n")
		for _, a := range d.Attributes {
			t := string(a.TypeRef)
			typeCell := "`" + t + "`"
			if sym, ok := types[t]; ok {
				typeCell = cell(sym) + "（`" + t + "`）"
			} else if t != "" {
				unresolved = append(unresolved, fmt.Sprintf("属性 %s の型 %s が定義されていない", a.ID, t))
				typeCell += "（未解決）"
			}
			col, storage, null := "—", "—", "—"
			if c, ok := cols[string(a.ID)]; ok {
				col, storage, null = cell(c.Column), string(c.Storage), string(c.Nullability)
			}
			w("| `%s` | %s | %s | %s | %s | %s |\n", a.ID, cell(a.Symbol), typeCell, col, storage, null)
		}
		w("\n")

		if len(d.CandidateKeys) > 0 {
			w("### 候補キー（業務上の識別）\n\n")
			for _, k := range d.CandidateKeys {
				w("- %s（`%s`）: %s\n", cell(k.Symbol), k.ID, ids(k.Attributes))
			}
			w("\n")
		}
		if len(d.FunctionalDependencies) > 0 {
			w("### 関数従属性\n\n")
			for _, f := range d.FunctionalDependencies {
				w("- %s（`%s`）: %s → %s\n", cell(f.Symbol), f.ID, ids(f.Determinants), ids(f.Dependents))
			}
			w("\n")
		}
		if len(d.Relations) > 0 {
			w("### 関連\n\n| 識別子 | 名称 | 関連先 | 多重度 | キーの対応 |\n| --- | --- | --- | --- | --- |\n")
			for _, r := range d.Relations {
				var keys []string
				for _, k := range r.Keys {
					keys = append(keys, fmt.Sprintf("`%s` = `%s`", k.SourceAttribute, k.TargetAttribute))
				}
				keyCell := "（未定義）"
				if len(keys) > 0 {
					keyCell = strings.Join(keys, "、")
				}
				target := "`" + string(r.Target) + "`"
				if !known[string(r.Target)] {
					unresolved = append(unresolved, fmt.Sprintf("関連 %s の関連先 %s が存在しない", r.ID, r.Target))
					target += "（未解決）"
				}
				w("| `%s` | %s | %s | %s | %s |\n", r.ID, cell(r.Symbol), target, r.Cardinality, keyCell)
			}
			w("\n")
		}

		w("### 物理表\n\n")
		if len(mappings[id]) == 0 {
			w("物理表への対応は宣言されていません。\n\n")
			continue
		}
		for _, m := range mappings[id] {
			w("- `%s`（データベース `%s`、対応 `%s`）\n", cell(m.Table), m.Database, m.ID)
			if len(m.PrimaryKey) > 0 {
				w("  - 主キー: %s\n", ids(m.PrimaryKey))
			}
			for _, u := range m.UniqueConstraints {
				w("  - 一意制約 %s: %s\n", cell(u.Name), ids(u.Attributes))
			}
			for _, f := range m.ForeignKeys {
				w("  - 外部キー %s: %s → `%s`\n", cell(f.Name), ids(f.Attributes), f.ReferencedMapping)
			}
		}
		w("\n")
	}

	sort.Strings(unresolved)
	if len(unresolved) > 0 {
		w("## 未解決の事項\n\n")
		for _, u := range unresolved {
			w("- %s\n", u)
		}
	}
	md := b.String()
	if unresolved == nil {
		unresolved = []string{}
	}
	return Document{
		Schema: Schema, StructureRevision: rev,
		Digest:     fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(md))),
		Unresolved: unresolved, Markdown: md,
	}, nil
}

func ids[T ~string](in []T) string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = "`" + string(v) + "`"
	}
	return strings.Join(out, "、")
}

// cell keeps authored text from breaking a Markdown table row.
func cell(s string) string {
	if s == "" {
		return "—"
	}
	s = strings.ReplaceAll(s, "|", `\|`)
	return strings.ReplaceAll(s, "\n", " ")
}
