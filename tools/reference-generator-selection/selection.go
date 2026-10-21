// Package generatorselection compares generator candidates for one accepted
// reference contract and records what each candidate actually produced. It
// selects by a fixed fixture; origin, self-report, and build success do not
// grant authority.
package generatorselection

import (
	"bytes"
	"context"
	"crypto/sha256"
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
	Schema   string      `json:"schema"`
	ID       string      `json:"id"`
	Version  string      `json:"version"`
	Status   string      `json:"status"`
	Note     string      `json:"note,omitempty"`
	Inputs   []InputSpec `json:"inputs"`
	Output   TypeSpec    `json:"output"`
	Rounding Rounding    `json:"rounding"`
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
	ExitCode     int    `json:"exitCode"`
	StdoutDigest string `json:"stdoutDigest"`
	StderrDigest string `json:"stderrDigest"`
	TimedOut     bool   `json:"timedOut,omitempty"`
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
	return selectWithRegistryAndFilter(ctx, workspaceDir, contractPath, fixturePath, candidatesDir, registryPath, allowedIDs)
}

func selectWithRegistry(ctx context.Context, workspaceDir, contractPath, fixturePath, candidatesDir, registryPath string) (Receipt, error) {
	return selectWithRegistryAndFilter(ctx, workspaceDir, contractPath, fixturePath, candidatesDir, registryPath, nil)
}

func selectWithRegistryAndFilter(ctx context.Context, workspaceDir, contractPath, fixturePath, candidatesDir, registryPath string, allowedIDs []string) (Receipt, error) {
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
		Schema: ReceiptSchema, ContractID: contract.ID, ContractVersion: contract.Version,
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
		candidate, err := evaluateCandidate(ctx, filepath.Join(candidatesDir, name), fixture, contractPath, digest(contractRaw), registry, allowed)
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

func evaluateCandidate(ctx context.Context, dir string, fixture Fixture, contractPath, contractDigest string, registry *Registry, allowed map[string]bool) (CandidateReceipt, error) {
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
	if language != LanguageGo && language != LanguagePython && language != LanguageJavaScript && language != LanguageJava {
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
	sourcePath := filepath.Join(dir, filepath.FromSlash(manifest.Source))
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return reject("source is unreadable: " + err.Error())
	}
	actualDigest := digest(source)
	if actualDigest != manifest.SourceDigest {
		return reject("source digest mismatch: declared " + manifest.SourceDigest + ", actual " + actualDigest)
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
	var binaryPath, scriptPath, javaPath, javacPath, javaClassesDir string
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
		reproduced, err := buildCandidateTwice(buildCtx, dir, binaryPath)
		if err != nil {
			receipt.BuildEvidence.Reproducibility = "mismatch"
			return reject("build is not reproducible: " + err.Error())
		}
		receipt.BuildEvidence.ReproducedDigest = reproduced
		receipt.BuildEvidence.Reproducibility = "same-digest-twice"
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
		compile := exec.CommandContext(compileCtx, javacPath, "-d", javaClassesDir, sourcePath)
		if out, err := compile.CombinedOutput(); err != nil {
			return reject("java compile failed: " + err.Error() + ": " + string(out))
		}
		classBytes, err := os.ReadFile(filepath.Join(javaClassesDir, "Main.class"))
		if err != nil {
			return reject("compiled class is unreadable: " + err.Error())
		}
		receipt.BuildEvidence = &BuildEvidence{
			Language: LanguageJava, Command: "javac", Toolchain: version,
			SourceDigest: manifest.SourceDigest, BinaryDigest: digest(classBytes),
			BuildInputDigest: javaBuildInputDigest(source, version),
			Reproducibility:  "not-tested",
		}
		receipt.BinaryDigest = receipt.BuildEvidence.BinaryDigest
	}

	for _, tc := range fixture.Cases {
		var got Result
		var execution *ExecutionEvidence
		var err error
		switch language {
		case LanguagePython, LanguageJavaScript:
			got, execution, err = runScriptCandidate(ctx, scriptPath, sourcePath, tc.Input)
		case LanguageJava:
			got, execution, err = runProgram(ctx, javaPath, []string{"-cp", javaClassesDir, "Main"}, tc.Input)
		default:
			got, execution, err = runCandidate(ctx, binaryPath, tc.Input)
		}
		result := CaseResult{CaseID: tc.ID, Want: tc.Want, Got: got, Execution: execution}
		if err != nil {
			result.Passed = false
			receipt.Results = append(receipt.Results, result)
			continue
		}
		result.Passed = reflect.DeepEqual(got, tc.Want)
		receipt.Results = append(receipt.Results, result)
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
	build := exec.CommandContext(ctx, "go", "build", "-trimpath", "-buildvcs=false", "-o", binaryPath, ".")
	build.Dir = dir
	out, err := build.CombinedOutput()
	if err == nil {
		return nil
	}
	// Generated candidates may live outside a Go module. Their sources use the
	// standard library only, so retry in GOPATH mode instead of requiring a
	// go.mod in a temporary directory.
	if strings.Contains(string(out), "go.mod file not found") {
		fallback := exec.CommandContext(ctx, "go", "build", "-trimpath", "-buildvcs=false", "-o", binaryPath, "main.go")
		fallback.Dir = dir
		fallback.Env = append(os.Environ(), "GO111MODULE=off")
		if fallbackOut, fallbackErr := fallback.CombinedOutput(); fallbackErr == nil {
			return nil
		} else {
			return fmt.Errorf("%v: %s", fallbackErr, fallbackOut)
		}
	}
	return fmt.Errorf("%v: %s", err, out)
}

// buildCandidateTwice builds the same source twice with path-independent flags
// and returns the second digest. A mismatch is an error, not a warning.
func buildCandidateTwice(ctx context.Context, dir, binaryPath string) (string, error) {
	if err := buildCandidate(ctx, dir, binaryPath); err != nil {
		return "", err
	}
	first, err := os.ReadFile(binaryPath)
	if err != nil {
		return "", err
	}
	secondPath := binaryPath + ".second"
	if err := buildCandidate(ctx, dir, secondPath); err != nil {
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

func runScriptCandidate(ctx context.Context, interpreter, sourcePath string, input json.RawMessage) (Result, *ExecutionEvidence, error) {
	absolute, err := filepath.Abs(sourcePath)
	if err != nil {
		return Result{}, nil, err
	}
	return runProgram(ctx, interpreter, []string{absolute}, input)
}

func runCandidate(ctx context.Context, binary string, input json.RawMessage) (Result, *ExecutionEvidence, error) {
	return runProgram(ctx, binary, nil, input)
}

func runProgram(ctx context.Context, name string, args []string, input json.RawMessage) (Result, *ExecutionEvidence, error) {
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, name, args...)
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	evidence := &ExecutionEvidence{
		ExitCode: 0, StdoutDigest: digest(stdout.Bytes()), StderrDigest: digest(stderr.Bytes()),
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
	var result Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return Result{}, evidence, fmt.Errorf("candidate output is not a result: %w: %q", err, stdout.Bytes())
	}
	return result, evidence, nil
}
