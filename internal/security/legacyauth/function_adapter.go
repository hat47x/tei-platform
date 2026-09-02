package legacyauth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"tei/internal/core"
	"tei/internal/semantic/capabilityinvoke"
)

// AuthenticateFunctionは、既存関数、stored procedure、外部APIなど、
// 既存システム側で認証判断まで完了する処理を呼び出すための境界である。
// ApplicationやDriverへlegacy固有のreturn codeを公開しない。
type AuthenticateFunction interface {
	Authenticate(context.Context, string, string) (FunctionResult, error)
}

// FunctionResultは、既存認証関数が返す最小の結果を表す。
// ReturnCodeはlegacy側の値であり、Semantic CapabilityのOutcomeではない。
type FunctionResult struct {
	ReturnCode string
	SubjectID  string
}

// FunctionMappingは、既存認証関数のreturn codeをsecurity.authenticateの
// Business Resultへ対応付ける。Sourceには、既存関数やAPIを識別できる
// secretではない名前だけを保持する。
type FunctionMapping struct {
	Source            string            `json:"source"`
	AuthenticatedCode string            `json:"authenticated_code"`
	DeniedCodes       map[string]string `json:"denied_codes"`
	sourcePath        string
}

type functionMappingDocument struct {
	Schema            string            `json:"schema"`
	Source            string            `json:"source"`
	AuthenticatedCode string            `json:"authenticated_code"`
	DeniedCodes       map[string]string `json:"denied_codes"`
}

// LoadFunctionMappingは、既存認証関数のreturn code対応をReference fixtureから読む。
// Mappingは既存関数の意味をCanonicalへ複製するためではなく、どのlegacy結果を
// どのSemantic Outcomeへ接続したかという来歴を確認するために使う。
func LoadFunctionMapping(path string) (FunctionMapping, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FunctionMapping{}, fmt.Errorf("legacy authentication function mappingを読み込めません: %w", err)
	}
	var doc functionMappingDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return FunctionMapping{}, fmt.Errorf("legacy authentication function mappingを解析できません: %w", err)
	}
	if doc.Schema == "" {
		return FunctionMapping{}, fmt.Errorf("legacy authentication function mappingにschema識別子がありません")
	}
	if doc.Source == "" {
		return FunctionMapping{}, fmt.Errorf("legacy authentication function mappingにsourceがありません")
	}
	if doc.AuthenticatedCode == "" {
		return FunctionMapping{}, fmt.Errorf("legacy authentication function mappingにauthenticated_codeがありません")
	}
	if _, duplicated := doc.DeniedCodes[doc.AuthenticatedCode]; duplicated {
		return FunctionMapping{}, fmt.Errorf("legacy return code %qがauthenticatedとdeniedの両方に定義されています", doc.AuthenticatedCode)
	}
	for code, reason := range doc.DeniedCodes {
		if code == "" || reason == "" {
			return FunctionMapping{}, fmt.Errorf("legacy authentication function mappingのdenied code / reasonに空値があります")
		}
	}
	return FunctionMapping{
		Source:            doc.Source,
		AuthenticatedCode: doc.AuthenticatedCode,
		DeniedCodes:       doc.DeniedCodes,
		sourcePath:        path,
	}, nil
}

// SourcePathは、return code対応を読み込んだReference sourceを返す。
func (m FunctionMapping) SourcePath() string {
	return m.sourcePath
}

// FunctionAdapterは、認証判断を既存関数側へ委ね、そのlegacy結果だけを
// security.authenticateのCanonical契約へ対応付ける実現方式である。
//
// 既存user recordをTEI側で読むAdapterとは構成が異なるが、Applicationから見える
// Capability identityとBusiness Resultは同じまま保つ。
type FunctionAdapter struct {
	Function AuthenticateFunction
	Mapping  FunctionMapping
}

func (a FunctionAdapter) Invoke(ctx context.Context, capability core.CanonicalID, raw any) (capabilityinvoke.BusinessResult, error) {
	if capability != AuthenticationCapability {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authentication function adapterはcapability %sを扱えません", capability)
	}
	input, ok := raw.(Input)
	if !ok {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("security.authenticateの入力型がlegacyauth.Inputではありません")
	}
	if a.Function == nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authentication functionが指定されていません")
	}
	if a.Mapping.Source == "" || a.Mapping.AuthenticatedCode == "" {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authentication function mappingが不完全です")
	}

	legacyResult, err := a.Function.Authenticate(ctx, input.LoginID, input.Credential)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authentication functionを実行できません: %w", err)
	}

	if legacyResult.ReturnCode == a.Mapping.AuthenticatedCode {
		if legacyResult.SubjectID == "" {
			return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authentication functionが認証成功を返しましたがsubject identityがありません")
		}
		return capabilityinvoke.BusinessResult{
			Capability: AuthenticationCapability,
			Mode:       capabilityinvoke.ResultOutcome,
			Outcome:    "authenticated",
			Payload:    AuthenticatedSubject{SubjectID: legacyResult.SubjectID},
		}, nil
	}

	if reason, ok := a.Mapping.DeniedCodes[legacyResult.ReturnCode]; ok {
		return denied(reason), nil
	}

	return capabilityinvoke.BusinessResult{}, fmt.Errorf(
		"legacy authentication function %sが未対応のreturn code %qを返しました",
		a.Mapping.Source,
		legacyResult.ReturnCode,
	)
}
