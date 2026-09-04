package main

import (
	"fmt"
	"strings"
)

type goEnvironmentEvidence struct {
	ExternalWorkspaceActive bool `json:"externalWorkspaceActive"`
	GoFlagsEmpty            bool `json:"goFlagsEmpty"`
}

// inspectGoEnvironmentは、candidate revision外のGo workspaceやGOFLAGSが
// final Evidenceのcompile / test入力へ介入しないことを確認する。
//
// GOWORKの実pathやGOFLAGS本文は、local pathや任意の値を含み得るためEvidenceへ保存しない。
// 成功・失敗の判定に必要な非機密booleanだけを返す。
func inspectGoEnvironment(output commandOutputFunc) (goEnvironmentEvidence, error) {
	var result goEnvironmentEvidence

	goWork, err := output("go", "env", "GOWORK")
	if err != nil {
		return result, fmt.Errorf("Go workspace状態を確認できません: %w", err)
	}
	goWork = strings.TrimSpace(goWork)
	result.ExternalWorkspaceActive = goWork != "" && !strings.EqualFold(goWork, "off")
	if result.ExternalWorkspaceActive {
		return result, fmt.Errorf("外部Go workspaceが有効なためGate B最終Evidenceを実行しません。Repositoryのgo.modだけを基準にするため、GOWORK=offまたはworkspace未使用の環境で再実行してください")
	}

	goFlags, err := output("go", "env", "GOFLAGS")
	if err != nil {
		return result, fmt.Errorf("effective GOFLAGSを確認できません: %w", err)
	}
	result.GoFlagsEmpty = strings.TrimSpace(goFlags) == ""
	if !result.GoFlagsEmpty {
		return result, fmt.Errorf("effective GOFLAGSが設定されているためGate B最終Evidenceを実行しません。-overlay / -modfile / -tags等によるRepository外の入力差を排除するため、GOFLAGSを空にして再実行してください")
	}

	return result, nil
}
