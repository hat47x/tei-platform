package targetprofile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	generatorselection "tei/tools/reference-generator-selection"
)

const DeploySchema = "tei.reference.target-profile-deploy/v0"

type DeployOptions struct {
	ProfilePath   string
	CandidatesDir string
	RootDir       string
}

type DeployReceipt struct {
	Schema           string   `json:"schema"`
	ProfileID        string   `json:"profileId"`
	Storage          string   `json:"storage"`
	Update           string   `json:"update"`
	ArtifactDigest   string   `json:"artifactDigest,omitempty"`
	PreviousDigest   string   `json:"previousDigest,omitempty"`
	HistoryPath      string   `json:"historyPath,omitempty"`
	Passed           bool     `json:"passed"`
	Reason           string   `json:"reason,omitempty"`
	GrantsAuthority  bool     `json:"grantsAuthority"`
	ProvesRealTarget bool     `json:"provesRealTarget"`
	Notes            []string `json:"notes"`
}

// Deploy builds a Go candidate and applies it to a local filesystem root using
// the profile's update mode. It is a filesystem-level reference, not a real
// target deployment, supervisor, or rollback system.
func Deploy(ctx context.Context, opts DeployOptions) (DeployReceipt, error) {
	profile, err := Load(opts.ProfilePath)
	if err != nil {
		return DeployReceipt{}, err
	}
	receipt := DeployReceipt{
		Schema: DeploySchema, ProfileID: profile.ID, Storage: profile.Storage, Update: profile.Update,
		GrantsAuthority: false, ProvesRealTarget: false,
		Notes: []string{
			"ローカルfilesystem上で候補をbuildし、profileのupdate方式でartifactを置き換える参照。",
			"実Targetの配備・supervisor・ロールバック・無停止更新を証明しない。",
		},
	}
	if profile.Storage != StorageFilesystem {
		receipt.Reason = "deploy reference supports only storage=filesystem, got " + profile.Storage
		return receipt, nil
	}
	if profile.Update != UpdateReplace && profile.Update != UpdateInPlace {
		receipt.Reason = "deploy reference supports update=replace or in-place, got " + profile.Update
		return receipt, nil
	}
	if profile.GeneratorID == "" {
		receipt.Reason = "profile has no generatorId"
		return receipt, nil
	}
	candidateDir := filepath.Join(opts.CandidatesDir, profile.GeneratorID)
	var manifest generatorselection.CandidateManifest
	raw, err := os.ReadFile(filepath.Join(candidateDir, "candidate.json"))
	if err != nil {
		receipt.Reason = "candidate manifest is unreadable: " + err.Error()
		return receipt, nil
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		receipt.Reason = "candidate manifest is invalid: " + err.Error()
		return receipt, nil
	}
	language := manifest.Language
	if language == "" {
		language = generatorselection.LanguageGo
	}
	if language != generatorselection.LanguageGo {
		receipt.Reason = "deploy reference supports only Go candidates"
		return receipt, nil
	}
	if err := os.MkdirAll(opts.RootDir, 0o755); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	stagingDir, err := os.MkdirTemp(opts.RootDir, ".staging-*")
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	defer os.RemoveAll(stagingDir)
	binaryPath := filepath.Join(stagingDir, "artifact")
	if err := buildGoCandidate(ctx, candidateDir, binaryPath); err != nil {
		receipt.Reason = "candidate build failed: " + err.Error()
		return receipt, nil
	}
	artifact, err := os.ReadFile(binaryPath)
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	receipt.ArtifactDigest = digestBytes(artifact)
	targetPath := filepath.Join(opts.RootDir, "artifact")
	if previous, err := os.ReadFile(targetPath); err == nil {
		receipt.PreviousDigest = digestBytes(previous)
		historyDir := filepath.Join(opts.RootDir, "history")
		if err := os.MkdirAll(historyDir, 0o755); err != nil {
			receipt.Reason = err.Error()
			return receipt, nil
		}
		historyPath := filepath.Join(historyDir, digestBytes(previous))
		if err := os.WriteFile(historyPath, previous, 0o644); err != nil {
			receipt.Reason = err.Error()
			return receipt, nil
		}
		receipt.HistoryPath = historyPath
	} else if !os.IsNotExist(err) {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	switch profile.Update {
	case UpdateReplace:
		tempPath := filepath.Join(opts.RootDir, ".artifact-next")
		if err := os.WriteFile(tempPath, artifact, 0o755); err != nil {
			receipt.Reason = err.Error()
			return receipt, nil
		}
		if err := os.Rename(tempPath, targetPath); err != nil {
			_ = os.Remove(tempPath)
			receipt.Reason = "atomic replace failed: " + err.Error()
			return receipt, nil
		}
	case UpdateInPlace:
		if err := os.WriteFile(targetPath, artifact, 0o755); err != nil {
			receipt.Reason = err.Error()
			return receipt, nil
		}
	}
	installed, err := os.ReadFile(targetPath)
	if err != nil {
		receipt.Reason = "installed artifact is unreadable: " + err.Error()
		return receipt, nil
	}
	if digestBytes(installed) != receipt.ArtifactDigest {
		receipt.Reason = fmt.Sprintf("installed digest %s does not match built digest %s",
			digestBytes(installed), receipt.ArtifactDigest)
		return receipt, nil
	}
	if receipt.HistoryPath != "" && receipt.PreviousDigest != "" {
		history, err := os.ReadFile(receipt.HistoryPath)
		if err != nil {
			receipt.Reason = "history artifact is unreadable: " + err.Error()
			return receipt, nil
		}
		if digestBytes(history) != receipt.PreviousDigest {
			receipt.Reason = "history artifact digest does not match the replaced artifact"
			return receipt, nil
		}
	}
	receipt.Passed = true
	return receipt, nil
}
