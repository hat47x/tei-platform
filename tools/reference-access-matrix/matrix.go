// Package accessmatrix derives an Application x Data matrix of declared data
// access from an accepted World. Every cell is a declaration read from a
// Pattern or Capability; none of it is an observed or actual effect.
package accessmatrix

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

const Schema = "tei.reference.declared-access-matrix/v0"

// Access letters. Delete is not expressible in the current model, so the
// matrix never shows it and says so instead of leaving a blank that reads as
// "does not delete".
const (
	Create = "C"
	Read   = "R"
	Update = "U"
)

type Cell struct {
	Application string   `json:"application"`
	Data        string   `json:"data"`
	Access      []string `json:"access"`
	// Via names the Pattern that declares each access, so a reader can see
	// where the entry comes from.
	Via []string `json:"via"`
}

type Gap struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
}

type Matrix struct {
	Schema            string   `json:"schema"`
	StructureRevision string   `json:"structureRevision"`
	Applications      []string `json:"applications"`
	Data              []string `json:"data"`
	Cells             []Cell   `json:"cells"`
	Gaps              []Gap    `json:"gaps"`
	Notes             []string `json:"notes"`
	Markdown          string   `json:"-"`
}

func Derive(world ir.World) (Matrix, error) {
	raw, err := json.Marshal(struct{ S, A interface{} }{world.Semantics, world.Applications})
	if err != nil {
		return Matrix{}, err
	}
	m := Matrix{
		Schema: Schema, StructureRevision: fmt.Sprintf("sha256:%x", sha256.Sum256(raw)),
		Applications: []string{}, Data: []string{}, Cells: []Cell{}, Gaps: []Gap{},
		Notes: []string{
			"各欄は、PatternまたはCapabilityの宣言から読み取った内容です。実際の副作用の観測ではありません。",
			"削除は現在のモデルで表現できないため、この表には現れません。空欄を「削除しない」と読まないでください。",
		},
	}
	caps := map[string]semantic.Capability{}
	known := map[string]bool{}
	for _, ns := range world.Semantics {
		for _, c := range ns.Capabilities {
			caps[string(c.ID)] = c
		}
		for _, d := range ns.Data {
			known[string(d.ID)] = true
		}
	}

	type key struct{ app, data string }
	access := map[key]map[string]bool{}
	via := map[key]map[string]bool{}
	add := func(app, data, letter, pattern string) {
		k := key{app, data}
		if access[k] == nil {
			access[k], via[k] = map[string]bool{}, map[string]bool{}
		}
		access[k][letter] = true
		via[k][pattern] = true
	}
	dataSeen := map[string]bool{}
	apps := append([]application.Application(nil), world.Applications...)
	sort.Slice(apps, func(i, j int) bool { return apps[i].ID < apps[j].ID })
	for _, app := range apps {
		a := string(app.ID)
		m.Applications = append(m.Applications, a)
		for _, p := range app.Patterns {
			pid := string(p.ID)
			switch p.Kind {
			case application.PatternCommand:
				c, ok := caps[string(p.Capability)]
				switch {
				case !ok:
					m.Gaps = append(m.Gaps, Gap{"capability-not-found", pid})
				case c.Effect == nil:
					m.Gaps = append(m.Gaps, Gap{"capability-declares-no-effect", pid})
				default:
					d := string(c.Effect.Data)
					letter := Update
					if c.Effect.Kind == semantic.DataEffectCreate {
						letter = Create
					}
					if !known[d] {
						m.Gaps = append(m.Gaps, Gap{"effect-data-not-found", pid})
						continue
					}
					add(a, d, letter, pid)
					dataSeen[d] = true
				}
			default:
				d := string(p.Data)
				if d == "" {
					m.Gaps = append(m.Gaps, Gap{"pattern-declares-no-data", pid})
					continue
				}
				if !known[d] {
					m.Gaps = append(m.Gaps, Gap{"pattern-data-not-found", pid})
					continue
				}
				add(a, d, Read, pid)
				dataSeen[d] = true
			}
		}
	}
	for d := range dataSeen {
		m.Data = append(m.Data, d)
	}
	sort.Strings(m.Data)
	for k, set := range access {
		c := Cell{Application: k.app, Data: k.data, Access: sorted(set), Via: sorted(via[k])}
		m.Cells = append(m.Cells, c)
	}
	sort.Slice(m.Cells, func(i, j int) bool {
		if m.Cells[i].Application != m.Cells[j].Application {
			return m.Cells[i].Application < m.Cells[j].Application
		}
		return m.Cells[i].Data < m.Cells[j].Data
	})
	sort.Slice(m.Gaps, func(i, j int) bool {
		if m.Gaps[i].Code != m.Gaps[j].Code {
			return m.Gaps[i].Code < m.Gaps[j].Code
		}
		return m.Gaps[i].Subject < m.Gaps[j].Subject
	})
	m.Markdown = render(m)
	return m, nil
}

func sorted(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func render(m Matrix) string {
	var b strings.Builder
	b.WriteString("# 宣言された利用の対応表（Application × Data）\n\n")
	b.WriteString("> 受理済みの意味から導出した文書です。編集しても意味の正本には反映されません。\n\n")
	fmt.Fprintf(&b, "- 導出元の構造の版: `%s`\n", m.StructureRevision)
	for _, n := range m.Notes {
		fmt.Fprintf(&b, "- %s\n", n)
	}
	b.WriteString("\n")
	if len(m.Data) == 0 {
		b.WriteString("宣言された利用はありません。\n")
	} else {
		cell := map[string]string{}
		for _, c := range m.Cells {
			cell[c.Application+"\x00"+c.Data] = strings.Join(c.Access, "")
		}
		b.WriteString("| Application |")
		for _, d := range m.Data {
			fmt.Fprintf(&b, " `%s` |", d)
		}
		b.WriteString("\n| --- |")
		for range m.Data {
			b.WriteString(" --- |")
		}
		b.WriteString("\n")
		for _, a := range m.Applications {
			fmt.Fprintf(&b, "| `%s` |", a)
			for _, d := range m.Data {
				v := cell[a+"\x00"+d]
				if v == "" {
					v = "—"
				}
				fmt.Fprintf(&b, " %s |", v)
			}
			b.WriteString("\n")
		}
	}
	if len(m.Gaps) > 0 {
		b.WriteString("\n## 読み取れなかった宣言\n\n")
		for _, g := range m.Gaps {
			fmt.Fprintf(&b, "- %s: `%s`\n", g.Code, g.Subject)
		}
	}
	return b.String()
}
