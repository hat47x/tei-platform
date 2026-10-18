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
	"path/filepath"
	"reflect"
	"sort"
	"strings"
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

type CandidateManifest struct {
	Schema       string               `json:"schema"`
	ID           string               `json:"id"`
	Origin       string               `json:"origin"`
	Source       string               `json:"source"`
	SourceDigest string               `json:"sourceDigest"`
	Note         string               `json:"note,omitempty"`
	Generator    *GeneratorProvenance `json:"generator,omitempty"`
}

type CaseResult struct {
	CaseID string `json:"caseId"`
	Passed bool   `json:"passed"`
	Got    Result `json:"got"`
	Want   Result `json:"want"`
}

type CandidateReceipt struct {
	ID           string               `json:"id"`
	Origin       string               `json:"origin"`
	SourceDigest string               `json:"sourceDigest"`
	BinaryDigest string               `json:"binaryDigest,omitempty"`
	Generator    *GeneratorProvenance `json:"generator,omitempty"`
	Results      []CaseResult         `json:"results,omitempty"`
	Passed       bool                 `json:"passed"`
	RejectReason string               `json:"rejectReason,omitempty"`
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
	var directories []string
	for _, entry := range entries {
		if entry.IsDir() {
			directories = append(directories, entry.Name())
		}
	}
	sort.Strings(directories)
	for _, name := range directories {
		candidate, err := evaluateCandidate(ctx, filepath.Join(candidatesDir, name), fixture, contractPath, digest(contractRaw))
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

func evaluateCandidate(ctx context.Context, dir string, fixture Fixture, contractPath, contractDigest string) (CandidateReceipt, error) {
	var manifest CandidateManifest
	if _, err := readJSON(filepath.Join(dir, "candidate.json"), &manifest); err != nil {
		return CandidateReceipt{}, err
	}
	receipt := CandidateReceipt{ID: manifest.ID, Origin: manifest.Origin, SourceDigest: manifest.SourceDigest, Generator: manifest.Generator}
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
	if manifest.Generator != nil {
		if manifest.Generator.ID == "" || manifest.Generator.Version == "" || manifest.Generator.Strategy == "" {
			return reject("generator provenance is incomplete")
		}
		if manifest.Generator.ContractDigest != contractDigest {
			return reject("generator contract digest mismatch")
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
	if manifest.Generator != nil {
		rendered, err := renderCandidate(contractPath, manifest.Generator.Strategy)
		if err != nil {
			return reject("generator provenance cannot be reproduced: " + err.Error())
		}
		if !bytes.Equal(rendered.Source, source) {
			return reject("source does not match the declared generator and contract")
		}
		if rendered.Manifest.Generator == nil || *rendered.Manifest.Generator != *manifest.Generator {
			return reject("generator provenance does not match the current contract and template")
		}
	}
	binaryDir, err := os.MkdirTemp("", "tei-generator-candidate-*")
	if err != nil {
		return CandidateReceipt{}, err
	}
	defer os.RemoveAll(binaryDir)
	binaryPath := filepath.Join(binaryDir, manifest.ID)
	buildCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := buildCandidate(buildCtx, dir, binaryPath); err != nil {
		return reject("build failed: " + err.Error())
	}
	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		return CandidateReceipt{}, err
	}
	receipt.BinaryDigest = digest(binary)

	for _, tc := range fixture.Cases {
		got, err := runCandidate(ctx, binaryPath, tc.Input)
		result := CaseResult{CaseID: tc.ID, Want: tc.Want, Got: got}
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

func buildCandidate(ctx context.Context, dir, binaryPath string) error {
	build := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, ".")
	build.Dir = dir
	out, err := build.CombinedOutput()
	if err == nil {
		return nil
	}
	// Generated candidates may live outside a Go module. Their sources use the
	// standard library only, so retry in GOPATH mode instead of requiring a
	// go.mod in a temporary directory.
	if strings.Contains(string(out), "go.mod file not found") {
		fallback := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "main.go")
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

func runCandidate(ctx context.Context, binary string, input json.RawMessage) (Result, error) {
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, binary)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return Result{}, fmt.Errorf("candidate exited %d: %s", exit.ExitCode(), string(exit.Stderr))
		}
		return Result{}, err
	}
	var result Result
	if err := json.Unmarshal(out, &result); err != nil {
		return Result{}, fmt.Errorf("candidate output is not a result: %w: %q", err, out)
	}
	return result, nil
}
