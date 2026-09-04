package main

import (
	"flag"
	"fmt"
	"os"

	"tei/internal/database/externalqueryevidence"
	"tei/internal/realization"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func main() {
	workspacePath := flag.String("workspace", "reference/workspace", "検証対象Workspace directory")
	evidencePath := flag.String("evidence", "", "external Contract Query Evidence JSON path（必須）")
	flag.Parse()

	if *evidencePath == "" {
		fmt.Fprintln(os.Stderr, "-evidenceでexternal Contract Query Evidence JSONを明示してください")
		os.Exit(2)
	}
	if err := run(*workspacePath, *evidencePath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(workspacePath, evidencePath string) error {
	raw, err := os.ReadFile(evidencePath)
	if err != nil {
		return fmt.Errorf("external Contract Query Evidence %sを読み込めません: %w", evidencePath, err)
	}
	evidence, err := externalqueryevidence.Load(raw)
	if err != nil {
		return err
	}

	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		return fmt.Errorf("Workspace %sを読み込めません: %w", workspacePath, err)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity != validation.Error {
			continue
		}
		return fmt.Errorf(
			"Workspace %sにValidation Errorがあるためexternal Evidenceを受け入れません: %s: %s",
			workspacePath,
			diagnostic.Code,
			diagnostic.Message,
		)
	}

	realizationDef, err := findRealization(loaded.World.ContractQuerySQLRealizations, evidence)
	if err != nil {
		return err
	}
	if err := externalqueryevidence.ValidateAgainst(loaded.World, realizationDef, evidence); err != nil {
		return err
	}

	fmt.Printf("external Contract Query Evidenceを受け入れました: %s / provider=%s\n", evidence.Target.Identity(), evidence.Provider)
	return nil
}

func findRealization(definitions []realization.ContractQuerySQLRealization, evidence externalqueryevidence.Evidence) (realization.ContractQuerySQLRealization, error) {
	var found realization.ContractQuerySQLRealization
	count := 0
	for _, candidate := range definitions {
		if candidate.Target != evidence.Target || candidate.Provider != evidence.Provider {
			continue
		}
		found = candidate
		count++
	}
	if count == 0 {
		return realization.ContractQuerySQLRealization{}, fmt.Errorf(
			"external Contract Query Evidence %s / provider=%sに対応する明示SQL realizationがWorkspaceにありません。Evidenceからrealizationを生成・補完しません",
			evidence.Target.Identity(), evidence.Provider,
		)
	}
	if count > 1 {
		return realization.ContractQuerySQLRealization{}, fmt.Errorf(
			"external Contract Query Evidence %s / provider=%sに対応するSQL realizationが複数あります: %d件",
			evidence.Target.Identity(), evidence.Provider, count,
		)
	}
	return found, nil
}
