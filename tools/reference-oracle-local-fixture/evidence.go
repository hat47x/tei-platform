package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"tei/internal/database"
	"tei/internal/database/externalqueryevidence"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/realization"
	"tei/internal/sourceindex"
)

// 第一Referenceの対象Contract Queryである。Workspaceに既に存在するDatabase / Contract / Queryを指す。
var fixtureTarget = realization.ContractQueryTarget{
	Database: "master_data",
	Contract: "master_data.customer",
	Query:    "find_by_id",
}

const (
	attributeCustomerID   = "customer.customer.customer_id"
	attributeCustomerName = "customer.customer.name"
)

// authoredRealizationは、このtoolの作者が明示的に書いたContract Query SQL realizationである。
//
// これは使い捨てfixture tableに対する実装であり、external-observedな実際のmaster_data資産の
// physical objectやcolumnを観測して生成したものではない。観測Evidenceから生成する経路は持たない。
func authoredRealization(qualifiedTable string) realization.ContractQuerySQLRealization {
	return realization.ContractQuerySQLRealization{
		Target:   fixtureTarget,
		Provider: environment.ProviderOracle,
		Source: "SELECT CUSTOMER_ID, CUSTOMER_NAME FROM " + qualifiedTable +
			" WHERE CUSTOMER_ID = :id",
		Inputs: []realization.ContractQueryInputParameterMapping{
			{Input: "id", Parameter: "id"},
		},
		ResultColumns: []query.ResultColumnMapping{
			{Column: "CUSTOMER_ID", Attribute: attributeCustomerID},
			{Column: "CUSTOMER_NAME", Attribute: attributeCustomerName},
		},
	}
}

// findQueryContractは、Workspaceから対象Contract Queryのsignatureを一意に解決する。
func findQueryContract(w ir.World, target realization.ContractQueryTarget) (database.QueryContract, error) {
	var found database.QueryContract
	count := 0
	for _, db := range w.Databases {
		if db.ID != target.Database {
			continue
		}
		for _, contract := range db.Contracts {
			if contract.ID != target.Contract {
				continue
			}
			for _, candidate := range contract.Queries {
				if candidate.Name == string(target.Query) {
					found = candidate
					count++
				}
			}
		}
	}
	if count != 1 {
		return database.QueryContract{}, fmt.Errorf("Contract Query %sをWorkspaceから一意に解決できません: %d件", target.Identity(), count)
	}
	return found, nil
}

// findAuthoredRealizationは、target + providerに完全一致する明示realizationを返す。
// 存在しなければEvidenceから補完せず停止する。
func findAuthoredRealization(definitions []realization.ContractQuerySQLRealization, target realization.ContractQueryTarget, provider environment.DatabaseProvider) (realization.ContractQuerySQLRealization, error) {
	var found realization.ContractQuerySQLRealization
	count := 0
	for _, candidate := range definitions {
		if candidate.Target == target && candidate.Provider == provider {
			found = candidate
			count++
		}
	}
	switch count {
	case 0:
		return realization.ContractQuerySQLRealization{}, fmt.Errorf("Contract Query %s / provider=%sに対応する明示SQL realizationがありません。Evidenceからrealizationを生成・補完しません", target.Identity(), provider)
	case 1:
		return found, nil
	default:
		return realization.ContractQuerySQLRealization{}, fmt.Errorf("Contract Query %s / provider=%sに対応するSQL realizationが複数あります: %d件", target.Identity(), provider, count)
	}
}

// observedFactsは、Evidenceへ写すために実際に観測した事実だけを保持する。
type observedFacts struct {
	RunID           string
	ObservedAt      time.Time
	ObjectKind      string
	ObjectIdentity  string
	InputParameter  string // 実行時にdatabase/sqlへ渡されたnamed parameter名
	InputPhysical   string // dictionaryから観測した列の型表記
	RawColumns      []observedColumn
	ReadOnlyMech    string
	ReadOnlyRef     string
	SourceRef       string
	ProcessingNote  string
	NativeSourceRef string
}

type observedColumn struct {
	Name         string
	PhysicalType string
	Nullable     *bool
}

