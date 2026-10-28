package generatorselection

import (
	"embed"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"tei/internal/ir"
)

//go:embed templates/*.tmpl
var templateFiles embed.FS

const GeneratorSchema = "tei.reference.generator-build/v0"

type GeneratorProvenance struct {
	ID             string `json:"id"`
	Version        string `json:"version"`
	Strategy       string `json:"strategy"`
	Language       string `json:"language,omitempty"`
	ContractDigest string `json:"contractDigest"`
	TemplateDigest string `json:"templateDigest"`
}

type Generated struct {
	Path         string
	SourceDigest string
	Manifest     CandidateManifest
}

type generatorTemplate struct {
	file     string
	id       string
	origin   string
	strategy string
	language string
	source   string
}

var generatorTemplates = map[string]generatorTemplate{
	"template":                         {file: "templates/template.go.tmpl", id: "line-amount-template", origin: "template", strategy: "template", language: LanguageGo, source: "main.go"},
	"alternative":                      {file: "templates/alternative.go.tmpl", id: "line-amount-alternative", origin: "handwritten", strategy: "alternative", language: LanguageGo, source: "main.go"},
	"dangerous":                        {file: "templates/dangerous.go.tmpl", id: "line-amount-dangerous", origin: "generated-ai", strategy: "dangerous", language: LanguageGo, source: "main.go"},
	"python-template":                  {file: "templates/template.py.tmpl", id: "line-amount-python-template", origin: "template", strategy: "python-template", language: LanguagePython, source: "main.py"},
	"javascript-template":              {file: "templates/template.js.tmpl", id: "line-amount-javascript-template", origin: "template", strategy: "javascript-template", language: LanguageJavaScript, source: "main.js"},
	"long-running-template":            {file: "templates/template-long-running.go.tmpl", id: "line-amount-long-running-template", origin: "template", strategy: "long-running-template", language: LanguageGo, source: "main.go"},
	"java-template":                    {file: "templates/Main.java.tmpl", id: "line-amount-java-template", origin: "template", strategy: "java-template", language: LanguageJava, source: "Main.java"},
	"python-long-running-template":     {file: "templates/template-long-running.py.tmpl", id: "line-amount-python-long-running-template", origin: "template", strategy: "python-long-running-template", language: LanguagePython, source: "main.py"},
	"javascript-long-running-template": {file: "templates/template-long-running.js.tmpl", id: "line-amount-javascript-long-running-template", origin: "template", strategy: "javascript-long-running-template", language: LanguageJavaScript, source: "main.js"},
	"wasm-template":                    {file: "templates/template.go.tmpl", id: "line-amount-wasm-template", origin: "template", strategy: "wasm-template", language: LanguageWasm, source: "main.go"},
	"java-long-running-template":       {file: "templates/Main-long-running.java.tmpl", id: "line-amount-java-long-running-template", origin: "template", strategy: "java-long-running-template", language: LanguageJava, source: "Main.java"},
	"wasm-long-running-template":       {file: "templates/template-long-running.go.tmpl", id: "line-amount-wasm-long-running-template", origin: "template", strategy: "wasm-long-running-template", language: LanguageWasm, source: "main.go"},
	"wasi-template":                    {file: "templates/template.go.tmpl", id: "line-amount-wasi-template", origin: "template", strategy: "wasi-template", language: LanguageWasi, source: "main.go"},
	"wasi-long-running-template":       {file: "templates/template-long-running.go.tmpl", id: "line-amount-wasi-long-running-template", origin: "template", strategy: "wasi-long-running-template", language: LanguageWasi, source: "main.go"},
}

type templateData struct {
	ContractID      string
	ContractVersion string
	ContractDigest  string
	GeneratorID     string
	Strategy        string
	Currency        string
	Precision       int
	Scale           int
	IntegerDigits   int
	ScalePow        int
}

// renderCandidate renders one source and its provenance without touching the
// filesystem. Generate writes the result; Select calls renderCandidate again to
// verify that a generated candidate really came from the declared contract and
// strategy.
func renderCandidate(contractPath, strategy string) (renderedCandidate, error) {
	return renderCandidateVersion(contractPath, strategy, "1")
}

// renderCandidateVersion renders a candidate at an explicit generator version.
// The source is unchanged by the version; the manifest and provenance carry it.
func renderCandidateVersion(contractPath, strategy, version string) (renderedCandidate, error) {
	contractRaw, err := os.ReadFile(contractPath)
	if err != nil {
		return renderedCandidate{}, err
	}
	return renderCandidateBytes(contractRaw, strategy, version, nil)
}

// RenderForWorld reuses the fixed generator after resolving the reference
// contract's inputs against the caller's already-loaded World. It neither
// accepts the composite rule nor builds, selects, or executes the source.
func RenderForWorld(world ir.World, contractJSON []byte, strategy, version string) ([]byte, CandidateManifest, error) {
	candidate, err := renderCandidateBytes(contractJSON, strategy, version, &world)
	if err != nil {
		return nil, CandidateManifest{}, err
	}
	return candidate.Source, candidate.Manifest, nil
}

