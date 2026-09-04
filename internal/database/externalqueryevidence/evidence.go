package externalqueryevidence

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/sourceindex"
)

// Evidenceは、external-observed Database上のContract Queryを実物理資産へ接続するために
// 観測した事実と安全境界を保持する非CanonicalなEvidenceである。
//
// この型はir.Worldへ取り込まず、Realizationを生成・補完する入力にも利用しない。
// Evidenceは、別途authoringされたContractQuerySQLRealizationが観測事実と一致するかを
// 受入時に検証するためだけに使う。
type Evidence struct {
	Target       realization.ContractQueryTarget `json:"target"`
	Provider     environment.DatabaseProvider    `json:"provider"`
	Object       PhysicalObject                  `json:"object"`
	Inputs       []InputObservation              `json:"inputs"`
	Results      []ResultObservation             `json:"results"`
	NativeSource NativeSourceProof               `json:"nativeSource"`
	Provenance   Provenance                      `json:"provenance"`
	ReadOnly     ReadOnlyProof                   `json:"readOnly"`
}

// PhysicalObjectは、実際に観測したtable / view / API / stored procedure等を指す。
// Kindを固定enumへ閉じないのは、外部資産の形をTEI側の先行taxonomyへ押し込めないためである。
type PhysicalObject struct {
	Kind     string `json:"kind"`
	Identity string `json:"identity"`
}

// InputObservationはContract inputと観測済みphysical parameterの対応を記録する。
// PhysicalTypeはEvidenceとして記録できるが、Canonical Query signatureの型を書き換えない。
type InputObservation struct {
	Input        core.LocalName `json:"input"`
	Parameter    string         `json:"parameter"`
	PhysicalType string         `json:"physicalType,omitempty"`
	Nullable     *bool          `json:"nullable,omitempty"`
}

// ResultObservationは実raw result列とSemantic attributeの対応を記録する。
// Semantic attributeは、既にauthoringされたrealizationとの一致確認にのみ用いる。
type ResultObservation struct {
	Column       string           `json:"column"`
	Attribute    core.CanonicalID `json:"attribute"`
	PhysicalType string           `json:"physicalType,omitempty"`
	Nullable     *bool            `json:"nullable,omitempty"`
}

// NativeSourceProofは、このEvidenceがどのnative SQL本文を確認したものかを固定する。
// EvidenceRefは元SQL・承認記録・抽出資料等への再確認可能な参照であり、Fingerprintは
// authoring済みContractQuerySQLRealization.Sourceと完全一致しなければならない。
type NativeSourceProof struct {
	EvidenceRef string `json:"evidenceRef"`
	Fingerprint string `json:"fingerprint"`
}

// Provenanceは、物理事実をどこから確認し、Reference用Evidenceへどのように加工したかを残す。
// SourceRefはsecretそのものではなく、再確認可能な文書・schema dump・ticket等への参照である。
type Provenance struct {
	SourceRef  string `json:"sourceRef"`
	ObservedAt string `json:"observedAt,omitempty"`
	Processing string `json:"processing"`
}

// ReadOnlyProofは「SQLがSELECTに見える」ことではなく、書込み不能を実効的に強制する境界のEvidenceを示す。
// Mechanismはread-only credential / DB permission / read-only session / read-only adapter等の実際の方式を記述する。
type ReadOnlyProof struct {
	Mechanism   string `json:"mechanism"`
	EvidenceRef string `json:"evidenceRef"`
}

