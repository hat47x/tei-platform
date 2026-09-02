package validation

import (
	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

// ValidateReferenceContractQueryInvocationsは、reference Patternに明示された
// Database Contract Query呼び出しの基本的な整合性を検査する。
//
// Relation keyやQuery signatureとの横断Validationは、それぞれのCanonical modelが
// 共通のWorldへ合流した後に扱う。ここでは、Application自身が所有する情報だけで
// 推測なしに確認できる境界へ検査を限定する。
func ValidateReferenceContractQueryInvocations(w ir.World) []Diagnostic {
	var out []Diagnostic

	for _, app := range w.Applications {
		for _, pattern := range app.Patterns {
			invocation := pattern.ContractQuery
			if invocation == nil {
				continue
			}

			patternSubject := subject.Subject{Kind: "application-pattern", ID: string(pattern.ID)}

			if pattern.Kind != application.PatternReference {
				out = append(out, errFor(
					patternSubject,
					"CONTRACT_QUERY_INVOCATION_REQUIRES_REFERENCE_PATTERN",
					"Pattern %sは%sですが、Database Contract Query呼び出しはreference Patternでのみ利用できます",
					pattern.ID,
					pattern.Kind,
				))
			}

			if invocation.Contract == "" {
				out = append(out, errFor(
					patternSubject,
					"REFERENCE_CONTRACT_QUERY_CONTRACT_REQUIRED",
					"reference Pattern %sのContract Query呼び出しにはContractが必要です",
					pattern.ID,
				))
			} else if _, err := core.ParseCanonicalID(string(invocation.Contract)); err != nil {
				out = append(out, errFor(
					patternSubject,
					"INVALID_REFERENCE_CONTRACT_QUERY_CONTRACT",
					"reference Pattern %sのContract %qは有効なCanonical IDではありません",
					pattern.ID,
					invocation.Contract,
				))
			}

			if invocation.Query == "" {
				out = append(out, errFor(
					patternSubject,
					"REFERENCE_CONTRACT_QUERY_NAME_REQUIRED",
					"reference Pattern %sのContract Query呼び出しにはQuery名が必要です",
					pattern.ID,
				))
			} else if _, err := core.ParseLocalName(string(invocation.Query)); err != nil {
				out = append(out, errFor(
					patternSubject,
					"INVALID_REFERENCE_CONTRACT_QUERY_NAME",
					"reference Pattern %sのContract Query名 %qは有効なLocalNameではありません",
					pattern.ID,
					invocation.Query,
				))
			}

			if pattern.Datasource == "" {
				out = append(out, errFor(
					patternSubject,
					"REFERENCE_CONTRACT_QUERY_DATASOURCE_REQUIRED",
					"reference Pattern %sでContract Queryを呼び出すにはDatasourceが必要です",
					pattern.ID,
				))
			} else if datasource, ok := app.DatabaseByAlias(pattern.Datasource); ok {
				if invocation.Contract != "" {
					required := false
					for _, requirement := range datasource.RequiredContracts {
						if requirement.ContractID == invocation.Contract {
							required = true
							break
						}
					}
					if !required {
						out = append(out, errFor(
							patternSubject,
							"REFERENCE_CONTRACT_QUERY_CONTRACT_NOT_REQUIRED",
							"reference Pattern %sはDatasource %sからContract %sを呼び出そうとしていますが、このContractはApplicationのrequiresにありません",
							pattern.ID,
							pattern.Datasource,
							invocation.Contract,
						))
					}
				}
			}

			seenTargetAttributes := map[core.CanonicalID]struct{}{}
			seenInputs := map[core.LocalName]struct{}{}
			for _, binding := range invocation.Inputs {
				if binding.TargetAttribute == "" {
					out = append(out, errFor(
						patternSubject,
						"REFERENCE_CONTRACT_QUERY_TARGET_ATTRIBUTE_REQUIRED",
						"reference Pattern %sのContract Query input bindingにはtarget attributeが必要です",
						pattern.ID,
					))
				} else {
					if _, duplicate := seenTargetAttributes[binding.TargetAttribute]; duplicate {
						out = append(out, errFor(
							patternSubject,
							"DUPLICATE_REFERENCE_CONTRACT_QUERY_TARGET_ATTRIBUTE",
							"reference Pattern %sではtarget attribute %sが複数のContract Query input bindingに使われています",
							pattern.ID,
							binding.TargetAttribute,
						))
					} else {
						seenTargetAttributes[binding.TargetAttribute] = struct{}{}
					}
				}

				if binding.Input == "" {
					out = append(out, errFor(
						patternSubject,
						"REFERENCE_CONTRACT_QUERY_INPUT_REQUIRED",
						"reference Pattern %sのContract Query input bindingにはinput名が必要です",
						pattern.ID,
					))
					continue
				}
				if _, err := core.ParseLocalName(string(binding.Input)); err != nil {
					out = append(out, errFor(
						patternSubject,
						"INVALID_REFERENCE_CONTRACT_QUERY_INPUT",
						"reference Pattern %sのContract Query input名 %qは有効なLocalNameではありません",
						pattern.ID,
						binding.Input,
					))
				}
				if _, duplicate := seenInputs[binding.Input]; duplicate {
					out = append(out, errFor(
						patternSubject,
						"DUPLICATE_REFERENCE_CONTRACT_QUERY_INPUT",
						"reference Pattern %sではContract Query input %sが複数のbindingで使われています",
						pattern.ID,
						binding.Input,
					))
				} else {
					seenInputs[binding.Input] = struct{}{}
				}
			}
		}
	}

	return out
}
