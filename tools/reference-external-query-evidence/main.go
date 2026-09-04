package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/database/externalqueryevidence"
	"tei/internal/realization"
	"tei/internal/validation"
	"tei/internal/workspace"
	"tei/tools/internal/repositoryevidence"
)

const acceptanceEvidenceNature = "external-query-evidence-acceptance"

type acceptanceReceipt struct {
	EvidenceNature              string                     `json:"evidenceNature"`
	RepositoryRevision          string                     `json:"repositoryRevision"`
	WorkingTreeClean            bool                       `json:"workingTreeClean"`
	ReproducibleRevision        bool                       `json:"reproducibleRevision"`
	PostRunRepository           repositoryevidence.PostRun `json:"postRunRepository"`
	EvidenceArtifactFingerprint string                     `json:"evidenceArtifactFingerprint"`
	Target                      string                     `json:"target"`
	Provider                    string                     `json:"provider"`
	Accepted                    bool                       `json:"accepted"`
}

func main() {
	workspacePath := flag.String("workspace", "reference/workspace", "検証対象Workspace directory")
	evidencePath := flag.String("evidence", "", "external Contract Query Evidence JSON path（必須）")
	flag.Parse()

	if *evidencePath == "" {
		fmt.Fprintln(os.Stderr, "-evidenceでexternal Contract Query Evidence JSONを明示してください")
		os.Exit(2)
	}
	receipt, err := run(*workspacePath, *evidencePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
}

func run(workspacePath, evidencePath string) (acceptanceReceipt, error) {
	result := acceptanceReceipt{EvidenceNature: acceptanceEvidenceNature}

	repositorySource, err := repositoryevidence.InspectStart()
	if err != nil {
		return result, err
	}
	result.RepositoryRevision = repositorySource.RepositoryRevision
	result.WorkingTreeClean = repositorySource.WorkingTreeClean
	if !repositorySource.WorkingTreeClean {
		return result, fmt.Errorf("external EvidenceのGate B受入れはclean revisionに固定するため、working treeに未commit差分がある状態では受理しません")
	}

	raw, err := os.ReadFile(evidencePath)
	if err != nil {
		return result, fmt.Errorf("external Contract Query Evidence %sを読み込めません: %w", evidencePath, err)
	}
	result.EvidenceArtifactFingerprint = evidenceArtifactFingerprint(raw)

	evidence, err := externalqueryevidence.Load(raw)
	if err != nil {
		return result, err
	}
	result.Target = evidence.Target.Identity()
	result.Provider = string(evidence.Provider)

	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		return result, fmt.Errorf("Workspace %sを読み込めません: %w", workspacePath, err)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity != validation.Error {
			continue
		}
		return result, fmt.Errorf(
			"Workspace %sにValidation Errorがあるためexternal Evidenceを受け入れません: %s: %s",
			workspacePath,
			diagnostic.Code,
			diagnostic.Message,
		)
	}
	realizationDef, err := findRealization(loaded.World.ContractQuerySQLRealizations, evidence)
	if err != nil {
		return result, err
	}
	if err := externalqueryevidence.ValidateAgainst(loaded.World, realizationDef, evidence); err != nil {
		return result, err
	}

	postRun, err := repositoryevidence.InspectPost(repositorySource.RepositoryRevision)
	if err != nil {
		return result, err
	}
	result.PostRunRepository = postRun
	result.ReproducibleRevision = repositoryevidence.IsReproducible(repositorySource, postRun)
	if !postRun.RevisionStable {
		return result, fmt.Errorf("external Evidence受入れ中にrepository HEADが変化したため、単一revisionのGate B Evidenceとして採用できません")
	}
	if !postRun.WorkingTreeClean {
		return result, fmt.Errorf("external Evidence受入れ後にworking tree差分が残ったため、記録revisionと検証sourceの同一性を保証できません")
	}
	if !result.ReproducibleRevision {
		return result, fmt.Errorf("external Evidence受入れを単一のclean revisionへ固定できません")
	}

	result.Accepted = true
	return result, nil
}

func evidenceArtifactFingerprint(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
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