func renderCandidateBytes(contractRaw []byte, strategy, version string, world *ir.World) (renderedCandidate, error) {
	var contract Contract
	if err := json.Unmarshal(contractRaw, &contract); err != nil {
		return renderedCandidate{}, err
	}
	if err := validateContract(contract); err != nil {
		return renderedCandidate{}, err
	}
	if world != nil {
		if err := validateAgainstWorkspace(contract, *world); err != nil {
			return renderedCandidate{}, err
		}
	}
	tmpl, ok := generatorTemplates[strategy]
	if !ok {
		return renderedCandidate{}, fmt.Errorf("unknown generator strategy %q", strategy)
	}
	if strings.TrimSpace(version) == "" {
		return renderedCandidate{}, fmt.Errorf("generator version is required")
	}
	if contract.Rounding.Mode != "half-up" {
		return renderedCandidate{}, fmt.Errorf("fixed generator strategy %q does not support rounding mode %q", strategy, contract.Rounding.Mode)
	}

	templateRaw, err := templateFiles.ReadFile(tmpl.file)
	if err != nil {
		return renderedCandidate{}, err
	}
	parsed, err := template.New(tmpl.strategy).Parse(string(templateRaw))
	if err != nil {
		return renderedCandidate{}, err
	}
	integerDigits := contract.Rounding.Precision - contract.Rounding.Scale
	scalePow := 1
	for i := 0; i < contract.Rounding.Scale; i++ {
		scalePow *= 10
	}
	data := templateData{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ContractDigest: digest(contractRaw), GeneratorID: tmpl.id, Strategy: tmpl.strategy,
		Currency: "JPY", Precision: contract.Rounding.Precision, Scale: contract.Rounding.Scale,
		IntegerDigits: integerDigits, ScalePow: scalePow,
	}
	var rendered strings.Builder
	if err := parsed.Execute(&rendered, data); err != nil {
		return renderedCandidate{}, err
	}
	formatted := []byte(rendered.String())
	if tmpl.language == LanguageGo || tmpl.language == LanguageWasm || tmpl.language == LanguageWasi {
		formatted, err = format.Source(formatted)
		if err != nil {
			return renderedCandidate{}, fmt.Errorf("generated source is not valid Go: %w", err)
		}
	}
	manifest := CandidateManifest{
		Schema: CandidateSchema, ID: tmpl.strategy, Version: version, Origin: tmpl.origin,
		Language: tmpl.language, Source: tmpl.source, SourceDigest: digest(formatted),
		Generator: &GeneratorProvenance{
			ID: tmpl.id, Version: version, Strategy: tmpl.strategy, Language: tmpl.language,
			ContractDigest: data.ContractDigest, TemplateDigest: digest(templateRaw),
		},
	}
	return renderedCandidate{Source: formatted, Manifest: manifest}, nil
}

type renderedCandidate struct {
	Source   []byte
	Manifest CandidateManifest
}

// Generate renders one candidate source at version 1 from the contract and a
// fixed strategy.
func Generate(contractPath, strategy, outDir string) (Generated, error) {
	return GenerateVersion(contractPath, strategy, "1", outDir)
}

// GenerateVersion renders one candidate source from the contract and a fixed
// strategy. The generated source is gofmt'ed before it is written, and the
// candidate manifest records the contract, template, and generator version.
// The version is a declaration for lifecycle tracking; it does not grant
// acceptance or execution authority.
func GenerateVersion(contractPath, strategy, version, outDir string) (Generated, error) {
	rendered, err := renderCandidateVersion(contractPath, strategy, version)
	if err != nil {
		return Generated{}, err
	}
	if _, err := os.Lstat(outDir); err == nil {
		return Generated{}, fmt.Errorf("output directory already exists: %s", outDir)
	} else if !os.IsNotExist(err) {
		return Generated{}, err
	}
	if err := os.MkdirAll(filepath.Dir(outDir), 0o755); err != nil {
		return Generated{}, err
	}
	if err := os.Mkdir(outDir, 0o755); err != nil {
		return Generated{}, err
	}
	cleanup := func() { _ = os.RemoveAll(outDir) }
	sourcePath := filepath.Join(outDir, rendered.Manifest.Source)
	if err := os.WriteFile(sourcePath, rendered.Source, 0o644); err != nil {
		cleanup()
		return Generated{}, err
	}
	manifestRaw, err := json.MarshalIndent(rendered.Manifest, "", "  ")
	if err != nil {
		cleanup()
		return Generated{}, err
	}
	manifestRaw = append(manifestRaw, '\n')
	if err := os.WriteFile(filepath.Join(outDir, "candidate.json"), manifestRaw, 0o644); err != nil {
		cleanup()
		return Generated{}, err
	}
	return Generated{Path: outDir, SourceDigest: rendered.Manifest.SourceDigest, Manifest: rendered.Manifest}, nil
}

// GenerateAll renders the three fixed strategies under outRoot in a stable
// order.
func GenerateAll(contractPath, outRoot string) ([]Generated, error) {
	var results []Generated
	for _, strategy := range []string{"template", "alternative", "dangerous"} {
		generated, err := Generate(contractPath, strategy, filepath.Join(outRoot, strategy))
		if err != nil {
			return nil, err
		}
		results = append(results, generated)
	}
	return results, nil
}

// ContractDigest returns the digest a generated manifest records.
func ContractDigest(contractPath string) (string, error) {
	raw, err := os.ReadFile(contractPath)
	if err != nil {
		return "", err
	}
	return digest(raw), nil
}
