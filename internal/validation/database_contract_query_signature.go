package validation

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

// ValidateDatabaseContractQuerySignaturesは、signatureが明示されたDatabase Contract Queryについて、
// inputのSemantic型、result Semantic Data、result cardinalityの参照整合性を確認する。
// signature未定義の既存Queryはこの段階では拒否せず、Referenceから段階的に契約を追加できるようにする。
func ValidateDatabaseContractQuerySignatures(w ir.World) []Diagnostic {
	semanticTypes := map[core.CanonicalID]struct{}{}
	semanticData := map[core.CanonicalID]struct{}{}
	for _, ns := range w.Semantics {
		for _, typ := range ns.Types {
			semanticTypes[typ.ID] = struct{}{}
		}
		for _, data := range ns.Data {
			semanticData[data.ID] = struct{}{}
		}
	}

	var out []Diagnostic
	for _, db := range w.Databases {
		for _, contract := range db.Contracts {
			for _, query := range contract.Queries {
				if len(query.Inputs) == 0 && query.Result == "" && query.ResultCardinality == "" {
					continue
				}

				querySubject := subject.Subject{
					Kind: "database-contract-query",
					ID:   fmt.Sprintf("%s/%s", contract.ID, query.Name),
				}
				seenInputs := map[core.LocalName]struct{}{}

				for _, input := range query.Inputs {
					if input.Name == "" {
						out = append(out, errFor(
							querySubject,
							"DATABASE_CONTRACT_QUERY_INPUT_NAME_REQUIRED",
							"Database Contract Query %s / %s has an input without a name",
							contract.ID,
							query.Name,
						))
					} else {
						if _, err := core.ParseLocalName(string(input.Name)); err != nil {
							out = append(out, errFor(
								querySubject,
								"INVALID_DATABASE_CONTRACT_QUERY_INPUT_NAME",
								"Database Contract Query %s / %s has input name %q, which is not a valid LocalName",
								contract.ID,
								query.Name,
								input.Name,
							))
						}
						if _, duplicate := seenInputs[input.Name]; duplicate {
							out = append(out, errFor(
								querySubject,
								"DUPLICATE_DATABASE_CONTRACT_QUERY_INPUT",
								"Database Contract Query %s / %s declares input %s more than once",
								contract.ID,
								query.Name,
								input.Name,
							))
						} else {
							seenInputs[input.Name] = struct{}{}
						}
					}

					if input.Type == "" {
						out = append(out, errFor(
							querySubject,
							"DATABASE_CONTRACT_QUERY_INPUT_TYPE_REQUIRED",
							"Database Contract Query %s / %s input %s needs a Semantic type",
							contract.ID,
							query.Name,
							input.Name,
						))
					} else if _, ok := semanticTypes[input.Type]; !ok {
						out = append(out, errFor(
							querySubject,
							"UNKNOWN_DATABASE_CONTRACT_QUERY_INPUT_TYPE",
							"Database Contract Query %s / %s input %s refers to Semantic type %s, which does not exist",
							contract.ID,
							query.Name,
							input.Name,
							input.Type,
						))
					}
				}

				if query.Result == "" {
					out = append(out, errFor(
						querySubject,
						"DATABASE_CONTRACT_QUERY_RESULT_REQUIRED",
						"Database Contract Query %s / %s has a signature, so it needs a result Semantic Data",
						contract.ID,
						query.Name,
					))
				} else if _, ok := semanticData[query.Result]; !ok {
					out = append(out, errFor(
						querySubject,
						"UNKNOWN_DATABASE_CONTRACT_QUERY_RESULT",
						"Database Contract Query %s / %s refers to result Semantic Data %s, which does not exist",
						contract.ID,
						query.Name,
						query.Result,
					))
				}

				if query.ResultCardinality == "" {
					out = append(out, errFor(
						querySubject,
						"DATABASE_CONTRACT_QUERY_RESULT_CARDINALITY_REQUIRED",
						"Database Contract Query %s / %s has a signature, so it needs a result cardinality",
						contract.ID,
						query.Name,
					))
				} else if !query.ResultCardinality.Valid() {
					out = append(out, errFor(
						querySubject,
						"INVALID_DATABASE_CONTRACT_QUERY_RESULT_CARDINALITY",
						"Database Contract Query %s / %s has result cardinality %q, which is undefined",
						contract.ID,
						query.Name,
						query.ResultCardinality,
					))
				}
			}
		}
	}

	return out
}