// ValidateAgainstはexternal-observed物理Evidenceが、既存Canonical Worldと明示SQL realizationに
// 矛盾しないことを検査する。
//
// SQL本文は解析しない。物理object名からparameterやSemantic attributeを推測しない。
// Evidenceにしかない情報からRealizationを補完することもしない。
func ValidateAgainst(w ir.World, realizationDef realization.ContractQuerySQLRealization, evidence Evidence) error {
	if !evidence.Target.Valid() {
		return fmt.Errorf("external Contract Query Evidenceのtarget identityが不完全です")
	}
	if evidence.Target != realizationDef.Target {
		return fmt.Errorf("external Contract Query Evidence target %sとSQL realization target %sが一致しません", evidence.Target.Identity(), realizationDef.Target.Identity())
	}
	if !evidence.Provider.Known() || evidence.Provider == environment.ProviderGeneric {
		return fmt.Errorf("external Contract Query Evidence %sのprovider %qは実物理Evidenceとして使用できません", evidence.Target.Identity(), evidence.Provider)
	}
	if evidence.Provider != realizationDef.Provider {
		return fmt.Errorf("external Contract Query Evidence %sのprovider %sとSQL realization provider %sが一致しません", evidence.Target.Identity(), evidence.Provider, realizationDef.Provider)
	}

	db, queryContract, err := externalObservedQuery(w, evidence.Target)
	if err != nil {
		return err
	}
	if db.Ownership != database.OwnershipExternalObserved {
		return fmt.Errorf("Database %sはexternal-observedではないためexternal Contract Query Evidenceの対象にできません: ownership=%s", db.ID, db.Ownership)
	}
	if err := realizationDef.ValidateContract(queryContract); err != nil {
		return fmt.Errorf("SQL realization %sをexternal Evidenceと照合できません: %w", realizationDef.Identity(), err)
	}

	if strings.TrimSpace(evidence.Object.Kind) == "" || strings.TrimSpace(evidence.Object.Identity) == "" {
		return fmt.Errorf("external Contract Query Evidence %sに観測済みphysical objectがありません", evidence.Target.Identity())
	}
	if strings.TrimSpace(evidence.NativeSource.EvidenceRef) == "" || strings.TrimSpace(evidence.NativeSource.Fingerprint) == "" {
		return fmt.Errorf("external Contract Query Evidence %sにnative SQL sourceのEvidence参照またはfingerprintがありません", evidence.Target.Identity())
	}
	wantFingerprint := sourceindex.Fingerprint([]byte(realizationDef.Source))
	if evidence.NativeSource.Fingerprint != wantFingerprint {
		return fmt.Errorf("external Contract Query Evidence %sのnative SQL fingerprintがstaleです: have %s want %s", evidence.Target.Identity(), evidence.NativeSource.Fingerprint, wantFingerprint)
	}
	if strings.TrimSpace(evidence.Provenance.SourceRef) == "" {
		return fmt.Errorf("external Contract Query Evidence %sにphysical factsのsource provenanceがありません", evidence.Target.Identity())
	}
	if strings.TrimSpace(evidence.Provenance.Processing) == "" {
		return fmt.Errorf("external Contract Query Evidence %sにEvidence加工方法がありません", evidence.Target.Identity())
	}
	if strings.TrimSpace(evidence.ReadOnly.Mechanism) == "" || strings.TrimSpace(evidence.ReadOnly.EvidenceRef) == "" {
		return fmt.Errorf("external Contract Query Evidence %sに実効的なread-only保証Evidenceがありません", evidence.Target.Identity())
	}

	if err := validateInputs(realizationDef, evidence.Inputs); err != nil {
		return err
	}
	if err := validateResults(realizationDef, evidence.Results); err != nil {
		return err
	}
	return nil
}

