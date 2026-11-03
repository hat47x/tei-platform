package designoutput

import (
	"fmt"
	"sort"
	"strings"

	accessmatrix "tei/tools/reference-access-matrix"
)

// DeclaredAccess is one declared Application x Data access. It is the meaning
// the model states, never an observation of what ran.
type DeclaredAccess struct {
	Application string   `json:"application"`
	Data        string   `json:"data"`
	Access      []string `json:"access"`
}

// ObservedEffect is one side effect a caller reports it observed. The caller
// must mark the whole observation as observed; the reference never infers an
// effect from a declaration.
type ObservedEffect struct {
	Application string `json:"application"`
	Data        string `json:"data"`
	Access      string `json:"access"`
	Source      string `json:"source,omitempty"`
}

// EffectsObservation separates the declared access from the observed effects.
// Differences are reported, not reconciled: a declared update without an
// observation stays unobserved, and an observed effect without a declaration is
// an undeclared effect.
type EffectsObservation struct {
	Observed        bool             `json:"observed"`
	Source          string           `json:"source,omitempty"`
	Declared        []DeclaredAccess `json:"declared"`
	ObservedEffects []ObservedEffect `json:"observedEffects,omitempty"`
	Differences     []string         `json:"differences"`
	Note            string           `json:"note"`
}

// effectAccesses are the declared accesses that produce a side effect. Read is
// not an effect, and delete is not representable in the current matrix.
var effectAccesses = map[string]bool{"create": true, "update": true}

// buildEffects compares the declared matrix with an optional observation.
// Supplying observed effects without marking the observation as observed is an
// error, so a caller cannot imply an observation by accident.
func buildEffects(matrix accessmatrix.Matrix, observed bool, source string, effects []ObservedEffect) (EffectsObservation, error) {
	if !observed && len(effects) > 0 {
		return EffectsObservation{}, fmt.Errorf("observed effects were supplied without marking the observation as observed")
	}
	out := EffectsObservation{
		Observed:    observed,
		Source:      strings.TrimSpace(source),
		Declared:    []DeclaredAccess{},
		Differences: []string{},
	}
	if !observed {
		out.Note = "実際の副作用は観測していません。宣言された利用だけを示します（#1132）。"
	} else {
		out.Note = "宣言された利用と、呼出元が報告した副作用を別々に示します。照合は記録であり、受理ではありません。"
	}

	declared := map[string][]string{}
	for _, cell := range matrix.Cells {
		access := append([]string(nil), cell.Access...)
		sort.Strings(access)
		out.Declared = append(out.Declared, DeclaredAccess{Application: cell.Application, Data: cell.Data, Access: access})
		declared[cell.Application+"\x00"+cell.Data] = access
	}
	sort.Slice(out.Declared, func(i, j int) bool {
		if out.Declared[i].Application == out.Declared[j].Application {
			return out.Declared[i].Data < out.Declared[j].Data
		}
		return out.Declared[i].Application < out.Declared[j].Application
	})

	observedSet := map[string]bool{}
	for i := range effects {
		effect := &effects[i]
		access := strings.ToLower(strings.TrimSpace(effect.Access))
		if strings.TrimSpace(effect.Source) == "" {
			effect.Source = out.Source
		}
		effect.Access = access
		key := effect.Application + "\x00" + effect.Data
		observedSet[key+"\x00"+access] = true
		if !containsFold(declared[key], access) {
			out.Differences = append(out.Differences,
				fmt.Sprintf("undeclared-effect:%s:%s:%s", effect.Application, effect.Data, access))
		}
	}
	sort.Slice(effects, func(i, j int) bool {
		if effects[i].Application == effects[j].Application {
			if effects[i].Data == effects[j].Data {
				return effects[i].Access < effects[j].Access
			}
			return effects[i].Data < effects[j].Data
		}
		return effects[i].Application < effects[j].Application
	})
	if observed {
		out.ObservedEffects = effects
	}
	if observed {
		for key, access := range declared {
			parts := strings.SplitN(key, "\x00", 2)
			for _, declaredAccess := range access {
				if !effectAccesses[declaredAccess] {
					continue
				}
				if !observedSet[key+"\x00"+declaredAccess] {
					out.Differences = append(out.Differences,
						fmt.Sprintf("unobserved-declared-effect:%s:%s:%s", parts[0], parts[1], declaredAccess))
				}
			}
		}
	}
	sort.Strings(out.Differences)
	return out, nil
}

func containsFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}

// renderEffectsReport writes the declared access, the observation (or its
// absence), and the differences as separate sections.
func renderEffectsReport(effects EffectsObservation, structureRevision string) string {
	var b strings.Builder
	b.WriteString("# 宣言された利用と観測された副作用\n\n")
	b.WriteString("> 宣言は受理済みの意味、観測は実行時の記録です。この文書は両者を混ぜず、差を記録するだけです。\n\n")
	fmt.Fprintf(&b, "- 導出元の構造の版: `%s`\n", structureRevision)
	fmt.Fprintf(&b, "- %s\n\n", effects.Note)

	b.WriteString("## 宣言された利用\n\n")
	if len(effects.Declared) == 0 {
		b.WriteString("宣言された利用はありません。\n\n")
	} else {
		b.WriteString("| Application | Data | Access（宣言） |\n| --- | --- | --- |\n")
		for _, cell := range effects.Declared {
			fmt.Fprintf(&b, "| `%s` | `%s` | %s |\n", cell.Application, cell.Data, strings.Join(cell.Access, ", "))
		}
		b.WriteString("\n")
	}

	b.WriteString("## 観測された副作用\n\n")
	if !effects.Observed {
		b.WriteString("未観測です。この節の空欄を「副作用が無かった」と読みません（#1132）。\n\n")
	} else if len(effects.ObservedEffects) == 0 {
		fmt.Fprintf(&b, "観測は行われましたが、報告された副作用はありません。観測元: `%s`\n\n", effects.Source)
	} else {
		b.WriteString("| Application | Data | Access（観測） | 観測元 |\n| --- | --- | --- | --- |\n")
		for _, effect := range effects.ObservedEffects {
			fmt.Fprintf(&b, "| `%s` | `%s` | %s | `%s` |\n", effect.Application, effect.Data, effect.Access, effect.Source)
		}
		b.WriteString("\n")
	}

	b.WriteString("## 差\n\n")
	if len(effects.Differences) == 0 {
		if effects.Observed {
			b.WriteString("宣言と観測の間に差はありません。\n")
		} else {
			b.WriteString("未観測のため、差は判定していません。\n")
		}
		return b.String()
	}
	for _, difference := range effects.Differences {
		fmt.Fprintf(&b, "- %s\n", difference)
	}
	return b.String()
}
