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
//
// The command runs with the temporary directory as its working directory and
// the inputs are copied there, so every path it is given is a bare file name.
// The bash a Windows host provides is WSL bash: it addresses files as
// /mnt/d/... and cannot open a C:\... path, and it replaces $NAME in the -c
// string from the Windows environment before bash parses it (#3461).
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
	// The inputs are copied next to the output so every path the command is
	// given is a bare file name: WSL bash, the bash a Windows host provides,
	// cannot open a C:\... path (#3461).
	inputs := []struct{ source, staged string }{
		{opts.ContractPath, filepath.Join(tempDir, "contract.txt")},
		{opts.PromptPath, filepath.Join(tempDir, "prompt.txt")},
		{opts.ContextPath, filepath.Join(tempDir, "context.json")},
	}
	for _, input := range inputs {
		raw, err := os.ReadFile(input.source)
		if err != nil {
			return CandidateManifest{}, err
		}
		if err := os.WriteFile(input.staged, raw, 0o600); err != nil {
			return CandidateManifest{}, err
		}
	}
	runCtx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}
	values := map[string]string{
		"TEI_CONTRACT": filepath.Base(inputs[0].staged),
		"TEI_LANGUAGE": language,
		"TEI_PROMPT":   filepath.Base(inputs[1].staged),
		"TEI_CONTEXT":  filepath.Base(inputs[2].staged),
		"TEI_SOURCE":   filepath.Base(sourcePath),
		"TEI_TMPDIR":   ".",
		"TEI_PROVIDER": opts.Provider,
		"TEI_MODEL":    opts.Model,
	}
	cmd := exec.CommandContext(runCtx, "bash", "-c", substituteCommandVariables(opts.Command, values))
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(),
		"TEI_CONTRACT="+values["TEI_CONTRACT"],
		"TEI_LANGUAGE="+values["TEI_LANGUAGE"],
		"TEI_PROMPT="+values["TEI_PROMPT"],
		"TEI_CONTEXT="+values["TEI_CONTEXT"],
		"TEI_SOURCE="+values["TEI_SOURCE"],
		"TEI_TMPDIR="+values["TEI_TMPDIR"],
		"TEI_PROVIDER="+values["TEI_PROVIDER"],
		"TEI_MODEL="+values["TEI_MODEL"],
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

// substituteCommandVariables replaces $NAME and ${NAME} with the value the
// command would otherwise read from its environment. The bash a Windows host
// provides is WSL bash, which replaces $NAME in the -c string from the Windows
// environment before bash parses it, so an exported value never reaches the
// command there (#3461). The substituted values are file names this package
// chooses, provider and model names, and ".", so no shell quoting is added and
// a command that already quotes the variable keeps working.
func substituteCommandVariables(command string, values map[string]string) string {
	for name, value := range values {
		command = strings.ReplaceAll(command, "${"+name+"}", value)
		command = strings.ReplaceAll(command, "$"+name, value)
	}
	return command
}