func validateInputs(realizationDef realization.ContractQuerySQLRealization, observed []InputObservation) error {
	if len(observed) != len(realizationDef.Inputs) {
		return fmt.Errorf("external Contract Query Evidence %sのinput observation件数%dがSQL realizationのmapping件数%dと一致しません", realizationDef.Target.Identity(), len(observed), len(realizationDef.Inputs))
	}
	byInput := make(map[core.LocalName]InputObservation, len(observed))
	for _, observation := range observed {
		if observation.Input == "" || strings.TrimSpace(observation.Parameter) == "" {
			return fmt.Errorf("external Contract Query Evidence %sに不完全なinput observationがあります", realizationDef.Target.Identity())
		}
		if _, exists := byInput[observation.Input]; exists {
			return fmt.Errorf("external Contract Query Evidence %sでinput %sのobservationが重複しています", realizationDef.Target.Identity(), observation.Input)
		}
		byInput[observation.Input] = observation
	}
	for _, mapping := range realizationDef.Inputs {
		observation, exists := byInput[mapping.Input]
		if !exists {
			return fmt.Errorf("external Contract Query Evidence %sにinput %sのobservationがありません", realizationDef.Target.Identity(), mapping.Input)
		}
		if observation.Parameter != mapping.Parameter {
			return fmt.Errorf("external Contract Query Evidence %sのinput %sはphysical parameter %qを観測していますが、SQL realizationは%qを定義しています", realizationDef.Target.Identity(), mapping.Input, observation.Parameter, mapping.Parameter)
		}
	}
	return nil
}

func validateResults(realizationDef realization.ContractQuerySQLRealization, observed []ResultObservation) error {
	if len(observed) != len(realizationDef.ResultColumns) {
		return fmt.Errorf("external Contract Query Evidence %sのresult observation件数%dがSQL realizationのmapping件数%dと一致しません", realizationDef.Target.Identity(), len(observed), len(realizationDef.ResultColumns))
	}
	byColumn := make(map[string]ResultObservation, len(observed))
	for _, observation := range observed {
		column := strings.TrimSpace(observation.Column)
		if column == "" || observation.Attribute == "" {
			return fmt.Errorf("external Contract Query Evidence %sに不完全なresult observationがあります", realizationDef.Target.Identity())
		}
		if _, exists := byColumn[column]; exists {
			return fmt.Errorf("external Contract Query Evidence %sでraw result列 %qのobservationが重複しています", realizationDef.Target.Identity(), column)
		}
		byColumn[column] = observation
	}
	for _, mapping := range realizationDef.ResultColumns {
		observation, exists := byColumn[mapping.Column]
		if !exists {
			return fmt.Errorf("external Contract Query Evidence %sにraw result列 %qのobservationがありません", realizationDef.Target.Identity(), mapping.Column)
		}
		if observation.Attribute != mapping.Attribute {
			return fmt.Errorf("external Contract Query Evidence %sのraw result列 %qはSemantic attribute %sへの対応を観測していますが、SQL realizationは%sを定義しています", realizationDef.Target.Identity(), mapping.Column, observation.Attribute, mapping.Attribute)
		}
	}
	return nil
}

func externalObservedQuery(w ir.World, target realization.ContractQueryTarget) (database.Database, database.QueryContract, error) {
	var db database.Database
	dbCount := 0
	for _, candidate := range w.Databases {
		if candidate.ID != target.Database {
			continue
		}
		db = candidate
		dbCount++
	}
	if dbCount != 1 {
		return database.Database{}, database.QueryContract{}, fmt.Errorf("external Contract Query Evidence targetのDatabase %sを一意に解決できません: %d件", target.Database, dbCount)
	}

	var contract database.Contract
	contractCount := 0
	for _, candidate := range db.Contracts {
		if candidate.ID != target.Contract {
			continue
		}
		contract = candidate
		contractCount++
	}
	if contractCount != 1 {
		return database.Database{}, database.QueryContract{}, fmt.Errorf("external Contract Query Evidence targetのContract %sをDatabase %sで一意に解決できません: %d件", target.Contract, db.ID, contractCount)
	}

	var queryContract database.QueryContract
	queryCount := 0
	for _, candidate := range contract.Queries {
		if candidate.Name != string(target.Query) {
			continue
		}
		queryContract = candidate
		queryCount++
	}
	if queryCount != 1 {
		return database.Database{}, database.QueryContract{}, fmt.Errorf("external Contract Query Evidence targetのQuery %sをContract %sで一意に解決できません: %d件", target.Query, contract.ID, queryCount)
	}
	return db, queryContract, nil
}
