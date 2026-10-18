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
)

//go:embed templates/*.tmpl
var templateFiles embed.FS

const GeneratorSchema = "tei.reference.generator-build/v0"

type GeneratorProvenance struct {
	ID             string `json:"id"`
	Version        string `json:"version"`
	Strategy       string `json:"strategy"`
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
}

var generatorTemplates = map[string]generatorTemplate{
	"template":    {file: "templates/template.go.tmpl", id: "line-amount-template", origin: "template", strategy: "template"},
	"alternative": {file: "templates/alternative.go.tmpl", id: "line-amount-alternative", origin: "handwritten", strategy: "alternative"},
	"dangerous":   {file: "templates/dangerous.go.tmpl", id: "line-amount-dangerous", origin: "generated-ai", strategy: "dangerous"},
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

// Generate renders one candidate source from the contract and a fixed
// strategy. The generated source is gofmt'ed before it is written, and the
// candidate manifest records the contract and template digests.
func Generate(contractPath, strategy, outDir string) (Generated, error) {
	tmpl, ok := generatorTemplates[strategy]
	if !ok {
		return Generated{}, fmt.Errorf("unknown generator strategy %q", strategy)
	}
	contractRaw, err := os.ReadFile(contractPath)
	if err != nil {
		return Generated{}, err
	}
	var contract Contract
	if _, err := readJSON(contractPath, &contract); err != nil {
		return Generated{}, err
	}
	if err := validateContract(contract); err != nil {
		return Generated{}, err
	}
	templateRaw, err := templateFiles.ReadFile(tmpl.file)
	if err != nil {
		return Generated{}, err
	}
	parsed, err := template.New(tmpl.strategy).Parse(string(templateRaw))
	if err != nil {
		return Generated{}, err
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
		return Generated{}, err
	}
	formatted, err := format.Source([]byte(rendered.String()))
	if err != nil {
		return Generated{}, fmt.Errorf("generated source is not valid Go: %w", err)
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
	sourcePath := filepath.Join(outDir, "main.go")
	if err := os.WriteFile(sourcePath, formatted, 0o644); err != nil {
		cleanup()
		return Generated{}, err
	}
	manifest := CandidateManifest{
		Schema: CandidateSchema, ID: tmpl.strategy, Origin: tmpl.origin,
		Source: "main.go", SourceDigest: digest(formatted),
		Generator: &GeneratorProvenance{
			ID: tmpl.id, Version: "1", Strategy: tmpl.strategy,
			ContractDigest: data.ContractDigest, TemplateDigest: digest(templateRaw),
		},
	}
	manifestRaw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		cleanup()
		return Generated{}, err
	}
	manifestRaw = append(manifestRaw, '\n')
	if err := os.WriteFile(filepath.Join(outDir, "candidate.json"), manifestRaw, 0o644); err != nil {
		cleanup()
		return Generated{}, err
	}
	return Generated{Path: outDir, SourceDigest: manifest.SourceDigest, Manifest: manifest}, nil
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
