package targetprofile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	generatorselection "tei/tools/reference-generator-selection"
)

const DeploySchema = "tei.reference.target-profile-deploy/v0"

type DeployOptions struct {
	ProfilePath   string
	CandidatesDir string
	RootDir       string
}

type DeployReceipt struct {
	Schema            string   `json:"schema"`
	ProfileID         string   `json:"profileId"`
	Storage           string   `json:"storage"`
	Update            string   `json:"update"`
	ArtifactDigest    string   `json:"artifactDigest,omitempty"`
	PreviousDigest    string   `json:"previousDigest,omitempty"`
	HistoryPath       string   `json:"historyPath,omitempty"`
	CurrentSlot       string   `json:"currentSlot,omitempty"`
	PreviousSlot      string   `json:"previousSlot,omitempty"`
	RollbackAvailable bool     `json:"rollbackAvailable"`
	Passed            bool     `json:"passed"`
	Reason            string   `json:"reason,omitempty"`
	GrantsAuthority   bool     `json:"grantsAuthority"`
	ProvesRealTarget  bool     `json:"provesRealTarget"`
	Notes             []string `json:"notes"`
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
	if profile.Update != UpdateReplace && profile.Update != UpdateInPlace && profile.Update != UpdateRolling {
		receipt.Reason = "deploy reference supports update=replace, in-place, or rolling, got " + profile.Update
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
	if profile.Update == UpdateRolling {
		if err := deployRolling(opts.RootDir, artifact, &receipt); err != nil {
			receipt.Reason = err.Error()
			return receipt, nil
		}
		receipt.Passed = true
		return receipt, nil
	}
	targetPath := filepath.Join(opts.RootDir, "artifact")
	if previous, err := os.ReadFile(targetPath); err == nil {
		receipt.PreviousDigest = digestBytes(previous)
		historyDir := filepath.Join(opts.RootDir, "history")
		if err := os.MkdirAll(historyDir, 0o755); err != nil {
			receipt.Reason = err.Error()
			return receipt, nil
		}
		historyPath := filepath.Join(historyDir, strings.TrimPrefix(digestBytes(previous), "sha256:"))
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

const RollbackSchema = "tei.reference.target-profile-rollback/v0"

type RollbackReceipt struct {
	Schema           string `json:"schema"`
	CurrentSlot      string `json:"currentSlot,omitempty"`
	PreviousSlot     string `json:"previousSlot,omitempty"`
	ArtifactDigest   string `json:"artifactDigest,omitempty"`
	Passed           bool   `json:"passed"`
	Reason           string `json:"reason,omitempty"`
	GrantsAuthority  bool   `json:"grantsAuthority"`
	ProvesRealTarget bool   `json:"provesRealTarget"`
}

func deployRolling(root string, artifact []byte, receipt *DeployReceipt) error {
	slotsDir := filepath.Join(root, "slots")
	if err := os.MkdirAll(slotsDir, 0o755); err != nil {
		return err
	}
	current, err := readPointer(filepath.Join(root, "current"))
	if err != nil {
		return err
	}
	target, previous := "a", ""
	if current != "" {
		if current != "a" && current != "b" {
			return fmt.Errorf("current slot %q is not a or b", current)
		}
		previous = current
		if current == "a" {
			target = "b"
		} else {
			target = "a"
		}
	}
	slotDir := filepath.Join(slotsDir, target)
	if err := os.MkdirAll(slotDir, 0o755); err != nil {
		return err
	}
	tempArtifact := filepath.Join(slotDir, ".artifact-next")
	if err := os.WriteFile(tempArtifact, artifact, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tempArtifact, filepath.Join(slotDir, "artifact")); err != nil {
		_ = os.Remove(tempArtifact)
		return fmt.Errorf("writing slot artifact: %w", err)
	}
	if previous != "" {
		if err := writePointer(filepath.Join(root, "previous"), previous); err != nil {
			return err
		}
	}
	if err := writePointer(filepath.Join(root, "current"), target); err != nil {
		return err
	}
	installed, err := os.ReadFile(filepath.Join(slotDir, "artifact"))
	if err != nil {
		return fmt.Errorf("reading installed slot artifact: %w", err)
	}
	if digestBytes(installed) != receipt.ArtifactDigest {
		return fmt.Errorf("installed slot digest %s does not match built digest %s",
			digestBytes(installed), receipt.ArtifactDigest)
	}
	pointer, err := readPointer(filepath.Join(root, "current"))
	if err != nil {
		return err
	}
	if pointer != target {
		return fmt.Errorf("current pointer is %q, want %q", pointer, target)
	}
	receipt.CurrentSlot = target
	receipt.PreviousSlot = previous
	receipt.RollbackAvailable = previous != ""
	return nil
}

// Rollback swaps the current and previous slots of a rolling deployment. It
// verifies the artifacts are readable, but does not check health or drain
// traffic like a real rolling update.
func Rollback(root string) (RollbackReceipt, error) {
	receipt := RollbackReceipt{
		Schema: RollbackSchema, GrantsAuthority: false, ProvesRealTarget: false,
	}
	current, err := readPointer(filepath.Join(root, "current"))
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	previous, err := readPointer(filepath.Join(root, "previous"))
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	if current == "" || previous == "" {
		receipt.Reason = "current and previous slots are required"
		return receipt, nil
	}
	if current == previous {
		receipt.Reason = "current and previous slots are the same"
		return receipt, nil
	}
	for _, slot := range []string{current, previous} {
		if slot != "a" && slot != "b" {
			receipt.Reason = "invalid slot " + slot
			return receipt, nil
		}
		if _, err := os.Stat(filepath.Join(root, "slots", slot, "artifact")); err != nil {
			receipt.Reason = "slot " + slot + " artifact is unavailable: " + err.Error()
			return receipt, nil
		}
	}
	if err := writePointer(filepath.Join(root, "current"), previous); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	if err := writePointer(filepath.Join(root, "previous"), current); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	artifact, err := os.ReadFile(filepath.Join(root, "slots", previous, "artifact"))
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	receipt.CurrentSlot = previous
	receipt.PreviousSlot = current
	receipt.ArtifactDigest = digestBytes(artifact)
	receipt.Passed = true
	return receipt, nil
}

func readPointer(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

func writePointer(path, value string) error {
	temp := path + ".next"
	if err := os.WriteFile(temp, []byte(value+"\n"), 0o644); err != nil {
		return err
	}
	if err := os.Rename(temp, path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}
