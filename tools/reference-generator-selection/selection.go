// Package generatorselection compares generator candidates for one accepted
// reference contract and records what each candidate actually produced. It
// selects by a fixed fixture; origin, self-report, and build success do not
// grant authority.
package generatorselection

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

//go:embed templates/wasi_node.js
var wasiRunner []byte

const (
	ContractSchema  = "tei.reference.generator-contract/v0"
	FixtureSchema   = "tei.reference.generator-fixture/v0"
	CandidateSchema = "tei.reference.generator-candidate/v0"
	ReceiptSchema   = "tei.reference.generator-selection-receipt/v0"
)

const (
	LanguageGo         = "go"
	LanguagePython     = "python"
	LanguageJavaScript = "javascript"
	LanguageJava       = "java"
	LanguageWasm       = "wasm"
	LanguageWasi       = "wasi"
)

type InputSpec struct {
	Attribute string `json:"attribute"`
	Primitive string `json:"primitive"`
}

type TypeSpec struct {
	Primitive string `json:"primitive"`
}

type Rounding struct {
	Precision int    `json:"precision"`
	Scale     int    `json:"scale"`
	Mode      string `json:"mode"`
}

type Contract struct {
	Schema     string          `json:"schema"`
	ID         string          `json:"id"`
	Version    string          `json:"version"`
	Status     string          `json:"status"`
	Note       string          `json:"note,omitempty"`
	Inputs     []InputSpec     `json:"inputs"`
	Output     TypeSpec        `json:"output"`
	Rounding   Rounding        `json:"rounding"`
	Projection *ProjectionSpec `json:"projection,omitempty"`
	Validation *ValidationSpec `json:"validation,omitempty"`
}

// ValidationSpec marks a contract for one accepted Validation. The result state
// is satisfied, not-satisfied, or unevaluable (Code: missing, null,
// type-mismatch). Unevaluable mirrors validationeval.Evaluate returning an
// error rather than NotSatisfied. ID and Kind must match the accepted
// Validation in the Workspace; Input is the key the single contract input
// arrives under. Rounding and Projection must be absent, and Output only
// carries the state text.
type ValidationSpec struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Input string `json:"input"`
}

// ProjectionSpec marks a non-numeric reference contract. Identity keeps the
// input value; normalize trims surrounding whitespace; record projects several
// named fields. Rounding must be absent.
type ProjectionSpec struct {
	Kind   string            `json:"kind"`
	Fields []ProjectionField `json:"fields,omitempty"`
}

// ProjectionField maps one input key to one output field.
type ProjectionField struct {
	Name      string `json:"name"`
	Input     string `json:"input"`
	Attribute string `json:"attribute"`
	Primitive string `json:"primitive"`
	Many      bool   `json:"many,omitempty"`
	Optional  bool   `json:"optional,omitempty"`
}

type Case struct {
	ID    string          `json:"id"`
	Input json.RawMessage `json:"input"`
	Want  Result          `json:"want"`
}

type Fixture struct {
	Schema          string `json:"schema"`
	ContractID      string `json:"contractId"`
	ContractVersion string `json:"contractVersion"`
	Cases           []Case `json:"cases"`
}

type Result struct {
	State    string `json:"state"`
	Amount   string `json:"amount,omitempty"`
	Currency string `json:"currency,omitempty"`
	Code     string `json:"code,omitempty"`
	Text     string `json:"text,omitempty"`
}

type GenerationProvenance struct {
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	ContractDigest string `json:"contractDigest"`
	PromptDigest   string `json:"promptDigest"`
	ContextDigest  string `json:"contextDigest"`
	OutputDigest   string `json:"outputDigest"`
	PromptPath     string `json:"promptPath,omitempty"`
	ContextPath    string `json:"contextPath,omitempty"`
}

type Lineage struct {
	Depth                 int    `json:"depth"`
	GeneratorSource       string `json:"generatorSource"`
	GeneratorSourceDigest string `json:"generatorSourceDigest"`
	GeneratorOrigin       string `json:"generatorOrigin,omitempty"`
}

type CandidateManifest struct {
	Schema       string                `json:"schema"`
	ID           string                `json:"id"`
	Version      string                `json:"version,omitempty"`
	Language     string                `json:"language,omitempty"`
	Origin       string                `json:"origin"`
	Source       string                `json:"source"`
	SourceDigest string                `json:"sourceDigest"`
	Note         string                `json:"note,omitempty"`
	Generator    *GeneratorProvenance  `json:"generator,omitempty"`
	Generation   *GenerationProvenance `json:"generation,omitempty"`
	Lineage      *Lineage              `json:"lineage,omitempty"`
}

// RuntimeEvidence records which interpreter or build tool ran the candidate.
// For Python there is no compiled artifact, so BinaryDigest is empty.
type RuntimeEvidence struct {
	Language  string `json:"language"`
	Command   string `json:"command"`
	Version   string `json:"version,omitempty"`
	Available bool   `json:"available"`
}

// ExecutionEvidence records what the harness observed around one case
// execution. It is a digest of stdout/stderr and the process exit state, not a
// copy of the content and not a claim about a real target.
type ExecutionEvidence struct {
	Measurement  *ProcessMeasurement `json:"measurement,omitempty"`
	ExitCode     int                 `json:"exitCode"`
	StdoutDigest string              `json:"stdoutDigest"`
	StderrDigest string              `json:"stderrDigest"`
	TimedOut     bool                `json:"timedOut,omitempty"`
	// SourceDigest is the digest of the execution snapshot handed to the
	// interpreter, read immediately before this case ran. The snapshot is
	// written from the bytes verified against the manifest digest. It is not an
	// OS-level observation of what the interpreter read. It is empty for
	// compiled candidates.
	SourceDigest string `json:"sourceDigest,omitempty"`
}

type CaseResult struct {
	CaseID    string             `json:"caseId"`
	Passed    bool               `json:"passed"`
	Got       Result             `json:"got"`
	Want      Result             `json:"want"`
	Execution *ExecutionEvidence `json:"execution,omitempty"`
}

// BuildEvidence records the source and toolchain inputs used to build a native
// candidate. It is not a signature or an attestation, and it does not claim
// that the binary is reproducible across environments.
type BuildEvidence struct {
	Language         string `json:"language"`
	Command          string `json:"command"`
	Toolchain        string `json:"toolchain,omitempty"`
	GOOS             string `json:"goos,omitempty"`
	GOARCH           string `json:"goarch,omitempty"`
	CGOEnabled       string `json:"cgoEnabled,omitempty"`
	GOFLAGS          string `json:"goflags,omitempty"`
	SourceDigest     string `json:"sourceDigest"`
	BinaryDigest     string `json:"binaryDigest,omitempty"`
	ReproducedDigest string `json:"reproducedDigest,omitempty"`
	BuildInputDigest string `json:"buildInputDigest"`
	Reproducibility  string `json:"reproducibility"`
}

type CandidateReceipt struct {
	ID            string                `json:"id"`
	Language      string                `json:"language"`
	Origin        string                `json:"origin"`
	SourceDigest  string                `json:"sourceDigest"`
	BinaryDigest  string                `json:"binaryDigest,omitempty"`
	Runtime       *RuntimeEvidence      `json:"runtime,omitempty"`
	BuildEvidence *BuildEvidence        `json:"buildEvidence,omitempty"`
	Generator     *GeneratorProvenance  `json:"generator,omitempty"`
	Generation    *GenerationProvenance `json:"generation,omitempty"`
	Lineage       *Lineage              `json:"lineage,omitempty"`
	Results       []CaseResult          `json:"results,omitempty"`
	Passed        bool                  `json:"passed"`
	RejectReason  string                `json:"rejectReason,omitempty"`
}

