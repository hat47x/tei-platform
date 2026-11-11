package artifact

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// EvidenceSchemaVersion is the version of the evidence bundle schema.
const EvidenceSchemaVersion = "tei.artifact.evidence/v0"

// CLIBuildIdentity describes the executable that produced the artifacts.
type CLIBuildIdentity struct {
	SchemaVersion    string   `json:"schemaVersion"`
	GoVersion        string   `json:"goVersion"`
	GOOS             string   `json:"goos"`
	GOARCH           string   `json:"goarch"`
	VCSRevision      string   `json:"vcsRevision,omitempty"`
	VCSModified      bool     `json:"vcsModified,omitempty"`
	BuildSettings    []string `json:"buildSettings,omitempty"`
	DeclaredRevision string   `json:"declaredRevision,omitempty"`
	Source           string   `json:"source"`
}

// EvidenceBundle records the inputs, toolchain, generation, and verification
// for one derived artifact set. It is written beside the artifact files so a
// reader can trace every output back to the source and tool state that created
// it.
type EvidenceBundle struct {
	SchemaVersion string `json:"schemaVersion"`
	GeneratedAt   string `json:"generatedAt"`

	Workspace              string `json:"workspace"`
	ApplicationID          string `json:"applicationId"`
	WorldFingerprint       string `json:"worldFingerprint"`
	SourceIndexFingerprint string `json:"sourceIndexFingerprint"`
	ModelRevision          string `json:"modelRevision"`

	Toolchain ToolchainEvidence `json:"toolchain"`
	Render    RenderEvidence    `json:"render"`
	Verify    VerifyEvidence    `json:"verify"`
	Output    *OutputEvidence   `json:"output,omitempty"`
}

// ToolchainEvidence records the generator and CLI identity that produced the set.
type ToolchainEvidence struct {
	Generator string            `json:"generator"`
	CLI       *CLIBuildIdentity `json:"cli,omitempty"`
}

// RenderEvidence records what the render phase produced.
type RenderEvidence struct {
	ArtifactCount  int              `json:"artifactCount"`
	Complete       bool             `json:"complete"`
	ManifestDigest string           `json:"manifestDigest"`
	Artifacts      []ArtifactDigest `json:"artifacts"`
}

// ArtifactDigest pairs an artifact kind with its digest.
type ArtifactDigest struct {
	Kind   string `json:"kind"`
	Digest string `json:"digest"`
}

// VerifyEvidence records the consistency check result.
type VerifyEvidence struct {
	FindingsCount int      `json:"findingsCount"`
	FindingCodes  []string `json:"findingCodes,omitempty"`
	Verified      bool     `json:"verified"`
}

// OutputEvidence records where the files were written.
type OutputEvidence struct {
	Directory      string       `json:"directory"`
	Files          []OutputFile `json:"files"`
	EvidenceFile   string       `json:"evidenceFile,omitempty"`
	EvidenceDigest string       `json:"evidenceDigest,omitempty"`
}

// OutputFile is one written artifact file.
type OutputFile struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

// NewEvidenceBundle builds an evidence bundle bound to the exact source and
// artifact state passed in. The outDir may be empty when the bundle is built
// before files are written.
func NewEvidenceBundle(root, applicationID string, loaded workspace.Result, index sourceindex.Index, set WorkspaceSet, cliIdentity *CLIBuildIdentity, generator string, outDir string) (EvidenceBundle, error) {
	worldFP, sourceFP, err := operations.WorkspaceProjectionFingerprints(loaded.World, index)
	if err != nil {
		return EvidenceBundle{}, err
	}

	modelRevision := sourceindex.ModelRevision(index)

	findingCodes := make([]string, 0, len(set.Findings))
	for _, f := range set.Findings {
		findingCodes = append(findingCodes, f.Code)
	}
	sort.Strings(findingCodes)

	verified := true
	for _, a := range set.Artifacts {
		if err := a.Verify(); err != nil {
			return EvidenceBundle{}, fmt.Errorf("artifact %s does not verify: %w", a.Kind, err)
		}
	}
	if set.Manifest.Digest != "" {
		if err := set.Manifest.Verify(); err != nil {
			return EvidenceBundle{}, fmt.Errorf("manifest does not verify: %w", err)
		}
	}

	artifactDigests := make([]ArtifactDigest, 0, len(set.Artifacts))
	for _, a := range set.Artifacts {
		artifactDigests = append(artifactDigests, ArtifactDigest{Kind: a.Kind, Digest: a.Digest})
	}
	sort.Slice(artifactDigests, func(i, j int) bool {
		if artifactDigests[i].Kind != artifactDigests[j].Kind {
			return artifactDigests[i].Kind < artifactDigests[j].Kind
		}
		return artifactDigests[i].Digest < artifactDigests[j].Digest
	})

	bundle := EvidenceBundle{
		SchemaVersion:          EvidenceSchemaVersion,
		GeneratedAt:            time.Now().UTC().Format(time.RFC3339),
		Workspace:              root,
		ApplicationID:          applicationID,
		WorldFingerprint:       worldFP,
		SourceIndexFingerprint: sourceFP,
		ModelRevision:          modelRevision,
		Toolchain: ToolchainEvidence{
			Generator: generator,
			CLI:       cliIdentity,
		},
		Render: RenderEvidence{
			ArtifactCount:  len(set.Artifacts),
			Complete:       set.Manifest.Complete,
			ManifestDigest: set.Manifest.Digest,
			Artifacts:      artifactDigests,
		},
		Verify: VerifyEvidence{
			FindingsCount: len(set.Findings),
			FindingCodes:  findingCodes,
			Verified:      verified && len(set.Findings) == 0,
		},
	}

	if outDir != "" {
		output := OutputEvidence{Directory: outDir}
		bundle.Output = &output
	}

	return bundle, nil
}

