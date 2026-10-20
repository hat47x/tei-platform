package generatorselection

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RecordOptions describes one externally generated candidate. The model call
// happens outside this package; Record only writes the source and the
// provenance needed to reproduce and verify the selection input.
type RecordOptions struct {
	ID           string
	Origin       string
	Provider     string
	Model        string
	Language     string
	ContractPath string
	SourcePath   string
	PromptPath   string
	ContextPath  string
	Note         string
}

// Record writes a candidate directory from an externally generated source. It
// does not call a model, execute the source, or accept it. The generated source
// is copied byte-for-byte; the prompt and context are stored beside it so the
// selection can verify their digests independently.
func Record(opts RecordOptions, outDir string) (CandidateManifest, error) {
	if strings.TrimSpace(opts.ID) == "" {
		return CandidateManifest{}, fmt.Errorf("id is required")
	}
	if strings.TrimSpace(opts.Provider) == "" || strings.TrimSpace(opts.Model) == "" {
		return CandidateManifest{}, fmt.Errorf("provider and model are required")
	}
	if strings.TrimSpace(opts.SourcePath) == "" || strings.TrimSpace(opts.PromptPath) == "" || strings.TrimSpace(opts.ContextPath) == "" {
		return CandidateManifest{}, fmt.Errorf("source, prompt, and context are required")
	}
	if strings.TrimSpace(opts.ContractPath) == "" {
		return CandidateManifest{}, fmt.Errorf("contract is required")
	}
	language, err := normalizeLanguage(opts.Language)
	if err != nil {
		return CandidateManifest{}, err
	}
	expectedExtension := ".go"
	switch language {
	case LanguagePython:
		expectedExtension = ".py"
	case LanguageJavaScript:
		expectedExtension = ".js"
	}
	if strings.ToLower(filepath.Ext(opts.SourcePath)) != expectedExtension {
		return CandidateManifest{}, fmt.Errorf("source extension does not match language %s", language)
	}
	sourceName := candidateSourceName(language)
	var contract Contract
	if _, err := readJSON(opts.ContractPath, &contract); err != nil {
		return CandidateManifest{}, err
	}
	if err := validateContract(contract); err != nil {
		return CandidateManifest{}, err
	}
	contractRaw, err := os.ReadFile(opts.ContractPath)
	if err != nil {
		return CandidateManifest{}, err
	}
	origin := opts.Origin
	if origin == "" {
		origin = "generated-ai"
	}
	source, err := os.ReadFile(opts.SourcePath)
	if err != nil {
		return CandidateManifest{}, err
	}
	prompt, err := os.ReadFile(opts.PromptPath)
	if err != nil {
		return CandidateManifest{}, err
	}
	context, err := os.ReadFile(opts.ContextPath)
	if err != nil {
		return CandidateManifest{}, err
	}
	if _, err := os.Lstat(outDir); err == nil {
		return CandidateManifest{}, fmt.Errorf("output directory already exists: %s", outDir)
	} else if !os.IsNotExist(err) {
		return CandidateManifest{}, err
	}
	if err := os.MkdirAll(filepath.Dir(outDir), 0o755); err != nil {
		return CandidateManifest{}, err
	}
	if err := os.Mkdir(outDir, 0o755); err != nil {
		return CandidateManifest{}, err
	}
	cleanup := func() { _ = os.RemoveAll(outDir) }
	if err := os.WriteFile(filepath.Join(outDir, sourceName), source, 0o644); err != nil {
		cleanup()
		return CandidateManifest{}, err
	}
	if err := os.WriteFile(filepath.Join(outDir, "prompt.txt"), prompt, 0o644); err != nil {
		cleanup()
		return CandidateManifest{}, err
	}
	if err := os.WriteFile(filepath.Join(outDir, "context.json"), context, 0o644); err != nil {
		cleanup()
		return CandidateManifest{}, err
	}
	manifest := CandidateManifest{
		Schema: CandidateSchema, ID: opts.ID, Version: "1", Language: language, Origin: origin,
		Source: sourceName, SourceDigest: digest(source), Note: opts.Note,
		Generation: &GenerationProvenance{
			Provider: opts.Provider, Model: opts.Model, ContractDigest: digest(contractRaw),
			PromptDigest: digest(prompt), ContextDigest: digest(context), OutputDigest: digest(source),
			PromptPath: "prompt.txt", ContextPath: "context.json",
		},
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		cleanup()
		return CandidateManifest{}, err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(outDir, "candidate.json"), raw, 0o644); err != nil {
		cleanup()
		return CandidateManifest{}, err
	}
	return manifest, nil
}

func normalizeLanguage(language string) (string, error) {
	switch strings.TrimSpace(language) {
	case "":
		return LanguageGo, nil
	case LanguageGo:
		return LanguageGo, nil
	case LanguagePython:
		return LanguagePython, nil
	case LanguageJavaScript:
		return LanguageJavaScript, nil
	default:
		return "", fmt.Errorf("unsupported language %q", language)
	}
}

func candidateSourceName(language string) string {
	switch language {
	case LanguagePython:
		return "main.py"
	case LanguageJavaScript:
		return "main.js"
	default:
		return "main.go"
	}
}