type Receipt struct {
	Execution              ExecutionOptions   `json:"execution,omitempty"`
	Schema                 string             `json:"schema"`
	ContractID             string             `json:"contractId"`
	ContractVersion        string             `json:"contractVersion"`
	ContractStatus         string             `json:"contractStatus"`
	WorkspaceRevision      string             `json:"workspaceRevision"`
	FixtureDigest          string             `json:"fixtureDigest"`
	SuiteVersion           string             `json:"suiteVersion"`
	Candidates             []CandidateReceipt `json:"candidates"`
	Qualified              []string           `json:"qualified"`
	Rejected               []string           `json:"rejected"`
	Passed                 bool               `json:"passed"`
	GrantsRuntimeAuthority bool               `json:"grantsRuntimeAuthority"`
	ProvesRealProvider     bool               `json:"provesRealProvider"`
	Notes                  []string           `json:"notes"`
}

func digest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func validateGeneration(g *GenerationProvenance) error {
	if strings.TrimSpace(g.Provider) == "" || strings.TrimSpace(g.Model) == "" {
		return fmt.Errorf("generation provider and model are required")
	}
	for name, value := range map[string]string{
		"contractDigest": g.ContractDigest, "promptDigest": g.PromptDigest,
		"contextDigest": g.ContextDigest, "outputDigest": g.OutputDigest,
	} {
		if !digestPattern.MatchString(value) {
			return fmt.Errorf("generation %s is not a sha256 digest", name)
		}
	}
	return nil
}

func candidateRelativePath(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || strings.HasPrefix(value, "/") || filepath.IsAbs(value) {
		return "", fmt.Errorf("path %q must be relative", value)
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return "", fmt.Errorf("path %q must not contain ..", value)
		}
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path %q escapes the candidate", value)
	}
	return clean, nil
}

// verifyGenerationFiles checks prompt/context content when the candidate
// carries their relative paths. A digest without the content is still
// recorded, but it is not an independent verification of what the model saw.
func verifyGenerationFiles(dir string, g *GenerationProvenance) error {
	checks := []struct{ name, path, want string }{
		{"prompt", g.PromptPath, g.PromptDigest},
		{"context", g.ContextPath, g.ContextDigest},
	}
	for _, check := range checks {
		if check.path == "" {
			continue
		}
		relative, err := candidateRelativePath(check.path)
		if err != nil {
			return fmt.Errorf("%s: %w", check.name, err)
		}
		full := filepath.Join(dir, filepath.FromSlash(relative))
		info, err := os.Lstat(full)
		if err != nil {
			return fmt.Errorf("%s: %w", check.name, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s %s is a symbolic link", check.name, relative)
		}
		content, err := os.ReadFile(full)
		if err != nil {
			return fmt.Errorf("%s: %w", check.name, err)
		}
		if actual := digest(content); actual != check.want {
			return fmt.Errorf("%s digest mismatch: declared %s, actual %s", check.name, check.want, actual)
		}
	}
	return nil
}

func verifyLineage(ctx context.Context, dir, contractPath string, lineage *Lineage, source []byte) error {
	if lineage.Depth != 1 {
		return fmt.Errorf("generator lineage depth %d is not supported; only depth 1 is allowed", lineage.Depth)
	}
	relative, err := candidateRelativePath(lineage.GeneratorSource)
	if err != nil {
		return err
	}
	full := filepath.Join(dir, filepath.FromSlash(relative))
	info, err := os.Lstat(full)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("generator source %s is a symbolic link", relative)
	}
	generatorSource, err := os.ReadFile(full)
	if err != nil {
		return err
	}
	if actual := digest(generatorSource); actual != lineage.GeneratorSourceDigest {
		return fmt.Errorf("generator source digest mismatch: declared %s, actual %s", lineage.GeneratorSourceDigest, actual)
	}
	absoluteContract, err := filepath.Abs(contractPath)
	if err != nil {
		return err
	}
	generated, err := runMetaGenerator(ctx, generatorSource, absoluteContract, 2*time.Minute)
	if err != nil {
		return err
	}
	if !bytes.Equal(generated, source) {
		return fmt.Errorf("candidate source does not match the recorded generator output")
	}
	return nil
}

func readJSON(path string, into any) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return raw, nil
}

func validateContract(contract Contract) error {
	if contract.Schema != ContractSchema {
		return fmt.Errorf("contract schema %q is not %q", contract.Schema, ContractSchema)
	}
	if strings.TrimSpace(contract.ID) == "" || strings.TrimSpace(contract.Version) == "" {
		return fmt.Errorf("contract id and version are required")
	}
	if len(contract.Inputs) == 0 {
		return fmt.Errorf("contract inputs must not be empty")
	}
	for _, input := range contract.Inputs {
		if strings.TrimSpace(input.Attribute) == "" || strings.TrimSpace(input.Primitive) == "" {
			return fmt.Errorf("contract input attribute and primitive are required")
		}
	}
	if strings.TrimSpace(contract.Output.Primitive) == "" {
		return fmt.Errorf("contract output primitive is required")
	}
	if contract.Validation != nil {
		return validateValidationSpec(contract)
	}
	if contract.Projection != nil {
		switch contract.Projection.Kind {
		case "identity", "normalize":
			if len(contract.Projection.Fields) != 0 {
				return fmt.Errorf("a single-value projection must not declare fields")
			}
		case "record":
			if len(contract.Projection.Fields) == 0 {
				return fmt.Errorf("a record projection requires fields")
			}
			inputsByAttribute := map[string]InputSpec{}
			for _, input := range contract.Inputs {
				inputsByAttribute[input.Attribute] = input
			}
			seen := map[string]bool{}
			seenInputs := map[string]bool{}
			for _, field := range contract.Projection.Fields {
				if strings.TrimSpace(field.Name) == "" || strings.TrimSpace(field.Input) == "" ||
					strings.TrimSpace(field.Attribute) == "" || strings.TrimSpace(field.Primitive) == "" {
					return fmt.Errorf("record fields require name, input, attribute, and primitive")
				}
				if seen[field.Name] {
					return fmt.Errorf("record field %q is duplicated", field.Name)
				}
				seen[field.Name] = true
				// The runtime reads one map key per field, so two fields sharing
				// an input key would silently read the same value.
				if seenInputs[field.Input] {
					return fmt.Errorf("record field input %q is duplicated", field.Input)
				}
				seenInputs[field.Input] = true
				input, ok := inputsByAttribute[field.Attribute]
				if !ok {
					return fmt.Errorf("record field %q references undeclared contract input attribute %q", field.Name, field.Attribute)
				}
				if field.Primitive != input.Primitive {
					return fmt.Errorf("record field %q primitive %q does not match contract input %q primitive %q",
						field.Name, field.Primitive, field.Attribute, input.Primitive)
				}
				// The record template renders many fields as strings. Accepting a
				// many field with another primitive would pass validation and then
				// fail at runtime on the first element (#2585 PFC-06).
				if field.Many && field.Primitive != "std.text" && field.Primitive != "std.identifier" {
					return fmt.Errorf("record field %q: many is not supported for primitive %q in this Target profile",
						field.Name, field.Primitive)
				}
			}
		default:
			return fmt.Errorf("projection kind must be identity, normalize, or record")
		}
		if contract.Rounding != (Rounding{}) {
			return fmt.Errorf("projection contracts must not set rounding")
		}
		return nil
	}
	if contract.Rounding.Precision < 1 || contract.Rounding.Scale < 0 || contract.Rounding.Scale > contract.Rounding.Precision {
		return fmt.Errorf("contract rounding precision/scale is invalid")
	}
	if contract.Rounding.Mode != "half-up" && contract.Rounding.Mode != "half-even" {
		return fmt.Errorf("contract rounding mode must be half-up or half-even")
	}
	return nil
}

