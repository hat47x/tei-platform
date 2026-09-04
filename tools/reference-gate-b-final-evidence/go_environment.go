package main

import (
	"fmt"
	"os"
	"strings"
)

type goEnvironmentEvidence struct {
	ExternalWorkspaceActive bool `json:"externalWorkspaceActive"`
	GoFlagsEmpty            bool `json:"goFlagsEmpty"`
	GoEnvDisabled           bool `json:"goEnvDisabled"`
}

type setEnvFunc func(string, string) error
type lookupEnvFunc func(string) (string, bool)

// inspectGoEnvironmentは、final runner自身をcompileする段階からuser-level Go environment fileを
// 暗黙入力へしないため、process environmentにGOENV=offが明示されていることを最初に確認する。
//
// その後、candidate revision外のGo workspaceやGOFLAGSがcompile / test入力へ介入しないことを確認し、
// ambient環境が安全である場合だけprocess environmentをGOWORK=off / GOFLAGS= / GOENV=offへ固定する。
// これにより各laneと通常のnested Go commandも同じ境界を継承する。
//
// GOWORKの実pathやGOFLAGS本文、GOENVの実pathはlocal情報を含み得るためEvidenceへ保存しない。
// 成功・失敗の判定に必要な非機密booleanだけを返す。
func inspectGoEnvironment(output commandOutputFunc) (goEnvironmentEvidence, error) {
	return inspectGoEnvironmentForLaunch(output, os.Setenv, os.LookupEnv)
}

// inspectGoEnvironmentWithは既存pure test用のseamである。
// launch guardを満たした状態を前提にし、ambient Go command観測とchild environment固定を検証する。
func inspectGoEnvironmentWith(output commandOutputFunc, setEnv setEnvFunc) (goEnvironmentEvidence, error) {
	return inspectGoEnvironmentForLaunch(output, setEnv, func(string) (string, bool) {
		return "off", true
	})
}

func inspectGoEnvironmentForLaunch(
	output commandOutputFunc,
	setEnv setEnvFunc,
	lookupEnv lookupEnvFunc,
) (goEnvironmentEvidence, error) {
	var result goEnvironmentEvidence

	goEnv, ok := lookupEnv("GOENV")
	result.GoEnvDisabled = ok && strings.EqualFold(strings.TrimSpace(goEnv), "off")
	if !result.GoEnvDisabled {
		return result, fmt.Errorf("Gate B最終Evidenceはfinal runner自身のcompile前からuser-level Go environmentを無効化するため、GOENV=offを明示して起動してください")
	}

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

	for _, variable := range []struct {
		name  string
		value string
	}{
		{name: "GOWORK", value: "off"},
		{name: "GOFLAGS", value: ""},
		{name: "GOENV", value: "off"},
	} {
		if err := setEnv(variable.name, variable.value); err != nil {
			return result, fmt.Errorf("child Go環境を固定できません: %s: %w", variable.name, err)
		}
	}

	return result, nil
}