// buildEvidenceは、観測事実とauthoring済みrealizationから、外部Evidence intake用のEvidenceを組み立てる。
//
// Contract inputとraw列に対するSemantic attributeの対応はrealizationが明示している値を使う。
// 観測側は「実際にそのparameter名・raw列名が使われ、返った」ことの確認に限る。
// Evidenceに基づいてrealizationを補完する経路はない。
func buildEvidence(def realization.ContractQuerySQLRealization, facts observedFacts) (externalqueryevidence.Evidence, error) {
	observedColumns := make(map[string]observedColumn, len(facts.RawColumns))
	for _, column := range facts.RawColumns {
		observedColumns[column.Name] = column
	}
	if len(def.Inputs) != 1 || def.Inputs[0].Parameter != facts.InputParameter {
		return externalqueryevidence.Evidence{}, fmt.Errorf("実行で観測したnamed parameterがauthoring済みrealizationの入力と一致しません")
	}

	inputs := []externalqueryevidence.InputObservation{{
		Input:        def.Inputs[0].Input,
		Parameter:    facts.InputParameter,
		PhysicalType: facts.InputPhysical,
	}}
	results := make([]externalqueryevidence.ResultObservation, 0, len(def.ResultColumns))
	for _, mapping := range def.ResultColumns {
		column, ok := observedColumns[mapping.Column]
		if !ok {
			return externalqueryevidence.Evidence{}, fmt.Errorf("authoring済みrealizationのraw列 %qが実行結果で観測されていません", mapping.Column)
		}
		results = append(results, externalqueryevidence.ResultObservation{
			Column:       mapping.Column,
			Attribute:    mapping.Attribute,
			PhysicalType: column.PhysicalType,
			Nullable:     column.Nullable,
		})
	}
	if len(facts.RawColumns) != len(def.ResultColumns) {
		return externalqueryevidence.Evidence{}, fmt.Errorf("実行で観測したraw列数%dがrealizationのmapping数%dと一致しません", len(facts.RawColumns), len(def.ResultColumns))
	}

	return externalqueryevidence.Evidence{
		Target:   def.Target,
		Provider: def.Provider,
		Object:   externalqueryevidence.PhysicalObject{Kind: facts.ObjectKind, Identity: facts.ObjectIdentity},
		Inputs:   inputs,
		Results:  results,
		NativeSource: externalqueryevidence.NativeSourceProof{
			EvidenceRef: facts.NativeSourceRef,
			Fingerprint: sourceindex.Fingerprint([]byte(def.Source)),
		},
		Provenance: externalqueryevidence.Provenance{
			SourceRef:  facts.SourceRef,
			ObservedAt: facts.ObservedAt.UTC().Format(time.RFC3339),
			Processing: facts.ProcessingNote,
		},
		ReadOnly: externalqueryevidence.ReadOnlyProof{
			Mechanism:   facts.ReadOnlyMech,
			EvidenceRef: facts.ReadOnlyRef,
		},
	}, nil
}

// intakeResultは、既存のexternalqueryevidenceによる受入検査の結果である。
type intakeResult struct {
	// Acceptedは、既存の厳格loaderとValidateAgainstを通過したことだけを示す。
	Accepted                    bool   `json:"accepted"`
	EvidenceArtifactFingerprint string `json:"evidenceArtifactFingerprint"`
	Target                      string `json:"target"`
	Provider                    string `json:"provider"`
	Error                       string `json:"error,omitempty"`
}

// intakeEvidenceは、Evidenceを既存の厳格loaderで読み直し、Workspaceとauthoring済みrealizationとの照合を行う。
// 検査自体は既存のexternalqueryevidence.ValidateAgainstに委譲し、このtool側で緩めない。
func intakeEvidence(w ir.World, definitions []realization.ContractQuerySQLRealization, evidence externalqueryevidence.Evidence) (intakeResult, []byte, error) {
	raw, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return intakeResult{}, nil, fmt.Errorf("Evidence JSONを作成できません: %w", err)
	}
	sum := sha256.Sum256(raw)
	result := intakeResult{
		EvidenceArtifactFingerprint: "sha256:" + hex.EncodeToString(sum[:]),
		Target:                      evidence.Target.Identity(),
		Provider:                    string(evidence.Provider),
	}

	loaded, err := externalqueryevidence.Load(raw)
	if err != nil {
		result.Error = err.Error()
		return result, raw, nil
	}
	def, err := findAuthoredRealization(definitions, loaded.Target, loaded.Provider)
	if err != nil {
		result.Error = err.Error()
		return result, raw, nil
	}
	if err := externalqueryevidence.ValidateAgainst(w, def, loaded); err != nil {
		result.Error = err.Error()
		return result, raw, nil
	}
	result.Accepted = true
	return result, raw, nil
}