func typeIndex(world ir.World) map[string]semantic.Type {
	index := map[string]semantic.Type{}
	for _, ns := range world.Semantics {
		for _, typ := range ns.Types {
			index[string(typ.ID)] = typ
		}
	}
	return index
}

func primitiveOf(types map[string]semantic.Type, id string) (string, error) {
	visited := map[string]bool{}
	for {
		if visited[id] {
			return "", fmt.Errorf("cyclic type alias at %s", id)
		}
		visited[id] = true
		typ, ok := types[id]
		if !ok {
			return "", fmt.Errorf("unknown type %s", id)
		}
		switch typ.Kind {
		case semantic.TypePrimitive:
			return string(typ.ID), nil
		case semantic.TypeAlias:
			if typ.Underlying == "" {
				return "", fmt.Errorf("alias %s has no underlying type", id)
			}
			id = string(typ.Underlying)
		default:
			return "", fmt.Errorf("type %s is not a primitive or alias", id)
		}
	}
}

func findAttribute(world ir.World, id string) (semantic.Attribute, bool) {
	for _, ns := range world.Semantics {
		for _, data := range ns.Data {
			for _, attribute := range data.Attributes {
				if string(attribute.ID) == id {
					return attribute, true
				}
			}
		}
	}
	return semantic.Attribute{}, false
}

func validateAgainstWorkspace(contract Contract, world ir.World) error {
	types := typeIndex(world)
	for _, input := range contract.Inputs {
		attribute, ok := findAttribute(world, input.Attribute)
		if !ok {
			return fmt.Errorf("accepted attribute %s was not found in the Workspace", input.Attribute)
		}
		primitive, err := primitiveOf(types, string(attribute.TypeRef))
		if err != nil {
			return fmt.Errorf("resolve %s: %w", input.Attribute, err)
		}
		if primitive != input.Primitive {
			return fmt.Errorf("%s resolves to %s, contract requires %s", input.Attribute, primitive, input.Primitive)
		}
	}
	if _, err := primitiveOf(types, contract.Output.Primitive); err != nil {
		return fmt.Errorf("output primitive %s: %w", contract.Output.Primitive, err)
	}
	if contract.Validation != nil {
		if err := validateValidationAgainstWorkspace(contract, world); err != nil {
			return err
		}
	}
	// #2585 PFC-08: validate the projection fields directly against the
	// Workspace, not only through the contract inputs. A field whose attribute
	// is absent or resolves to another primitive is a contract error even when
	// an unrelated input still resolves.
	if contract.Projection != nil && contract.Projection.Kind == "record" {
		for _, field := range contract.Projection.Fields {
			attribute, ok := findAttribute(world, field.Attribute)
			if !ok {
				return fmt.Errorf("projection field %q references attribute %s which was not found in the Workspace", field.Name, field.Attribute)
			}
			primitive, err := primitiveOf(types, string(attribute.TypeRef))
			if err != nil {
				return fmt.Errorf("resolve projection field %q attribute %s: %w", field.Name, field.Attribute, err)
			}
			if primitive != field.Primitive {
				return fmt.Errorf("projection field %q attribute %s resolves to %s, field requires %s",
					field.Name, field.Attribute, primitive, field.Primitive)
			}
		}
	}
	return nil
}

func validateFixture(fixture Fixture, contract Contract) error {
	if fixture.Schema != FixtureSchema {
		return fmt.Errorf("fixture schema %q is not %q", fixture.Schema, FixtureSchema)
	}
	if fixture.ContractID != contract.ID || fixture.ContractVersion != contract.Version {
		return fmt.Errorf("fixture targets %s/%s, contract is %s/%s",
			fixture.ContractID, fixture.ContractVersion, contract.ID, contract.Version)
	}
	if len(fixture.Cases) == 0 {
		return fmt.Errorf("fixture has no cases")
	}
	seen := map[string]bool{}
	for _, tc := range fixture.Cases {
		if tc.ID == "" {
			return fmt.Errorf("fixture case id is required")
		}
		if seen[tc.ID] {
			return fmt.Errorf("duplicate fixture case %s", tc.ID)
		}
		seen[tc.ID] = true
	}
	return nil
}

// Select loads one accepted Workspace, one reference contract, one fixed
// fixture, and every candidate directory under candidatesDir.
func Select(ctx context.Context, workspaceDir, contractPath, fixturePath, candidatesDir string) (Receipt, error) {
	return selectWithRegistry(ctx, workspaceDir, contractPath, fixturePath, candidatesDir, "")
}

// SelectWithRegistry additionally requires every candidate to be the active
// registered version for its id. A registry constrains selection; it does not
// grant authority.
func SelectWithRegistry(ctx context.Context, workspaceDir, contractPath, fixturePath, candidatesDir, registryPath string) (Receipt, error) {
	return selectWithRegistry(ctx, workspaceDir, contractPath, fixturePath, candidatesDir, registryPath)
}

// SelectForTarget restricts evaluation to the listed generator ids. Candidates
// outside the list are reported as rejected, so a profile cannot silently fall
// back to another generator.
func SelectForTarget(ctx context.Context, workspaceDir, contractPath, fixturePath, candidatesDir, registryPath string, allowedIDs []string) (Receipt, error) {
	return selectWithRegistryAndFilter(ctx, workspaceDir, contractPath, fixturePath, candidatesDir, registryPath, allowedIDs, ExecutionOptions{})
}

// SelectForTargetWithExecution applies a wrapper to every candidate fixture
// process. Lineage execution is rejected because it has a separate runner.
func SelectForTargetWithExecution(ctx context.Context, workspaceDir, contractPath, fixturePath, candidatesDir, registryPath string, allowedIDs []string, execution ExecutionOptions) (Receipt, error) {
	return selectWithRegistryAndFilter(ctx, workspaceDir, contractPath, fixturePath, candidatesDir, registryPath, allowedIDs, execution)
}

func selectWithRegistry(ctx context.Context, workspaceDir, contractPath, fixturePath, candidatesDir, registryPath string) (Receipt, error) {
	return selectWithRegistryAndFilter(ctx, workspaceDir, contractPath, fixturePath, candidatesDir, registryPath, nil, ExecutionOptions{})
}

