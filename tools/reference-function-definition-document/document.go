// Package funcdoc renders a function definition document from an accepted
// World: for each Application, the Patterns it declares and the Capabilities
// and Validations they rely on. The document is derived; editing it never
// changes the model, and it states only what the model declares.
package funcdoc

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/semantic"
)

const Schema = "tei.reference.function-definition-document/v0"

type Document struct {
	Schema            string   `json:"schema"`
	StructureRevision string   `json:"structureRevision"`
	Digest            string   `json:"digest"`
	Unresolved        []string `json:"unresolved"`
	Markdown          string   `json:"-"`
}

func Render(world ir.World) (Document, error) {
	raw, err := json.Marshal(struct{ S, A interface{} }{world.Semantics, world.Applications})
	if err != nil {
		return Document{}, err
	}
	rev := fmt.Sprintf("sha256:%x", sha256.Sum256(raw))

	caps := map[string]semantic.Capability{}
	data := map[string]semantic.Data{}
	validations := map[string]bool{}
	for _, ns := range world.Semantics {
		for _, c := range ns.Capabilities {
			caps[string(c.ID)] = c
		}
		for _, d := range ns.Data {
			data[string(d.ID)] = d
		}
		for _, v := range ns.Validations {
			validations[string(v.ID)] = true
		}
	}
	var unresolved []string
	note := func(format string, a ...interface{}) { unresolved = append(unresolved, fmt.Sprintf(format, a...)) }

	apps := append([]application.Application(nil), world.Applications...)
	sort.Slice(apps, func(i, j int) bool { return apps[i].ID < apps[j].ID })

	var b strings.Builder
	w := func(format string, a ...interface{}) { fmt.Fprintf(&b, format, a...) }
	w("# 機能定義書\n\n")
	w("> 受理済みの意味から導出した文書です。この文書を編集しても、意味の正本には反映されません。\n\n")
	w("- 導出元の構造の版: `%s`\n", rev)
	w("- Application: %d件\n", len(apps))
	w("- 記載は宣言された内容だけです。画面の見た目、実行時の副作用の実績、権限の付与は含みません。\n\n")

	for _, app := range apps {
		a := string(app.ID)
		w("## `%s`\n\n", a)
		if len(app.SemanticUses) > 0 {
			w("- 利用する意味: %s\n", codes(app.SemanticUses))
		}
		if len(app.Validations) > 0 {
			var parts []string
			for _, v := range app.Validations {
				p := "`" + string(v) + "`"
				if !validations[string(v)] {
					p += "（未解決）"
					note("Application %s が評価する検証 %s が定義されていない", a, v)
				}
				parts = append(parts, p)
			}
			w("- 評価する検証（宣言順）: %s\n", strings.Join(parts, "、"))
		}
		for _, d := range app.Databases {
			w("- データベース要求: 別名 `%s` → `%s`（%s）\n", d.Alias, d.TargetDatabase, d.Access)
		}
		w("\n")
		if len(app.Patterns) == 0 {
			w("Patternは宣言されていません。\n\n")
			continue
		}
		w("### Pattern\n\n| 識別子 | 種類 | 対象Data | 関連 | 問い合わせ | 機能（Capability） |\n| --- | --- | --- | --- | --- | --- |\n")
		for _, p := range app.Patterns {
			dataCell := orDash(string(p.Data))
			if p.Data != "" {
				if _, ok := data[string(p.Data)]; !ok {
					dataCell += "（未解決）"
					note("Pattern %s の対象Data %s が存在しない", p.ID, p.Data)
				}
			}
			capCell := orDash(string(p.Capability))
			if p.Capability != "" {
				if _, ok := caps[string(p.Capability)]; !ok {
					capCell += "（未解決）"
					note("Pattern %s のCapability %s が存在しない", p.ID, p.Capability)
				}
			}
			w("| `%s` | %s | %s | %s | %s | %s |\n", p.ID, p.Kind, dataCell, orDash(string(p.Relation)), orDash(string(p.Query)), capCell)
		}
		w("\n")

		used := map[string]bool{}
		for _, p := range app.Patterns {
			if _, ok := caps[string(p.Capability)]; ok {
				used[string(p.Capability)] = true
			}
		}
		var ids []string
		for id := range used {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			c := caps[id]
			w("### Capability `%s`\n\n", id)
			if len(c.Inputs) > 0 {
				var in []string
				for _, p := range c.Inputs {
					in = append(in, fmt.Sprintf("`%s`: `%s`", p.Name, p.Type))
				}
				w("- 入力: %s\n", strings.Join(in, "、"))
			} else {
				w("- 入力: なし\n")
			}
			w("- 出力: %s\n", orDash(string(c.Output)))
			for _, o := range c.Outcomes {
				w("- 業務上の結末 `%s`: `%s`\n", o.Name, o.Payload)
			}
			if c.Effect != nil {
				w("- 宣言された効果: %s（`%s`）。実際の副作用の観測ではありません。\n", c.Effect.Kind, c.Effect.Data)
			} else {
				w("- 宣言された効果: なし\n")
			}
			w("\n")
		}
	}

	sort.Strings(unresolved)
	unresolved = dedupe(unresolved)
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

func codes[T ~string](in []T) string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = "`" + string(v) + "`"
	}
	return strings.Join(out, "、")
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return "`" + strings.ReplaceAll(s, "|", `\|`) + "`"
}

func dedupe(in []string) []string {
	var out []string
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}