// WriteEvidenceBundle writes the bundle as evidence.json in outDir and returns
// the updated bundle with its file digest.
func WriteEvidenceBundle(outDir string, bundle EvidenceBundle) (EvidenceBundle, error) {
	if outDir == "" {
		return bundle, nil
	}

	output := OutputEvidence{Directory: outDir}
	bundle.Output = &output

	raw, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return bundle, err
	}
	raw = append(raw, '\n')

	path := filepath.Join(outDir, "evidence.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return bundle, err
	}

	digest, err := fileDigest(path)
	if err != nil {
		return bundle, err
	}

	bundle.Output.EvidenceFile = "evidence.json"
	bundle.Output.EvidenceDigest = digest
	return bundle, nil
}

// StageArtifacts writes artifact files and manifest to a temporary directory
// inside outDir, verifies the staged set, and returns the temp directory path.
// The caller is responsible for removing the temp directory after publishing.
func StageArtifacts(outDir string, set WorkspaceSet) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}

	tmpDir, err := os.MkdirTemp(outDir, ".tei-artifacts-")
	if err != nil {
		return "", err
	}

	names := map[string]string{}
	for _, item := range set.Artifacts {
		name, body := ArtifactFileName(item)
		if previous, duplicated := names[name]; duplicated && previous != item.Digest {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("two artifacts want the same file name %s; refusing to overwrite", name)
		}
		names[name] = item.Digest
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(body), 0o644); err != nil {
			os.RemoveAll(tmpDir)
			return "", err
		}
	}

	manifest, err := json.MarshalIndent(struct {
		Manifest Manifest  `json:"manifest"`
		Findings []Finding `json:"findings"`
	}{Manifest: set.Manifest, Findings: set.Findings}, "", "  ")
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "manifest.json"), append(manifest, '\n'), 0o644); err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	if err := verifyStagedSet(tmpDir, set); err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	return tmpDir, nil
}

// PublishStage moves every file from the staging directory into outDir. It
// removes stale files that are no longer produced and leaves the staging
// directory empty.
func PublishStage(stageDir, outDir string) error {
	entries, err := os.ReadDir(stageDir)
	if err != nil {
		return err
	}

	seen := map[string]bool{}
	for _, entry := range entries {
		src := filepath.Join(stageDir, entry.Name())
		dst := filepath.Join(outDir, entry.Name())
		if err := os.RemoveAll(dst); err != nil {
			return err
		}
		if err := os.Rename(src, dst); err != nil {
			return err
		}
		seen[entry.Name()] = true
	}

	// Remove files that were in outDir but are no longer produced.
	existing, err := os.ReadDir(outDir)
	if err != nil {
		return err
	}
	for _, entry := range existing {
		if entry.IsDir() || seen[entry.Name()] {
			continue
		}
		if err := os.Remove(filepath.Join(outDir, entry.Name())); err != nil {
			return err
		}
	}

	return nil
}

func verifyStagedSet(dir string, set WorkspaceSet) error {
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return err
	}
	var envelope struct {
		Manifest Manifest  `json:"manifest"`
		Findings []Finding `json:"findings"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("staged manifest is not valid JSON: %w", err)
	}
	if envelope.Manifest.Digest != set.Manifest.Digest {
		return fmt.Errorf("staged manifest digest mismatch: have %s want %s", envelope.Manifest.Digest, set.Manifest.Digest)
	}
	if envelope.Manifest.Digest != "" {
		if err := envelope.Manifest.Verify(); err != nil {
			return fmt.Errorf("staged manifest does not verify: %w", err)
		}
	}

	for _, artifact := range set.Artifacts {
		name, _ := ArtifactFileName(artifact)
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if string(raw) != artifact.Content {
			return fmt.Errorf("staged artifact %s content mismatch", name)
		}
		if err := artifact.Verify(); err != nil {
			return err
		}
	}

	// Verify evidence.json if present.
	evidencePath := filepath.Join(dir, "evidence.json")
	if _, err := os.Stat(evidencePath); err == nil {
		raw, err := os.ReadFile(evidencePath)
		if err != nil {
			return err
		}
		var bundle EvidenceBundle
		if err := json.Unmarshal(raw, &bundle); err != nil {
			return fmt.Errorf("staged evidence is not valid JSON: %w", err)
		}
		if bundle.SchemaVersion != EvidenceSchemaVersion {
			return fmt.Errorf("staged evidence has unexpected schema %q", bundle.SchemaVersion)
		}
		if bundle.Render.ManifestDigest != set.Manifest.Digest {
			return fmt.Errorf("staged evidence manifest digest mismatch")
		}
	}

	return nil
}

// fileDigest is a small helper that matches sourceindex.Fingerprint.
func fileDigest(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return sourceindex.Fingerprint(content), nil
}