func selectWithRegistryAndFilter(ctx context.Context, workspaceDir, contractPath, fixturePath, candidatesDir, registryPath string, allowedIDs []string, execution ExecutionOptions) (Receipt, error) {
	loaded, err := workspace.Load(workspaceDir)
	if err != nil {
		return Receipt{}, fmt.Errorf("load Workspace: %w", err)
	}
	index, err := workspace.BuildSourceIndex(workspaceDir, loaded)
	if err != nil {
		return Receipt{}, fmt.Errorf("build Source Index: %w", err)
	}
	contractRaw, err := os.ReadFile(contractPath)
	if err != nil {
		return Receipt{}, err
	}
	var contract Contract
	if _, err := readJSON(contractPath, &contract); err != nil {
		return Receipt{}, err
	}
	if err := validateContract(contract); err != nil {
		return Receipt{}, err
	}
	if err := validateAgainstWorkspace(contract, loaded.World); err != nil {
		return Receipt{}, err
	}
	var registry *Registry
	if registryPath != "" {
		loadedRegistry, err := LoadRegistry(registryPath, contractPath)
		if err != nil {
			return Receipt{}, err
		}
		registry = &loadedRegistry
	}
	fixtureRaw, err := os.ReadFile(fixturePath)
	if err != nil {
		return Receipt{}, err
	}
	var fixture Fixture
	if _, err := readJSON(fixturePath, &fixture); err != nil {
		return Receipt{}, err
	}
	if err := validateFixture(fixture, contract); err != nil {
		return Receipt{}, err
	}

	entries, err := os.ReadDir(candidatesDir)
	if err != nil {
		return Receipt{}, err
	}
	receipt := Receipt{
		Execution: execution,
		Schema:    ReceiptSchema, ContractID: contract.ID, ContractVersion: contract.Version,
		ContractStatus: contract.Status, WorkspaceRevision: sourceindex.ModelRevision(index),
		FixtureDigest: digest(fixtureRaw),
		SuiteVersion:  digest(append(append(append([]byte("v0\x00"), contractRaw...), 0), fixtureRaw...)),
		Qualified:     []string{}, Rejected: []string{},
		GrantsRuntimeAuthority: false, ProvesRealProvider: false,
		Notes: []string{
			"受理済みWorkspaceの属性・型と、参照用の複合契約を使った選抜です。",
			"候補のorigin・自己申告・build成功は、正しさや権限の根拠にしません。",
			"選抜は固定fixtureの範囲だけを示し、実Provider・実Target・本番を証明しません。",
		},
	}
	seen := map[string]bool{}
	allowed := map[string]bool{}
	for _, id := range allowedIDs {
		allowed[id] = true
	}
	var directories []string
	for _, entry := range entries {
		if entry.IsDir() {
			directories = append(directories, entry.Name())
		}
	}
	sort.Strings(directories)
	for _, name := range directories {
		candidate, err := evaluateCandidate(ctx, filepath.Join(candidatesDir, name), fixture, contractPath, digest(contractRaw), registry, allowed, execution)
		if err != nil {
			return Receipt{}, err
		}
		if seen[candidate.ID] {
			return Receipt{}, fmt.Errorf("duplicate candidate id %q", candidate.ID)
		}
		seen[candidate.ID] = true
		receipt.Candidates = append(receipt.Candidates, candidate)
		if candidate.Passed {
			receipt.Qualified = append(receipt.Qualified, candidate.ID)
		} else {
			receipt.Rejected = append(receipt.Rejected, candidate.ID)
		}
	}
	receipt.Passed = len(receipt.Qualified) > 0
	return receipt, nil
}

