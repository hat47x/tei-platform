package generatorselection

// 用語を分けて扱う。
//   - Generator Source: TEI-native Transform Source (deterministic-transform-v1) のJSON。
//   - executing generator: Generator Sourceを評価する transform.Evaluate。
//   - Target artifact: 評価結果として得るGoソース(候補の main.go)。
//   - independent Expected: 手書きのfixture。このファイルは読まない。

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"tei/internal/ir"
	"tei/internal/plugin/transform"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

const (
	transformSourceFile = "generator.transform.json"
	transformInputFile  = "generator-input.json"
	// Provider と Model は Generation provenance の既存欄を使うための識別子で、
	// 外部のモデルではない。promptDigest は Generator Source、contextDigest は
	// その入力(受理済みValidationの写し)のdigestを持つ。
	transformProvider = "tei-transform-evaluator"
	// TransformOrigin は Source を決定的に評価して得た候補を表す。AIが書いた
	// Generator Sourceであっても、候補の権限根拠にはしない。
	TransformOrigin = "generated-transform"
)

var validationInputKeyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)

func validateValidationSpec(contract Contract) error {
	spec := contract.Validation
	if contract.Projection != nil {
		return fmt.Errorf("validation contracts must not set projection")
	}
	if contract.Rounding != (Rounding{}) {
		return fmt.Errorf("validation contracts must not set rounding")
	}
	if len(contract.Inputs) != 1 {
		return fmt.Errorf("validation contracts require exactly one input, got %d", len(contract.Inputs))
	}
	if strings.TrimSpace(spec.ID) == "" {
		return fmt.Errorf("validation id is required")
	}
	// 生成するGoは positive だけを表せる。他のkindは契約の形(relationなど)を
	// まだ持てないため、ここで閉じる。
	if spec.Kind != string(semantic.ValidationPositive) {
		return fmt.Errorf("validation kind %q is not supported; only %q", spec.Kind, semantic.ValidationPositive)
	}
	// 入力keyは生成Goの文字列定数へ入るため、引用符や改行を含められない形に限る。
	if !validationInputKeyPattern.MatchString(spec.Input) {
		return fmt.Errorf("validation input key %q must match %s", spec.Input, validationInputKeyPattern)
	}
	return nil
}

func findValidation(world ir.World, id string) (semantic.Validation, bool) {
	for _, ns := range world.Semantics {
		for _, validation := range ns.Validations {
			if string(validation.ID) == id {
				return validation, true
			}
		}
	}
	return semantic.Validation{}, false
}

// validateValidationAgainstWorkspace rejects a Validation that the accepted
// world does not hold, or one whose kind or attribute differs from what the
// contract claims. A proposed or unknown Validation therefore cannot qualify.
func validateValidationAgainstWorkspace(contract Contract, world ir.World) error {
	spec := contract.Validation
	validation, ok := findValidation(world, spec.ID)
	if !ok {
		return fmt.Errorf("accepted validation %s was not found in the Workspace", spec.ID)
	}
	if string(validation.Kind) != spec.Kind {
		return fmt.Errorf("validation %s is %s in the Workspace, contract says %s", spec.ID, validation.Kind, spec.Kind)
	}
	if string(validation.Attribute) != contract.Inputs[0].Attribute {
		return fmt.Errorf("validation %s applies to %s, contract input is %s",
			spec.ID, validation.Attribute, contract.Inputs[0].Attribute)
	}
	return nil
}

// TransformGeneratorOptions names the files for one generation. The
// Validation is read from the loaded Workspace, not from the contract's text.
type TransformGeneratorOptions struct {
	ID                  string
	WorkspaceDir        string
	ContractPath        string
	GeneratorSourcePath string
	OutDir              string
}

// evaluateValidationGenerator runs the executing generator on one Generator
// Source and returns the Target artifact bytes and the canonical generator
// input. A Source that is not valid, asserts a different kind, or yields
// anything but valid Go source is an error.
func evaluateValidationGenerator(sourceRaw []byte, validation semantic.Validation, inputKey string) (target, input []byte, err error) {
	source, err := transform.ParseSourceJSON(sourceRaw)
	if err != nil {
		return nil, nil, err
	}
	value, err := transform.Object(
		transform.Field{Name: "attribute", Value: transform.String(string(validation.Attribute))},
		transform.Field{Name: "id", Value: transform.String(string(validation.ID))},
		transform.Field{Name: "inputKey", Value: transform.String(inputKey)},
		transform.Field{Name: "kind", Value: transform.String(string(validation.Kind))},
	)
	if err != nil {
		return nil, nil, err
	}
	result, err := transform.Evaluate(source, map[string]transform.Value{"validation": value})
	if err != nil {
		return nil, nil, fmt.Errorf("evaluate Generator Source: %w", err)
	}
	if result.Output.Kind != transform.ValueString {
		return nil, nil, fmt.Errorf("Generator Source output is %s, want a Go source string", result.Output.Kind)
	}
	target = []byte(result.Output.String)
	if _, err := parser.ParseFile(token.NewFileSet(), "main.go", target, parser.AllErrors); err != nil {
		return nil, nil, fmt.Errorf("Generator Source output is not valid Go: %w", err)
	}
	input, err = value.JSON()
	if err != nil {
		return nil, nil, err
	}
	return target, append(input, '\n'), nil
}

