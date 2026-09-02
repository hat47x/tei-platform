package validation

import (
	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/subject"
)

// ValidateReferenceContractQueryInvocationsは、reference Patternに明示された
// Database Contract Query呼び出しの整合性を検査する。
//
// Relation keyやtyped Query signatureとの横断Validationは、それぞれのCanonical modelが
// 共通のWorldへ合流した後に扱う。ここでは、Applicationが明示したDatasource / Contract / Queryを
// 名前や型から補完せずに一意に解決できるところまでを確認する。
func ValidateReferenceContractQueryInvocations(w ir.World) []Diagnostic {
	var out []Diagnostic

	for _, app := range w.Applications {
		for _, pattern := range app.Patterns {
			invocation := pattern.ContractQuery
			if invocation == nil {
				continue
			}

			patternSubject := subject.Subject{Kind: "application-pattern", ID: string(pattern.ID)}
			isReference := pattern.Kind == application.PatternReference

			if !isReference {
				out = append(out, errFor(
					patternSubject,
					"CONTRACT_QUERY_INVOCATION_REQUIRES_REFERENCE_PATTERN",
					"Pattern %sは%sですが、Database Contract Query呼び出しはreference Patternでのみ利用できます",
					pattern.ID,
					pattern.Kind,
				))
			}

			contractValid := true
			if invocation.Contract == "" {
				contractValid = false
				out = append(out, errFor(
					patternSubject,
					"REFERENCE_CONTRACT_QUERY_CONTRACT_REQUIRED",
					"reference Pattern %sのContract Query呼び出しにはContractが必要です",
					pattern.ID,
				))
			} else if _, err := core.ParseCanonicalID(string(invocation.Contract)); err != nil {
				contractValid = false
				out = append(out, errFor(
					patternSubject,
					"INVALID_REFERENCE_CONTRACT_QUERY_CONTRACT",
					"reference Pattern %sのContract %qは有効なCanonical IDではありません",
					pattern.ID,
					invocation.Contract,
				))
			}

			queryValid := true
			if invocation.Query == "" {
				queryValid = false
				out = append(out, errFor(
					patternSubject,
					"REFERENCE_CONTRACT_QUERY_NAME_REQUIRED",
					"reference Pattern %sのContract Query呼び出しにはQuery名が必要です",
					pattern.ID,
				))
			} else if _, err := core.ParseLocalName(string(invocation.Query)); err != nil {
				queryValid = false
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
			} else if datasource, ok := app.DatabaseByAlias(pattern.Datasource); ok && contractValid {
				requirements := requiredContractRequirements(datasource, invocation.Contract)
				switch len(requirements) {
				case 0:
					out = append(out, errFor(
						patternSubject,
						"REFERENCE_CONTRACT_QUERY_CONTRACT_NOT_REQUIRED",
						"reference Pattern %sはDatasource %sからContract %sを呼び出そうとしていますが、このContractはApplicationのrequiresにありません",
						pattern.ID,
						pattern.Datasource,
						invocation.Contract,
					))
				case 1:
					if isReference && queryValid {
						out = append(out, validateReferenceContractQueryTarget(
							w,
							patternSubject,
							pattern,
							datasource,
							requirements[0],
							*invocation,
						)...)
					}
				default:
					out = append(out, errFor(
						patternSubject,
						"AMBIGUOUS_REFERENCE_CONTRACT_QUERY_REQUIREMENT",
						"reference Pattern %sが利用するContract %sのrequiresがDatasource %sに%d件あり、一意に解決できません",
						pattern.ID,
						invocation.Contract,
						pattern.Datasource,
						len(requirements),
					))
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

func requiredContractRequirements(
	datasource application.DatabaseRequirement,
	contractID core.CanonicalID,
) []database.ContractRequirement {
	var out []database.ContractRequirement
	for _, requirement := range datasource.RequiredContracts {
		if requirement.ContractID == contractID {
			out = append(out, requirement)
		}
	}
	return out
}

func validateReferenceContractQueryTarget(
	w ir.World,
	patternSubject subject.Subject,
	pattern application.Pattern,
	datasource application.DatabaseRequirement,
	requirement database.ContractRequirement,
	invocation application.ContractQueryInvocation,
) []Diagnostic {
	var out []Diagnostic

	var databases []database.Database
	for _, candidate := range w.Databases {
		if candidate.ID == datasource.TargetDatabase {
			databases = append(databases, candidate)
		}
	}
	if len(databases) == 0 {
		return []Diagnostic{errFor(
			patternSubject,
			"UNKNOWN_REFERENCE_CONTRACT_QUERY_DATABASE",
			"reference Pattern %sのDatasource %sが参照するDatabase %sが見つかりません",
			pattern.ID,
			pattern.Datasource,
			datasource.TargetDatabase,
		)}
	}
	if len(databases) > 1 {
		return []Diagnostic{errFor(
			patternSubject,
			"AMBIGUOUS_REFERENCE_CONTRACT_QUERY_DATABASE",
			"reference Pattern %sのDatasource %sが参照するDatabase %sが%d件あり、一意に解決できません",
			pattern.ID,
			pattern.Datasource,
			datasource.TargetDatabase,
			len(databases),
		)}
	}

	db := databases[0]
	var contracts []database.Contract
	for _, candidate := range db.Contracts {
		if candidate.ID == invocation.Contract {
			contracts = append(contracts, candidate)
		}
	}
	if len(contracts) == 0 {
		return []Diagnostic{errFor(
			patternSubject,
			"UNKNOWN_REFERENCE_CONTRACT_QUERY_CONTRACT",
			"reference Pattern %sが利用するContract %sはDatabase %sにありません",
			pattern.ID,
			invocation.Contract,
			db.ID,
		)}
	}
	if len(contracts) > 1 {
		return []Diagnostic{errFor(
			patternSubject,
			"AMBIGUOUS_REFERENCE_CONTRACT_QUERY_CONTRACT",
			"reference Pattern %sが利用するContract %sはDatabase %sに%d件あり、一意に解決できません",
			pattern.ID,
			invocation.Contract,
			db.ID,
			len(contracts),
		)}
	}

	contract := contracts[0]
	if contract.Version.Major != requirement.CompatibleMajor {
		return []Diagnostic{errFor(
			patternSubject,
			"REFERENCE_CONTRACT_QUERY_MAJOR_VERSION_MISMATCH",
			"reference Pattern %sはContract %sのmajor version %dとの互換性を要求していますが、Database %sではv%d.%dです",
			pattern.ID,
			invocation.Contract,
			requirement.CompatibleMajor,
			db.ID,
			contract.Version.Major,
			contract.Version.Minor,
		)}
	}

	queryCount := 0
	for _, queryContract := range contract.Queries {
		if queryContract.Name == string(invocation.Query) {
			queryCount++
		}
	}
	if queryCount == 0 {
		out = append(out, errFor(
			patternSubject,
			"UNKNOWN_REFERENCE_CONTRACT_QUERY",
			"reference Pattern %sが利用するQuery %sはContract %sにありません",
			pattern.ID,
			invocation.Query,
			contract.ID,
		))
	} else if queryCount > 1 {
		out = append(out, errFor(
			patternSubject,
			"AMBIGUOUS_REFERENCE_CONTRACT_QUERY",
			"reference Pattern %sが利用するQuery %sはContract %sに%d件あり、一意に解決できません",
			pattern.ID,
			invocation.Query,
			contract.ID,
			queryCount,
		))
	}

	return out
}