func evaluateCandidate(ctx context.Context, dir string, fixture Fixture, contractPath, contractDigest string, registry *Registry, allowed map[string]bool, executionOptions ExecutionOptions) (CandidateReceipt, error) {
	var manifest CandidateManifest
	if _, err := readJSON(filepath.Join(dir, "candidate.json"), &manifest); err != nil {
		return CandidateReceipt{}, err
	}
	language := candidateLanguage(manifest)
	receipt := CandidateReceipt{
		ID: manifest.ID, Language: language, Origin: manifest.Origin, SourceDigest: manifest.SourceDigest,
		Generator: manifest.Generator, Generation: manifest.Generation, Lineage: manifest.Lineage,
	}
	reject := func(reason string) (CandidateReceipt, error) {
		receipt.Passed = false
		receipt.RejectReason = reason
		return receipt, nil
	}
	if manifest.Schema != CandidateSchema {
		return reject("candidate schema mismatch")
	}
	if manifest.ID == "" || manifest.Origin == "" || manifest.Source == "" || manifest.SourceDigest == "" {
		return reject("candidate identity is incomplete")
	}
	if language != LanguageGo && language != LanguagePython && language != LanguageJavaScript && language != LanguageJava && language != LanguageWasm && language != LanguageWasi {
		return reject("unsupported candidate language: " + language)
	}
	expectedExtension := ".go"
	switch language {
	case LanguagePython:
		expectedExtension = ".py"
	case LanguageJavaScript:
		expectedExtension = ".js"
	case LanguageJava:
		expectedExtension = ".java"
	case LanguageWasm, LanguageWasi:
		expectedExtension = ".go"
	}
	if strings.ToLower(filepath.Ext(manifest.Source)) != expectedExtension {
		return reject("candidate source extension does not match language " + language)
	}
	if len(allowed) > 0 && !allowed[manifest.ID] {
		return reject("not selected by target profile")
	}
	if registry != nil {
		if versionEntry, ok := Entry(*registry, manifest.ID, candidateVersion(manifest)); ok {
			switch versionEntry.Status {
			case RegistryRevoked:
				return reject("registry: candidate version is revoked")
			case RegistrySuperseded:
				return reject("registry: candidate version is superseded")
			}
		}
		entry, err := ActiveEntry(*registry, manifest.ID)
		if err != nil {
			return reject("registry: " + err.Error())
		}
		if entry.Version != candidateVersion(manifest) {
			return reject("registry version mismatch: active " + entry.Version + ", candidate " + candidateVersion(manifest))
		}
		if entry.SourceDigest != manifest.SourceDigest {
			return reject("registry source digest mismatch")
		}
		if entry.Language != "" && entry.Language != language {
			return reject("registry language mismatch: active " + entry.Language + ", candidate " + language)
		}
	}
	if manifest.Generator != nil {
		if manifest.Generator.ID == "" || manifest.Generator.Version == "" || manifest.Generator.Strategy == "" {
			return reject("generator provenance is incomplete")
		}
		if manifest.Generator.ContractDigest != contractDigest {
			return reject("generator contract digest mismatch")
		}
		if manifest.Generator.Language != "" && manifest.Generator.Language != language {
			return reject("generator language mismatch")
		}
	}
	if manifest.Origin == "generated-ai" && manifest.Generation == nil {
		return reject("generated-ai candidate has no generation provenance")
	}
	if manifest.Generation != nil {
		if err := validateGeneration(manifest.Generation); err != nil {
			return reject(err.Error())
		}
		if manifest.Generation.ContractDigest != contractDigest {
			return reject("generation contract digest mismatch")
		}
	}
	sourcePath, err := candidateSourcePath(dir, manifest.Source)
	if err != nil {
		return reject("candidate source is not contained: " + err.Error())
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return reject("source is unreadable: " + err.Error())
	}
	actualDigest := digest(source)
	if actualDigest != manifest.SourceDigest {
		return reject("source digest mismatch: declared " + manifest.SourceDigest + ", actual " + actualDigest)
	}
	if expectedExtension == ".go" {
		built, err := goSourceFiles(dir)
		if err != nil {
			return reject("go source selection failed: " + err.Error())
		}
		declaredName := filepath.Clean(filepath.FromSlash(manifest.Source))
		bound := false
		for _, name := range built {
			if name == declaredName {
				bound = true
				break
			}
		}
		if !bound {
			return reject("declared source " + manifest.Source + " is not among the files passed to go build: " + strings.Join(built, ", "))
		}
	}
	if manifest.Generation != nil && manifest.Generation.OutputDigest != actualDigest {
		return reject("generation output digest does not match the candidate source")
	}
	if manifest.Generation != nil {
		if err := verifyGenerationFiles(dir, manifest.Generation); err != nil {
			return reject("generation file verification failed: " + err.Error())
		}
	}
	if manifest.Lineage != nil {
		if len(executionOptions.CommandPrefix) > 0 {
			return reject("lineage execution is unsupported with a command wrapper")
		}
		if err := verifyLineage(ctx, dir, contractPath, manifest.Lineage, source); err != nil {
			return reject("generator lineage verification failed: " + err.Error())
		}
	}
	if manifest.Generator != nil {
		declared := *manifest.Generator
		if declared.Language == "" {
			declared.Language = language
		}
		rendered, err := renderCandidateVersion(contractPath, declared.Strategy, declared.Version)
		if err != nil {
			return reject("generator provenance cannot be reproduced: " + err.Error())
		}
		if !bytes.Equal(rendered.Source, source) {
			return reject("source does not match the declared generator and contract")
		}
		if rendered.Manifest.Generator == nil || *rendered.Manifest.Generator != declared {
			return reject("generator provenance does not match the current contract and template")
		}
	}
	// The bytes verified above are the only bytes of the declared source that
	// reach a build or an interpreter (#2595). The candidate directory is not
	// read again for it: a build runs from a selector-private staging directory
	// and a script runs from a selector-private execution snapshot.
	buildDir := dir
	if expectedExtension == ".go" {
		staged, err := stageVerifiedGoSources(dir, manifest.Source, source)
		if err != nil {
			return reject("verified source could not be staged for the build: " + err.Error())
		}
		defer os.RemoveAll(staged)
		buildDir = staged
	}
	var snapshotPath string
	var snapshot snapshotState
	var binaryPath, scriptPath, javaPath, javacPath, javaClassesDir, wasmPath, wasmExecPath, nodePath, wasiRunnerPath string
	switch language {
	case LanguageGo:
		binaryDir, err := os.MkdirTemp("", "tei-generator-candidate-*")
		if err != nil {
			return CandidateReceipt{}, err
		}
		defer os.RemoveAll(binaryDir)
		binaryPath = candidateBinaryPath(filepath.Join(binaryDir, manifest.ID))
		toolchain := goRuntimeVersion()
		goos, goarch, cgo := goBuildEnvironment()
		goflags := os.Getenv("GOFLAGS")
		receipt.Runtime = &RuntimeEvidence{Language: LanguageGo, Command: buildCommand, Version: toolchain, Available: true}
		receipt.BuildEvidence = &BuildEvidence{
			Language: LanguageGo, Command: buildCommand, Toolchain: toolchain,
			GOOS: goos, GOARCH: goarch, CGOEnabled: cgo, GOFLAGS: goflags,
			SourceDigest:     manifest.SourceDigest,
			BuildInputDigest: buildInputDigest(source, toolchain, goos, goarch, cgo, goflags),
			Reproducibility:  "not-tested",
		}
		buildCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		reproduced, err := buildCandidateTwice(buildCtx, buildDir, binaryPath)
		if err != nil {
			receipt.BuildEvidence.Reproducibility = "mismatch"
			return reject("build is not reproducible: " + err.Error())
		}
		receipt.BuildEvidence.ReproducedDigest = reproduced
		receipt.BuildEvidence.Reproducibility = "path-independent-twice"
		binary, err := os.ReadFile(binaryPath)
		if err != nil {
			return CandidateReceipt{}, err
		}
		receipt.BinaryDigest = digest(binary)
		receipt.BuildEvidence.BinaryDigest = receipt.BinaryDigest
	case LanguagePython, LanguageJavaScript:
		scriptPath, receipt.Runtime = ResolveScriptRuntime(language)
		if !receipt.Runtime.Available {
			return reject("runtime unavailable: " + language + " interpreter was not found")
		}
		snapshotDir, err := os.MkdirTemp("", "tei-script-snapshot-*")
		if err != nil {
			return CandidateReceipt{}, err
		}
		defer os.RemoveAll(snapshotDir)
		snapshotPath = filepath.Join(snapshotDir, filepath.Base(sourcePath))
		if err := os.WriteFile(snapshotPath, source, 0o600); err != nil {
			return CandidateReceipt{}, err
		}
		snapshot, err = readSnapshotState(snapshotPath)
		if err != nil {
			return CandidateReceipt{}, err
		}
		if snapshot.digest != actualDigest {
			return reject("execution snapshot does not match the verified source")
		}
	case LanguageJava:
		var version string
		var available bool
		javacPath, javaPath, version, available = findJava()
		command := "javac; java -cp <classes> Main"
		if javaPath != "" {
			command = "javac; " + javaPath + " -cp <classes> Main"
		}
		receipt.Runtime = &RuntimeEvidence{
			Language: LanguageJava, Command: command, Version: version, Available: available,
		}
		if !available {
			return reject("runtime unavailable: javac/java was not found")
		}
		javaClassesDir, err = os.MkdirTemp("", "tei-java-candidate-*")
		if err != nil {
			return CandidateReceipt{}, err
		}
		defer os.RemoveAll(javaClassesDir)
		compileCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		reproduced, err := buildJavaTwice(compileCtx, javacPath, dir, javaClassesDir)
		if err != nil {
			return reject("java compile is not reproducible: " + err.Error())
		}
		classBytes, err := os.ReadFile(filepath.Join(javaClassesDir, "Main.class"))
		if err != nil {
			return reject("compiled class is unreadable: " + err.Error())
		}
		receipt.BuildEvidence = &BuildEvidence{
			Language: LanguageJava, Command: "javac", Toolchain: version,
			SourceDigest: manifest.SourceDigest, BinaryDigest: digest(classBytes),
			ReproducedDigest: reproduced, BuildInputDigest: javaBuildInputDigest(source, version),
			Reproducibility: "path-independent-twice",
		}
		receipt.BinaryDigest = receipt.BuildEvidence.BinaryDigest
	case LanguageWasm:
		var version string
		var available bool
		nodePath, wasmExecPath, version, available = resolveWasmRuntime()
		receipt.Runtime = &RuntimeEvidence{
			Language: LanguageWasm, Command: "node wasm_exec_node.js <wasm>", Version: version, Available: available,
		}
		if !available {
			return reject("runtime unavailable: node or Go wasm_exec_node.js was not found")
		}
		wasmDir, err := os.MkdirTemp("", "tei-wasm-candidate-*")
		if err != nil {
			return CandidateReceipt{}, err
		}
		defer os.RemoveAll(wasmDir)
		wasmPath = filepath.Join(wasmDir, "candidate.wasm")
		buildCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		reproduced, err := buildWasmTwice(buildCtx, buildDir, wasmPath)
		if err != nil {
			return reject("wasm build is not reproducible: " + err.Error())
		}
		binary, err := os.ReadFile(wasmPath)
		if err != nil {
			return CandidateReceipt{}, err
		}
		toolchain := goRuntimeVersion()
		receipt.BuildEvidence = &BuildEvidence{
			Language: LanguageWasm, Command: "GOOS=js GOARCH=wasm go build -trimpath -buildvcs=false",
			Toolchain: toolchain, GOOS: "js", GOARCH: "wasm", CGOEnabled: "0",
			SourceDigest: manifest.SourceDigest, BinaryDigest: digest(binary),
			ReproducedDigest: reproduced, BuildInputDigest: wasmBuildInputDigest(source, toolchain),
			Reproducibility: "path-independent-twice",
		}
		receipt.BinaryDigest = receipt.BuildEvidence.BinaryDigest
	case LanguageWasi:
		var version string
		var available bool
		nodePath, version, available = resolveWasiRuntime()
		receipt.Runtime = &RuntimeEvidence{
			Language: LanguageWasi, Command: "node wasi_node.js <wasm>", Version: version, Available: available,
		}
		if !available {
			return reject("runtime unavailable: node was not found")
		}
		wasiDir, err := os.MkdirTemp("", "tei-wasi-candidate-*")
		if err != nil {
			return CandidateReceipt{}, err
		}
		defer os.RemoveAll(wasiDir)
		wasiRunnerPath = filepath.Join(wasiDir, "run.js")
		if err := os.WriteFile(wasiRunnerPath, wasiRunner, 0o644); err != nil {
			return CandidateReceipt{}, err
		}
		wasmPath = filepath.Join(wasiDir, "candidate.wasm")
		buildCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		reproduced, err := buildWasiTwice(buildCtx, buildDir, wasmPath)
		if err != nil {
			return reject("wasi build is not reproducible: " + err.Error())
		}
		binary, err := os.ReadFile(wasmPath)
		if err != nil {
			return CandidateReceipt{}, err
		}
		toolchain := goRuntimeVersion()
		receipt.BuildEvidence = &BuildEvidence{
			Language: LanguageWasi, Command: "GOOS=wasip1 GOARCH=wasm go build -trimpath -buildvcs=false",
			Toolchain: toolchain, GOOS: "wasip1", GOARCH: "wasm", CGOEnabled: "0",
			SourceDigest: manifest.SourceDigest, BinaryDigest: digest(binary),
			ReproducedDigest: reproduced, BuildInputDigest: wasiBuildInputDigest(source, toolchain),
			Reproducibility: "path-independent-twice",
		}
		receipt.BinaryDigest = receipt.BuildEvidence.BinaryDigest
	}

	for _, tc := range fixture.Cases {
		if language == LanguagePython || language == LanguageJavaScript {
			current, err := readSnapshotState(snapshotPath)
			if err != nil {
				return reject("source is unreadable during execution: " + err.Error())
			}
			if !current.same(snapshot) {
				receipt.Results = append(receipt.Results, CaseResult{CaseID: tc.ID, Passed: false})
				return reject("source changed during execution before case " + tc.ID)
			}
		}
		var got Result
		var execution *ExecutionEvidence
		var err error
		switch language {
		case LanguagePython, LanguageJavaScript:
			got, execution, err = runScriptCandidate(ctx, scriptPath, snapshotPath, tc.Input, executionOptions)
		case LanguageJava:
			got, execution, err = runProgram(ctx, javaPath, []string{"-cp", javaClassesDir, "Main"}, tc.Input, executionOptions)
		case LanguageWasm:
			got, execution, err = runWasmProgram(ctx, nodePath, []string{wasmExecPath, wasmPath}, tc.Input, executionOptions)
		case LanguageWasi:
			got, execution, err = runWasmProgram(ctx, nodePath, []string{"--no-warnings", wasiRunnerPath, wasmPath}, tc.Input, executionOptions)
		default:
			got, execution, err = runCandidate(ctx, binaryPath, tc.Input, executionOptions)
		}
		if execution != nil && (language == LanguagePython || language == LanguageJavaScript) {
			execution.SourceDigest = actualDigest
		}
		result := CaseResult{CaseID: tc.ID, Want: tc.Want, Got: got, Execution: execution}
		if err != nil {
			result.Passed = false
			receipt.Results = append(receipt.Results, result)
			continue
		}
		result.Passed = reflect.DeepEqual(got, tc.Want)
		receipt.Results = append(receipt.Results, result)
		if language == LanguagePython || language == LanguageJavaScript {
			current, err := readSnapshotState(snapshotPath)
			if err != nil {
				return reject("source is unreadable after case " + tc.ID + ": " + err.Error())
			}
			if !current.same(snapshot) {
				return reject("source changed during execution after case " + tc.ID)
			}
		}
	}
	receipt.Passed = true
	for _, result := range receipt.Results {
		if !result.Passed {
			receipt.Passed = false
			if receipt.RejectReason == "" {
				receipt.RejectReason = "fixture mismatch: " + result.CaseID
			}
		}
	}
	return receipt, nil
}

