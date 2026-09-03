package realization

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/environment"
	"tei/internal/query"
)

// ContractQueryInputParameterMappingは、Contract QueryのSemantic input名を
// native SQL realizationが利用する物理parameter名へ明示的に対応付ける。
//
// input名とparameter名が同じ場合でも、この対応は省略しない。Driverやprovider adapterが
// 名前の一致から暗黙に接続することを防ぐためである。
type ContractQueryInputParameterMapping struct {
	Input     core.LocalName `json:"input"`
	Parameter string         `json:"parameter"`
}

// ContractQuerySQLRealizationは、一つのDatabase-owned Contract Queryをnative SQLで実現するための
// provider固有情報を保持する。Query signatureやSemantic Relationとは別の層に置き、物理SQL、
// parameter名、raw result列名をCanonicalな業務意味へ持ち込まない。
type ContractQuerySQLRealization struct {
	Target        ContractQueryTarget                  `json:"target"`
	Provider      environment.DatabaseProvider         `json:"provider"`
	Source        string                               `json:"source"`
	Inputs        []ContractQueryInputParameterMapping `json:"inputs,omitempty"`
	ResultColumns []query.ResultColumnMapping          `json:"resultColumns,omitempty"`
}

// Identityは、第一段階のSQL realizationをtarget + providerで識別する。
// 同じproviderに複数variantが必要になるまではvariant IDを先回りで導入しない。
func (r ContractQuerySQLRealization) Identity() string {
	return r.Target.Identity() + "/sql/" + string(r.Provider)
}

// ValidateContractは、SQL realizationとDatabase Contract Query signatureの対応が
// 明示情報だけで一意に決まることを検査する。SQL本文を解析してparameterやresult列を推測しない。
func (r ContractQuerySQLRealization) ValidateContract(contract database.QueryContract) error {
	if !r.Target.Valid() {
		return fmt.Errorf("Contract Query SQL realizationのtarget identityが不完全です")
	}
	if !r.Provider.Known() {
		return fmt.Errorf("Contract Query SQL realization %sのDatabase provider %qは未定義です", r.Target.Identity(), r.Provider)
	}
	if r.Source == "" {
		return fmt.Errorf("Contract Query SQL realization %sにnative SQL sourceがありません", r.Identity())
	}
	if string(r.Target.Query) != contract.Name {
		return fmt.Errorf("Contract Query SQL realization %sのQuery %sとsignature %sが一致しません", r.Identity(), r.Target.Query, contract.Name)
	}

	declaredInputs := make(map[core.LocalName]database.QueryInput, len(contract.Inputs))
	for _, input := range contract.Inputs {
		name, err := core.ParseLocalName(string(input.Name))
		if err != nil {
			return fmt.Errorf("Contract Query %sのinput名 %qをSQL realizationへ接続できません: %w", contract.Name, input.Name, err)
		}
		if _, exists := declaredInputs[name]; exists {
			return fmt.Errorf("Contract Query %sのinput %sが重複しています", contract.Name, name)
		}
		declaredInputs[name] = input
	}

	usedInputs := make(map[core.LocalName]struct{}, len(r.Inputs))
	usedParameters := make(map[string]struct{}, len(r.Inputs))
	for _, mapping := range r.Inputs {
		if mapping.Input == "" {
			return fmt.Errorf("Contract Query SQL realization %sにinput名のないparameter mappingがあります", r.Identity())
		}
		if mapping.Parameter == "" {
			return fmt.Errorf("Contract Query SQL realization %sのinput %sに物理parameter名がありません", r.Identity(), mapping.Input)
		}
		if _, exists := declaredInputs[mapping.Input]; !exists {
			return fmt.Errorf("Contract Query SQL realization %sがsignatureにないinput %sをparameterへ対応付けています", r.Identity(), mapping.Input)
		}
		if _, exists := usedInputs[mapping.Input]; exists {
			return fmt.Errorf("Contract Query SQL realization %sでinput %sのparameter mappingが重複しています", r.Identity(), mapping.Input)
		}
		if _, exists := usedParameters[mapping.Parameter]; exists {
			return fmt.Errorf("Contract Query SQL realization %sで物理parameter %qが重複しています", r.Identity(), mapping.Parameter)
		}
		usedInputs[mapping.Input] = struct{}{}
		usedParameters[mapping.Parameter] = struct{}{}
	}
	for input := range declaredInputs {
		if _, exists := usedInputs[input]; !exists {
			return fmt.Errorf("Contract Query SQL realization %sにinput %sのparameter mappingがありません", r.Identity(), input)
		}
	}

	if len(r.ResultColumns) == 0 {
		return fmt.Errorf("Contract Query SQL realization %sにresult column mappingがありません", r.Identity())
	}
	usedColumns := make(map[string]struct{}, len(r.ResultColumns))
	usedAttributes := make(map[core.CanonicalID]struct{}, len(r.ResultColumns))
	for _, mapping := range r.ResultColumns {
		if mapping.Column == "" {
			return fmt.Errorf("Contract Query SQL realization %sに名前のないraw result列があります", r.Identity())
		}
		if mapping.Attribute == "" {
			return fmt.Errorf("Contract Query SQL realization %sのraw result列 %qにSemantic attributeがありません", r.Identity(), mapping.Column)
		}
		if _, exists := usedColumns[mapping.Column]; exists {
			return fmt.Errorf("Contract Query SQL realization %sでraw result列 %qのmappingが重複しています", r.Identity(), mapping.Column)
		}
		if _, exists := usedAttributes[mapping.Attribute]; exists {
			return fmt.Errorf("Contract Query SQL realization %sでSemantic attribute %sのmappingが重複しています", r.Identity(), mapping.Attribute)
		}
		usedColumns[mapping.Column] = struct{}{}
		usedAttributes[mapping.Attribute] = struct{}{}
	}
	return nil
}

// MapRuntimeParametersは、Contract Query input名で受け取った一回限りのruntime値を、
// SQL realizationで明示した物理parameter名へ対応付ける。値自体は変換せず、そのまま渡す。
//
// mappingされていないruntime inputや不足inputを黙って無視せず、SQL parameter名との一致からも補完しない。
func (r ContractQuerySQLRealization) MapRuntimeParameters(parameters map[string]any) (map[string]any, error) {
	mapped := make(map[string]any, len(r.Inputs))
	usedInputs := make(map[string]struct{}, len(r.Inputs))
	for _, mapping := range r.Inputs {
		input := string(mapping.Input)
		if input == "" || mapping.Parameter == "" {
			return nil, fmt.Errorf("Contract Query SQL realization %sのparameter mappingが不完全です", r.Identity())
		}
		value, exists := parameters[input]
		if !exists {
			return nil, fmt.Errorf("Contract Query SQL realization %sへ渡すinput %sのruntime値がありません", r.Identity(), input)
		}
		if _, exists := mapped[mapping.Parameter]; exists {
			return nil, fmt.Errorf("Contract Query SQL realization %sで物理parameter %qへ複数のruntime値を渡そうとしています", r.Identity(), mapping.Parameter)
		}
		mapped[mapping.Parameter] = value
		usedInputs[input] = struct{}{}
	}
	for input := range parameters {
		if _, exists := usedInputs[input]; !exists {
			return nil, fmt.Errorf("Contract Query SQL realization %sにruntime input %sのparameter mappingがありません", r.Identity(), input)
		}
	}
	return mapped, nil
}
