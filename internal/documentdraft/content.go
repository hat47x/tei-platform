package documentdraft

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/operations"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

// The generators below are deliberately literal. They restate what the accepted
// model declares — identity, counts, declared types, declared targets — and
// never invent business meaning, never complete a missing declaration and never
// assert that a declaration is correct. A reader that needs the meaning itself
// must go back to the Source and to Owner acceptance.

// dataExplanation generates the prose occurrence for one data element.
func dataExplanation(subject sourceindex.Subject, data semantic.Data) string {
	lines := []string{
		fmt.Sprintf("%s は %s として宣言されたデータ要素です。", subject.ID, subject.Kind),
	}
	if data.Symbol != "" {
		lines = append(lines, fmt.Sprintf("宣言された記号は %s です。", data.Symbol))
	}
	if data.Description != "" {
		lines = append(lines, fmt.Sprintf("Source に記述されている説明は「%s」です。", data.Description))
	} else {
		lines = append(lines, "Source に説明は宣言されていません。")
	}
	lines = append(lines, fmt.Sprintf("属性は %d 件、関係は %d 件宣言されています。",
		len(data.Attributes), len(data.Relations)))
	if len(data.CandidateKeys) > 0 {
		lines = append(lines, fmt.Sprintf("候補キーは %d 件宣言されています。", len(data.CandidateKeys)))
	}
	if len(data.FunctionalDependencies) > 0 {
		lines = append(lines, fmt.Sprintf("関数従属は %d 件宣言されています。", len(data.FunctionalDependencies)))
	}
	lines = append(lines,
		"この文は受理済みモデルからの機械的な投影であり、業務意味の説明・受理・是正ではありません。")
	return strings.Join(lines, "\n")
}

// typeExplanation generates the prose occurrence for one semantic type.
func typeExplanation(subject sourceindex.Subject, typ semantic.Type) string {
	lines := []string{
		fmt.Sprintf("%s は %s として宣言された型です。", subject.ID, subject.Kind),
		fmt.Sprintf("種別は %s です。", typ.Kind),
	}
	if typ.Underlying != "" {
		lines = append(lines, fmt.Sprintf("基底は %s です。", typ.Underlying))
	}
	if len(typ.Values) > 0 {
		lines = append(lines, fmt.Sprintf("値は %d 件宣言されています。", len(typ.Values)))
	} else {
		lines = append(lines, "値の列挙は宣言されていません。")
	}
	lines = append(lines,
		"この文は受理済みモデルからの機械的な投影であり、業務意味の説明・受理・是正ではありません。")
	return strings.Join(lines, "\n")
}

// capabilityExplanation generates the prose occurrence for one capability.
func capabilityExplanation(subject sourceindex.Subject, capability semantic.Capability) string {
	lines := []string{
		fmt.Sprintf("%s は %s として宣言された能力です。", subject.ID, subject.Kind),
		fmt.Sprintf("入力は %d 件、出力は %d 件、結末は %d 件宣言されています。",
			len(capability.Inputs), outcomeCount(capability), len(capability.Outcomes)),
	}
	if capability.Effect != nil {
		lines = append(lines, fmt.Sprintf("宣言された効果は %s(%s) です。",
			capability.Effect.Kind, capability.Effect.Data))
	} else {
		lines = append(lines, "データへの効果は宣言されていません。")
	}
	lines = append(lines,
		"この文は受理済みモデルからの機械的な投影であり、実行権限や業務意味の説明・受理ではありません。")
	return strings.Join(lines, "\n")
}

func outcomeCount(capability semantic.Capability) int {
	if capability.Output == "" {
		return 0
	}
	return 1
}

// attributeTable generates the attribute table of one data element. Only
// declared values are written; an undeclared flag stays empty rather than
// becoming "false".
func attributeTable(data semantic.Data) *TableContent {
	table := &TableContent{
		Caption: fmt.Sprintf("%s の属性", data.ID),
		Header:  []string{"属性", "型", "必須", "不変", "役割"},
		Rows:    [][]string{},
	}
	for _, attribute := range data.Attributes {
		table.Rows = append(table.Rows, []string{
			string(attribute.ID),
			string(attribute.TypeRef),
			declared(attribute.Required),
			declared(attribute.Immutable),
			attribute.Role,
		})
	}
	for _, relation := range data.Relations {
		table.Rows = append(table.Rows, []string{
			string(relation.ID),
			"relation -> " + string(relation.Target),
			"", "", string(relation.Cardinality),
		})
	}
	return table
}

func declared(value bool) string {
	if value {
		return "yes"
	}
	return ""
}

// typeTable generates the shape table of one semantic type.
func typeTable(typ semantic.Type) *TableContent {
	values := append([]string(nil), typ.Values...)
	sort.Strings(values)
	return &TableContent{
		Caption: fmt.Sprintf("%s の型定義", typ.ID),
		Header:  []string{"項目", "値"},
		Rows: [][]string{
			{"kind", string(typ.Kind)},
			{"underlying", string(typ.Underlying)},
			{"values", strings.Join(values, ", ")},
		},
	}
}

// capabilityTable generates the shape table of one capability.
func capabilityTable(capability semantic.Capability) *TableContent {
	rows := [][]string{}
	for _, input := range capability.Inputs {
		rows = append(rows, []string{"input", fmt.Sprintf("%s: %s", input.Name, input.Type)})
	}
	if capability.Output != "" {
		rows = append(rows, []string{"output", string(capability.Output)})
	}
	for _, outcome := range capability.Outcomes {
		rows = append(rows, []string{"outcome", fmt.Sprintf("%s: %s", outcome.Name, outcome.Payload)})
	}
	if capability.Effect != nil {
		rows = append(rows, []string{"effect", fmt.Sprintf("%s: %s", capability.Effect.Kind, capability.Effect.Data)})
	}
	return &TableContent{
		Caption: fmt.Sprintf("%s の宣言", capability.ID),
		Header:  []string{"項目", "値"},
		Rows:    rows,
	}
}

// citationOf generates one citation occurrence for one Source artifact. The
// resolution is always "recorded-not-resolved": a citation records where the
// declaration was read from, and does not acquire, re-read or verify it.
func citationOf(inspection operations.SubjectInspection, artifact sourceindex.Artifact) *CitationContent {
	role := ""
	for _, reference := range inspection.Sources.References {
		if reference.ArtifactPath == artifact.Path {
			role = string(reference.Role)
			break
		}
	}
	return &CitationContent{
		ArtifactPath: artifact.Path,
		Fingerprint:  artifact.Fingerprint,
		Role:         role,
		Ranges:       rangesFor(inspection, inspection.Subject, artifact.Path),
		Resolution:   "recorded-not-resolved",
	}
}