func candidateLanguage(manifest CandidateManifest) string {
	if manifest.Language == "" {
		return LanguageGo
	}
	return manifest.Language
}

// candidateBinaryPath gives the built executable a platform-appropriate name.
// A Windows binary without .exe is not reliably executable through os/exec.
func candidateBinaryPath(path string) string {
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(path), ".exe") {
		return path + ".exe"
	}
	return path
}

const buildCommand = "go build -trimpath -buildvcs=false"

func buildCandidate(ctx context.Context, dir, binaryPath string) error {
	files, err := goSourceFiles(dir)
	if err != nil {
		return err
	}
	args := append([]string{"build", "-trimpath", "-buildvcs=false", "-o", binaryPath}, files...)
	build := exec.CommandContext(ctx, "go", args...)
	build.Dir = dir
	out, err := build.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}

func goSourceFiles(dir string) ([]string, error) {
	if _, err := os.Stat(filepath.Join(dir, "main.go")); err == nil {
		return []string{"main.go"}, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".go") {
			continue
		}
		files = append(files, entry.Name())
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no Go source files in %s", dir)
	}
	sort.Strings(files)
	return files, nil
}

// buildCandidateTwice builds the same source twice with path-independent flags
// and returns the second digest. The second build uses a copy of the Go sources
// in a different directory, so a path-dependent build is rejected.
func buildCandidateTwice(ctx context.Context, dir, binaryPath string) (string, error) {
	firstDir, err := os.MkdirTemp("", "tei-generator-repro-a-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(firstDir)
	secondDir, err := os.MkdirTemp("", "tei-generator-repro-b-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(secondDir)
	if err := copyGoSources(dir, firstDir); err != nil {
		return "", err
	}
	if err := copyGoSources(dir, secondDir); err != nil {
		return "", err
	}
	if err := buildCandidate(ctx, firstDir, binaryPath); err != nil {
		return "", err
	}
	first, err := os.ReadFile(binaryPath)
	if err != nil {
		return "", err
	}
	secondPath := binaryPath + ".second"
	if err := buildCandidate(ctx, secondDir, secondPath); err != nil {
		return "", err
	}
	second, err := os.ReadFile(secondPath)
	if err != nil {
		return "", err
	}
	firstDigest, secondDigest := digest(first), digest(second)
	if firstDigest != secondDigest {
		return secondDigest, fmt.Errorf("path-dependent build: first %s, second %s", firstDigest, secondDigest)
	}
	return secondDigest, nil
}

func copyGoSources(srcDir, dstDir string) error {
	files, err := goSourceFiles(srcDir)
	if err != nil {
		return err
	}
	for _, name := range files {
		content, err := os.ReadFile(filepath.Join(srcDir, name))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dstDir, name), content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

var (
	goVersionOnce  sync.Once
	goVersionValue string
)

func goRuntimeVersion() string {
	goVersionOnce.Do(func() {
		out, err := exec.Command("go", "version").Output()
		if err == nil {
			goVersionValue = strings.TrimSpace(string(out))
		}
	})
	return goVersionValue
}

var (
	goEnvOnce   sync.Once
	goOSValue   string
	goArchValue string
	cgoValue    string
)

func goBuildEnvironment() (goos, goarch, cgoEnabled string) {
	goEnvOnce.Do(func() {
		out, err := exec.Command("go", "env", "GOOS", "GOARCH", "CGO_ENABLED").Output()
		if err != nil {
			return
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 0 {
			goOSValue = strings.TrimSpace(lines[0])
		}
		if len(lines) > 1 {
			goArchValue = strings.TrimSpace(lines[1])
		}
		if len(lines) > 2 {
			cgoValue = strings.TrimSpace(lines[2])
		}
	})
	return goOSValue, goArchValue, cgoValue
}

func buildInputDigest(source []byte, toolchain, goos, goarch, cgoEnabled, goflags string) string {
	return digest([]byte(strings.Join([]string{
		buildCommand, LanguageGo, digest(source), toolchain, goos, goarch, cgoEnabled, goflags,
	}, "\x00")))
}

func javaBuildInputDigest(source []byte, toolchain string) string {
	return digest([]byte(strings.Join([]string{
		"javac", LanguageJava, digest(source), toolchain,
	}, "\x00")))
}

func buildJavaTwice(ctx context.Context, javacPath, dir, classesDir string) (string, error) {
	firstDir, err := os.MkdirTemp("", "tei-java-repro-a-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(firstDir)
	secondDir, err := os.MkdirTemp("", "tei-java-repro-b-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(secondDir)
	if err := copyJavaSources(dir, firstDir); err != nil {
		return "", err
	}
	if err := copyJavaSources(dir, secondDir); err != nil {
		return "", err
	}
	if err := compileJavaCandidate(ctx, javacPath, firstDir, classesDir); err != nil {
		return "", err
	}
	secondClasses, err := os.MkdirTemp("", "tei-java-classes-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(secondClasses)
	if err := compileJavaCandidate(ctx, javacPath, secondDir, secondClasses); err != nil {
		return "", err
	}
	first, err := os.ReadFile(filepath.Join(classesDir, "Main.class"))
	if err != nil {
		return "", err
	}
	second, err := os.ReadFile(filepath.Join(secondClasses, "Main.class"))
	if err != nil {
		return "", err
	}
	firstDigest, secondDigest := digest(first), digest(second)
	if firstDigest != secondDigest {
		return secondDigest, fmt.Errorf("first %s, second %s", firstDigest, secondDigest)
	}
	return secondDigest, nil
}

func compileJavaCandidate(ctx context.Context, javacPath, dir, classesDir string) error {
	compile := exec.CommandContext(ctx, javacPath, "-d", classesDir, "Main.java")
	compile.Dir = dir
	if out, err := compile.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}

func copyJavaSources(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	copied := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".java") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(srcDir, entry.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dstDir, entry.Name()), content, 0o644); err != nil {
			return err
		}
		copied++
	}
	if copied == 0 {
		return fmt.Errorf("no Java sources in %s", srcDir)
	}
	return nil
}

func wasmBuildInputDigest(source []byte, toolchain string) string {
	return digest([]byte(strings.Join([]string{
		"GOOS=js GOARCH=wasm go build -trimpath -buildvcs=false", LanguageWasm, digest(source), toolchain,
	}, "\x00")))
}

func buildWasmTwice(ctx context.Context, dir, wasmPath string) (string, error) {
	firstDir, err := os.MkdirTemp("", "tei-wasm-repro-a-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(firstDir)
	secondDir, err := os.MkdirTemp("", "tei-wasm-repro-b-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(secondDir)
	if err := copyGoSources(dir, firstDir); err != nil {
		return "", err
	}
	if err := copyGoSources(dir, secondDir); err != nil {
		return "", err
	}
	if err := buildWasmCandidate(ctx, firstDir, wasmPath); err != nil {
		return "", err
	}
	first, err := os.ReadFile(wasmPath)
	if err != nil {
		return "", err
	}
	secondPath := wasmPath + ".second"
	if err := buildWasmCandidate(ctx, secondDir, secondPath); err != nil {
		return "", err
	}
	second, err := os.ReadFile(secondPath)
	if err != nil {
		return "", err
	}
	firstDigest, secondDigest := digest(first), digest(second)
	if firstDigest != secondDigest {
		return secondDigest, fmt.Errorf("first %s, second %s", firstDigest, secondDigest)
	}
	return secondDigest, nil
}

func buildWasmCandidate(ctx context.Context, dir, outputPath string) error {
	build := exec.CommandContext(ctx, "go", "build", "-trimpath", "-buildvcs=false", "-o", outputPath, "main.go")
	build.Dir = dir
	build.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	out, err := build.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}

func resolveWasmRuntime() (nodePath, wasmExecPath, version string, available bool) {
	node, err := exec.LookPath("node")
	if err != nil {
		return "", "", "", false
	}
	out, err := exec.Command(node, "--version").CombinedOutput()
	if err != nil {
		return "", "", "", false
	}
	gorootOut, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return "", "", "", false
	}
	wasmExec := filepath.Join(strings.TrimSpace(string(gorootOut)), "lib", "wasm", "wasm_exec_node.js")
	if _, err := os.Stat(wasmExec); err != nil {
		return "", "", "", false
	}
	return node, wasmExec, strings.TrimSpace(string(out)), true
}

func wasiBuildInputDigest(source []byte, toolchain string) string {
	return digest([]byte(strings.Join([]string{
		"GOOS=wasip1 GOARCH=wasm go build -trimpath -buildvcs=false", LanguageWasi, digest(source), toolchain,
	}, "\x00")))
}

func buildWasiTwice(ctx context.Context, dir, wasmPath string) (string, error) {
	firstDir, err := os.MkdirTemp("", "tei-wasi-repro-a-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(firstDir)
	secondDir, err := os.MkdirTemp("", "tei-wasi-repro-b-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(secondDir)
	if err := copyGoSources(dir, firstDir); err != nil {
		return "", err
	}
	if err := copyGoSources(dir, secondDir); err != nil {
		return "", err
	}
	if err := buildWasiCandidate(ctx, firstDir, wasmPath); err != nil {
		return "", err
	}
	first, err := os.ReadFile(wasmPath)
	if err != nil {
		return "", err
	}
	secondPath := wasmPath + ".second"
	if err := buildWasiCandidate(ctx, secondDir, secondPath); err != nil {
		return "", err
	}
	second, err := os.ReadFile(secondPath)
	if err != nil {
		return "", err
	}
	firstDigest, secondDigest := digest(first), digest(second)
	if firstDigest != secondDigest {
		return secondDigest, fmt.Errorf("first %s, second %s", firstDigest, secondDigest)
	}
	return secondDigest, nil
}

func buildWasiCandidate(ctx context.Context, dir, outputPath string) error {
	build := exec.CommandContext(ctx, "go", "build", "-trimpath", "-buildvcs=false", "-o", outputPath, "main.go")
	build.Dir = dir
	build.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	out, err := build.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}

func resolveWasiRuntime() (nodePath, version string, available bool) {
	node, err := exec.LookPath("node")
	if err != nil {
		return "", "", false
	}
	out, err := exec.Command(node, "--version").CombinedOutput()
	if err != nil {
		return "", "", false
	}
	return node, strings.TrimSpace(string(out)), true
}

// ResolveWasiRuntime exposes the same Node discovery for target-profile probes.
func ResolveWasiRuntime() (nodePath, version string, available bool) {
	return resolveWasiRuntime()
}

// WasiRunnerSource returns the embedded Node WASI preview1 runner.
func WasiRunnerSource() []byte {
	return wasiRunner
}

// ResolveScriptRuntime uses the same interpreter discovery for selection and
// target-profile probes. Availability is evidence of a runnable interpreter,
// not permission to execute a candidate or a limit on its resource use.
func ResolveScriptRuntime(language string) (string, *RuntimeEvidence) {
	var executable, version string
	var available bool
	command := ""
	switch language {
	case LanguagePython:
		executable, version, available = findPython()
		command = "python <source>"
	case LanguageJavaScript:
		executable, version, available = findJavaScript()
		command = "node <source>"
	}
	if executable != "" {
		command = executable + " <source>"
	}
	return executable, &RuntimeEvidence{Language: language, Command: command, Version: version, Available: available}
}

func findPython() (path, version string, available bool) {
	for _, name := range []string{"python3", "python"} {
		found, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		// Windows may provide a python3.exe app-execution alias that is not a
		// working interpreter. Only accept a command that can report its version.
		out, err := exec.Command(found, "--version").CombinedOutput()
		if err != nil {
			continue
		}
		return found, strings.TrimSpace(string(out)), true
	}
	return "", "", false
}

func findJavaScript() (path, version string, available bool) {
	found, err := exec.LookPath("node")
	if err != nil {
		return "", "", false
	}
	out, err := exec.Command(found, "--version").CombinedOutput()
	if err != nil {
		return "", "", false
	}
	return found, strings.TrimSpace(string(out)), true
}

func findJava() (javacPath, javaPath, version string, available bool) {
	javac, err := exec.LookPath("javac")
	if err != nil {
		return "", "", "", false
	}
	java, err := exec.LookPath("java")
	if err != nil {
		return "", "", "", false
	}
	out, err := exec.Command(javac, "-version").CombinedOutput()
	if err != nil {
		return "", "", "", false
	}
	return javac, java, strings.TrimSpace(string(out)), true
}

// candidateSourcePath resolves the declared manifest source inside the
// candidate directory. It refuses an absolute path, a path with "..", and a
// symlink whose resolved target is outside the directory. The declaration is
// never allowed to read a file beside the candidate.
func candidateSourcePath(dir, declared string) (string, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	relative := filepath.FromSlash(declared)
	if filepath.IsAbs(relative) {
		return "", fmt.Errorf("source %q is absolute", declared)
	}
	cleaned := filepath.Clean(relative)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("source %q escapes the candidate directory", declared)
	}
	path := filepath.Join(root, cleaned)
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	relativeToRoot, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil {
		return "", err
	}
	if relativeToRoot == ".." || strings.HasPrefix(relativeToRoot, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("source %q resolves outside the candidate directory", declared)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("source %q is not a regular file", declared)
	}
	return path, nil
}

// stageVerifiedGoSources copies the Go build inputs into a selector-private
// directory and writes the declared source from the bytes that were checked
// against the manifest digest, so the build cannot read a declared source that
// differs from the verified one. Both reproducibility builds copy from this
// directory. Other Go files of a multi-file candidate are read once here; the
// manifest carries no digest for them and they are not verified.
func stageVerifiedGoSources(dir, declared string, verified []byte) (string, error) {
	staged, err := os.MkdirTemp("", "tei-generator-verified-*")
	if err != nil {
		return "", err
	}
	if err := copyGoSources(dir, staged); err != nil {
		os.RemoveAll(staged)
		return "", err
	}
	name := filepath.Clean(filepath.FromSlash(declared))
	if err := os.WriteFile(filepath.Join(staged, name), verified, 0o644); err != nil {
		os.RemoveAll(staged)
		return "", err
	}
	return staged, nil
}

// snapshotState is what the selector can observe about the execution snapshot
// of a script without help from the operating system: its bytes and the
// identity, size and modification time of the file. A rewrite that restores
// the bytes (D1 -> D2 -> D1) leaves the digest unchanged but not the
// modification time, and a replaced file is a different file.
//
// This is detection, not isolation. A process with the same OS privileges that
// also restores the modification time is not detected, and nothing here
// observes the bytes the interpreter actually read, the modules it imported or
// its environment. Those belong to OS isolation (#2515), not to this check.
type snapshotState struct {
	digest string
	info   os.FileInfo
}

func readSnapshotState(path string) (snapshotState, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return snapshotState{}, err
	}
	if !before.Mode().IsRegular() {
		return snapshotState{}, fmt.Errorf("execution snapshot is not a regular file")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return snapshotState{}, err
	}
	return snapshotState{digest: digest(raw), info: before}, nil
}

func (s snapshotState) same(other snapshotState) bool {
	return s.digest == other.digest &&
		os.SameFile(s.info, other.info) &&
		s.info.Size() == other.info.Size() &&
		s.info.ModTime().Equal(other.info.ModTime())
}

func runScriptCandidate(ctx context.Context, interpreter, sourcePath string, input json.RawMessage, options ExecutionOptions) (Result, *ExecutionEvidence, error) {
	absolute, err := filepath.Abs(sourcePath)
	if err != nil {
		return Result{}, nil, err
	}
	return runProgram(ctx, interpreter, []string{absolute}, input, options)
}

func runCandidate(ctx context.Context, binary string, input json.RawMessage, options ExecutionOptions) (Result, *ExecutionEvidence, error) {
	return runProgram(ctx, binary, nil, input, options)
}

// minimalWasmEnvironment returns the variables a wasm runtime needs on this
// host and nothing else. Everything else is dropped on purpose: the Go wasm
// runtime copies the environment into the instance, and a CI runner's block
// overflows that buffer (#3971).
func minimalWasmEnvironment() []string {
	names := []string{"SystemRoot", "TEMP", "TMP", "HOME", "PATH"}
	env := make([]string, 0, len(names))
	for _, name := range names {
		if value, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+value)
		}
	}
	return env
}

// runWasmProgram runs a wasm candidate with a minimal environment. The Go wasm
// runtime copies the whole environment into the instance, and on a Windows
// runner the inherited block overflows that buffer ("total length of command
// line and environment variables exceeds limit"), which makes every case look
// like an empty result (#3971). A candidate is hermetic, so it gets the few
// variables the platform needs rather than the host's settings.
func runWasmProgram(ctx context.Context, name string, args []string, input json.RawMessage, options ExecutionOptions) (Result, *ExecutionEvidence, error) {
	result, evidence, err := runProgram(ctx, name, args, input, options.withMinimalEnvironment())
	return result, evidence, err
}
func runProgram(ctx context.Context, name string, args []string, input json.RawMessage, options ExecutionOptions) (Result, *ExecutionEvidence, error) {
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := options.Command(runCtx, name, args...)
	cmd.Stdin = bytes.NewReader(input)
	if options.MinimalEnvironment {
		cmd.Env = minimalWasmEnvironment()
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	started := time.Now()
	err := cmd.Run()
	elapsed := time.Since(started)
	evidence := &ExecutionEvidence{
		Measurement: MeasureProcess(cmd.ProcessState, elapsed),
		ExitCode:    0, StdoutDigest: digest(stdout.Bytes()), StderrDigest: digest(stderr.Bytes()),
	}
	// TEI_DEBUG_CANDIDATE_OUTPUT makes a run print what the candidate wrote.
	// It exists to explain a candidate that exits 0 and produces an empty
	// result, where the receipt only says that the fields were empty (#3971).
	if os.Getenv("TEI_DEBUG_CANDIDATE_OUTPUT") != "" {
		fmt.Fprintf(os.Stderr, "candidate %q %v: stdout=%q stderr=%q\n",
			name, args, truncateForLog(stdout.Bytes(), 2000), truncateForLog(stderr.Bytes(), 2000))
	}
	if err != nil {
		evidence.ExitCode = -1
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			evidence.ExitCode = exit.ExitCode()
		}
		if runCtx.Err() == context.DeadlineExceeded {
			evidence.TimedOut = true
		}
		return Result{}, evidence, fmt.Errorf("candidate exited %d: %s", evidence.ExitCode, stderr.String())
	}
	// TEI_DEBUG_CANDIDATE_OUTPUT makes a run print what the candidate wrote.
	// It exists to explain a candidate that exits 0 and produces an empty
	// result, where the receipt only says that the fields were empty (#3971).
	if os.Getenv("TEI_DEBUG_CANDIDATE_OUTPUT") != "" {
		fmt.Fprintf(os.Stderr, "candidate %q %v: stdout=%q stderr=%q\n",
			name, args, truncateForLog(stdout.Bytes(), 2000), truncateForLog(stderr.Bytes(), 2000))
	}
	var result Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return Result{}, evidence, fmt.Errorf("candidate output is not a result: %w: %q", err, stdout.Bytes())
	}
	return result, evidence, nil
}

// truncateForLog keeps a diagnostic line readable without dropping the start of
// the output, which is where a decoder error usually shows itself.
func truncateForLog(raw []byte, limit int) string {
	if len(raw) <= limit {
		return string(raw)
	}
	return string(raw[:limit]) + "...(truncated)"
}
