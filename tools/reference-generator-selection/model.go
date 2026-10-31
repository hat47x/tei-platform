package generatorselection

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ModelOptions describes one trusted external generation command. TEI does not
// embed a vendor API or model credential: the command is supplied by the
// caller, and its output is recorded through the same provenance boundary as a
// hand-recorded candidate.
type ModelOptions struct {
	ID           string
	Provider     string
	Model        string
	Language     string
	ContractPath string
	PromptPath   string
	ContextPath  string
	OutDir       string
	Command      string
	Timeout      time.Duration
}

// GenerateWithCommand runs one external command and records the source it
// writes. The command receives TEI_PROMPT, TEI_CONTEXT, TEI_SOURCE, TEI_TMPDIR,
// TEI_PROVIDER, and TEI_MODEL in its environment and must write Go source to
// TEI_SOURCE. The command is trusted; this is not a sandbox.
func GenerateWithCommand(ctx context.Context, opts ModelOptions) (CandidateManifest, error) {
	if strings.TrimSpace(opts.ID) == "" || strings.TrimSpace(opts.Provider) == "" || strings.TrimSpace(opts.Model) == "" {
		return CandidateManifest{}, fmt.Errorf("id, provider, and model are required")
	}
	if strings.TrimSpace(opts.ContractPath) == "" || strings.TrimSpace(opts.PromptPath) == "" || strings.TrimSpace(opts.ContextPath) == "" {
		return CandidateManifest{}, fmt.Errorf("contract, prompt, and context are required")
	}
	if strings.TrimSpace(opts.Command) == "" {
		return CandidateManifest{}, fmt.Errorf("command is required")
	}
	language, err := normalizeLanguage(opts.Language)
	if err != nil {
		return CandidateManifest{}, err
	}
	tempDir, err := os.MkdirTemp("", "tei-generator-model-*")
	if err != nil {
		return CandidateManifest{}, err
	}
	defer os.RemoveAll(tempDir)
	sourcePath := filepath.Join(tempDir, "source"+filepath.Ext(candidateSourceName(language)))
	runCtx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(runCtx, "bash", "-c", opts.Command)
	cmd.Env = append(os.Environ(),
		"TEI_CONTRACT="+opts.ContractPath,
		"TEI_LANGUAGE="+language,
		"TEI_PROMPT="+opts.PromptPath,
		"TEI_CONTEXT="+opts.ContextPath,
		"TEI_SOURCE="+sourcePath,
		"TEI_TMPDIR="+tempDir,
		"TEI_PROVIDER="+opts.Provider,
		"TEI_MODEL="+opts.Model,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return CandidateManifest{}, fmt.Errorf("generation command failed: %w: %s", err, output)
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return CandidateManifest{}, fmt.Errorf("generation command did not write TEI_SOURCE: %w", err)
	}
	if info.Size() == 0 {
		return CandidateManifest{}, fmt.Errorf("generation command wrote an empty source")
	}
	return Record(RecordOptions{
		ID: opts.ID, Provider: opts.Provider, Model: opts.Model, Language: language,
		ContractPath: opts.ContractPath,
		SourcePath:   sourcePath, PromptPath: opts.PromptPath, ContextPath: opts.ContextPath,
		Note: "recorded from an external generation command",
	}, opts.OutDir)
}