// loadValidationContract loads the accepted Workspace and the contract, and
// returns the accepted Validation the contract names.
func loadValidationContract(workspaceDir, contractPath string) (Contract, []byte, semantic.Validation, error) {
	loaded, err := workspace.Load(workspaceDir)
	if err != nil {
		return Contract{}, nil, semantic.Validation{}, fmt.Errorf("load Workspace: %w", err)
	}
	var contract Contract
	contractRaw, err := readJSON(contractPath, &contract)
	if err != nil {
		return Contract{}, nil, semantic.Validation{}, err
	}
	if err := validateContract(contract); err != nil {
		return Contract{}, nil, semantic.Validation{}, err
	}
	if contract.Validation == nil {
		return Contract{}, nil, semantic.Validation{}, fmt.Errorf("contract %s has no validation", contract.ID)
	}
	if err := validateAgainstWorkspace(contract, loaded.World); err != nil {
		return Contract{}, nil, semantic.Validation{}, err
	}
	validation, _ := findValidation(loaded.World, contract.Validation.ID)
	return contract, contractRaw, validation, nil
}

// GenerateFromTransformSource evaluates a Generator Source on the accepted
// Validation and writes the Target artifact as a candidate (main.go plus
// candidate.json). Nothing is written when any step fails.
//
// The Generator Source digest goes into the existing Generation provenance
// (promptDigest, with the Source copied to generator.transform.json), because
// Lineage verification re-runs a Go generator and cannot accept a Transform
// Source. Select verifies those files by digest.
func GenerateFromTransformSource(opts TransformGeneratorOptions) (CandidateManifest, error) {
	if strings.TrimSpace(opts.ID) == "" || strings.TrimSpace(opts.OutDir) == "" ||
		strings.TrimSpace(opts.WorkspaceDir) == "" || strings.TrimSpace(opts.ContractPath) == "" ||
		strings.TrimSpace(opts.GeneratorSourcePath) == "" {
		return CandidateManifest{}, fmt.Errorf("id, workspace, contract, generator source, and out are required")
	}
	contract, contractRaw, validation, err := loadValidationContract(opts.WorkspaceDir, opts.ContractPath)
	if err != nil {
		return CandidateManifest{}, err
	}
	sourceRaw, err := os.ReadFile(opts.GeneratorSourcePath)
	if err != nil {
		return CandidateManifest{}, err
	}
	target, input, err := evaluateValidationGenerator(sourceRaw, validation, contract.Validation.Input)
	if err != nil {
		return CandidateManifest{}, err
	}
	if _, err := os.Lstat(opts.OutDir); err == nil {
		return CandidateManifest{}, fmt.Errorf("output directory already exists: %s", opts.OutDir)
	} else if !os.IsNotExist(err) {
		return CandidateManifest{}, err
	}
	manifest := CandidateManifest{
		Schema: CandidateSchema, ID: opts.ID, Version: "1", Origin: TransformOrigin,
		Source: "main.go", SourceDigest: digest(target),
		Note: "Target artifact evaluated from a TEI transform Generator Source (deterministic-transform-v1)",
		Generation: &GenerationProvenance{
			Provider: transformProvider, Model: transform.ProfileV1, ContractDigest: digest(contractRaw),
			PromptDigest: digest(sourceRaw), ContextDigest: digest(input), OutputDigest: digest(target),
			PromptPath: transformSourceFile, ContextPath: transformInputFile,
		},
	}
	manifestRaw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return CandidateManifest{}, err
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return CandidateManifest{}, err
	}
	writes := []struct {
		name    string
		content []byte
	}{
		{"main.go", target},
		{transformSourceFile, sourceRaw},
		{transformInputFile, input},
		{"candidate.json", append(manifestRaw, '\n')},
	}
	for _, write := range writes {
		if err := os.WriteFile(filepath.Join(opts.OutDir, write.name), write.content, 0o644); err != nil {
			_ = os.RemoveAll(opts.OutDir)
			return CandidateManifest{}, err
		}
	}
	return manifest, nil
}

// VerifyTransformCandidate re-evaluates the Generator Source stored in a
// candidate directory on the accepted Validation and requires the Target
// artifact to be byte-identical. Select does not call it: Select only checks
// the stored Source by digest.
func VerifyTransformCandidate(dir, workspaceDir, contractPath string) error {
	contract, _, validation, err := loadValidationContract(workspaceDir, contractPath)
	if err != nil {
		return err
	}
	var manifest CandidateManifest
	if _, err := readJSON(filepath.Join(dir, "candidate.json"), &manifest); err != nil {
		return err
	}
	if manifest.Generation == nil || manifest.Generation.PromptPath != transformSourceFile {
		return fmt.Errorf("candidate does not record a transform Generator Source")
	}
	sourceRaw, err := os.ReadFile(filepath.Join(dir, transformSourceFile))
	if err != nil {
		return err
	}
	if actual := digest(sourceRaw); actual != manifest.Generation.PromptDigest {
		return fmt.Errorf("Generator Source digest mismatch: declared %s, actual %s", manifest.Generation.PromptDigest, actual)
	}
	target, _, err := evaluateValidationGenerator(sourceRaw, validation, contract.Validation.Input)
	if err != nil {
		return err
	}
	stored, err := os.ReadFile(filepath.Join(dir, manifest.Source))
	if err != nil {
		return err
	}
	if string(stored) != string(target) {
		return fmt.Errorf("candidate source does not match the Generator Source evaluation")
	}
	return nil
}
