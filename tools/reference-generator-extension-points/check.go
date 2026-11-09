// Package generatorextensionpoints is a read-only reference for #2090.
//
// It declares the extension points a generator exposes, keeps a case extension
// outside the generated artifact, and checks three things: regeneration keeps
// the case extension, a changed extension-point contract is detected, and a
// direct edit outside the extension points is reported before regeneration
// overwrites it. It never writes, regenerates, or applies anything.
package generatorextensionpoints

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

const (
	InputSchema  = "tei.reference.generator-extension-points-input/v0"
	ReportSchema = "tei.reference.generator-extension-points-report/v0"

	StatusPassed     = "passed"
	StatusConflict   = "conflict"
	StatusUnresolved = "unresolved"

	ExtensionBound    = "bound"
	ExtensionConflict = "conflict"

	FileDirectEdit       = "direct-edit"
	FileMissing          = "missing"
	FileExtensionChanged = "extension-changed"
)

// ExtensionPoint is one place a generated artifact lets a case add code. Order
// is the call order; Inputs and Outputs are the declared contract.
type ExtensionPoint struct {
	ID      string   `json:"id"`
	Version string   `json:"version"`
	Order   int      `json:"order"`
	Inputs  []string `json:"inputs"`
	Outputs []string `json:"outputs"`
	Default string   `json:"default"`
}

// ArtifactFile is one file the generator owns.
type ArtifactFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Outcome records where the presence of an extension changes a business
// outcome. It is evidence, not an acceptance decision.
type Outcome struct {
	Point         string `json:"point"`
	Outcome       string `json:"outcome"`
	Default       string `json:"default"`
	WithExtension string `json:"withExtension"`
}

// Generator is the reused generator and the artifact it produces.
type Generator struct {
	ID              string           `json:"id"`
	Version         string           `json:"version"`
	ExtensionPoints []ExtensionPoint `json:"extensionPoints"`
	Artifact        []ArtifactFile   `json:"artifact"`
	Outcomes        []Outcome        `json:"outcomes,omitempty"`
}

// Contract is what the case fixed when it bound an extension.
type Contract struct {
	Version string   `json:"version"`
	Order   int      `json:"order"`
	Inputs  []string `json:"inputs"`
	Outputs []string `json:"outputs"`
}

// CaseExtension is case-owned code kept outside the generated artifact.
type CaseExtension struct {
	Point    string   `json:"point"`
	Path     string   `json:"path"`
	Content  string   `json:"content"`
	Contract Contract `json:"contract"`
}

// Input is the model the check reads.
type Input struct {
	Schema         string          `json:"schema"`
	Generator      Generator       `json:"generator"`
	CaseExtensions []CaseExtension `json:"caseExtensions"`
	WorkspaceFiles []ArtifactFile  `json:"workspaceFiles"`
}

// PointResult is one declared extension point.
type PointResult struct {
	ID      string   `json:"id"`
	Version string   `json:"version"`
	Order   int      `json:"order"`
	Inputs  []string `json:"inputs"`
	Outputs []string `json:"outputs"`
	Default string   `json:"default"`
}

// ExtensionResult is the binding check for one case extension.
type ExtensionResult struct {
	Point   string   `json:"point"`
	Path    string   `json:"path"`
	Status  string   `json:"status"`
	Reasons []string `json:"reasons,omitempty"`
}

// RegenerationResult records that repeated regeneration kept the artifact and
// the case extension apart.
type RegenerationResult struct {
	Rounds                    int      `json:"rounds"`
	ArtifactStable            bool     `json:"artifactStable"`
	ExtensionsStable          bool     `json:"extensionsStable"`
	ExtensionsOutsideArtifact bool     `json:"extensionsOutsideArtifact"`
	ArtifactPaths             []string `json:"artifactPaths"`
	ExtensionPaths            []string `json:"extensionPaths"`
}

// DirectEdit is a workspace file that differs from the generator output.
type DirectEdit struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// OutcomeResult is the evidence for one business outcome.
type OutcomeResult struct {
	Point            string `json:"point"`
	Outcome          string `json:"outcome"`
	Default          string `json:"default"`
	WithExtension    string `json:"withExtension"`
	ExtensionPresent bool   `json:"extensionPresent"`
}

// Report is the check result.
type Report struct {
	Schema       string             `json:"schema"`
	Status       string             `json:"status"`
	GeneratorID  string             `json:"generatorId"`
	GeneratorVer string             `json:"generatorVersion"`
	Points       []PointResult      `json:"points"`
	Extensions   []ExtensionResult  `json:"extensions"`
	Regeneration RegenerationResult `json:"regeneration"`
	DirectEdits  []DirectEdit       `json:"directEdits,omitempty"`
	Outcomes     []OutcomeResult    `json:"outcomes,omitempty"`
	Problems     []string           `json:"problems,omitempty"`
	Note         string             `json:"note"`
}

// LoadInput reads an input file.
func LoadInput(path string) (Input, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Input{}, err
	}
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		return Input{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return input, nil
}

// Check runs the declaration check, the regeneration check, the direct-edit
// check, and the outcome evidence. It does not change the input.
func Check(input Input) Report {
	report := Report{
		Schema:       ReportSchema,
		GeneratorID:  input.Generator.ID,
		GeneratorVer: input.Generator.Version,
		Note:         "読取り専用の検査です。生成・再生成・適用は行いません。拡張点の外の変更は再生成で上書きされる前に報告します。",
	}

	var problems []string
	if input.Schema != InputSchema {
		problems = append(problems, fmt.Sprintf("schema is %q, want %q", input.Schema, InputSchema))
	}
	if input.Generator.ID == "" {
		problems = append(problems, "generator id is required")
	}
	if input.Generator.Version == "" {
		problems = append(problems, "generator version is required")
	}

	points := make(map[string]ExtensionPoint, len(input.Generator.ExtensionPoints))
	orders := make(map[int]string, len(input.Generator.ExtensionPoints))
	for _, point := range input.Generator.ExtensionPoints {
		if point.ID == "" {
			problems = append(problems, "extension point with empty id")
			continue
		}
		if _, exists := points[point.ID]; exists {
			problems = append(problems, fmt.Sprintf("duplicate extension point id %q", point.ID))
			continue
		}
		if point.Version == "" {
			problems = append(problems, fmt.Sprintf("extension point %q has no version", point.ID))
		}
		if len(point.Inputs) == 0 || len(point.Outputs) == 0 {
			problems = append(problems, fmt.Sprintf("extension point %q must declare inputs and outputs", point.ID))
		}
		if existing, exists := orders[point.Order]; exists {
			problems = append(problems, fmt.Sprintf("extension points %q and %q share order %d", existing, point.ID, point.Order))
		}
		orders[point.Order] = point.ID
		points[point.ID] = point
	}

	artifact := make(map[string]string, len(input.Generator.Artifact))
	for _, file := range input.Generator.Artifact {
		if file.Path == "" {
			problems = append(problems, "artifact file with empty path")
			continue
		}
		if _, exists := artifact[file.Path]; exists {
			problems = append(problems, fmt.Sprintf("duplicate artifact path %q", file.Path))
			continue
		}
		artifact[file.Path] = file.Content
	}

	extensionPaths := make(map[string]bool, len(input.CaseExtensions))
	var extensions []ExtensionResult
	for _, extension := range input.CaseExtensions {
		result := ExtensionResult{Point: extension.Point, Path: extension.Path, Status: ExtensionBound}
		point, exists := points[extension.Point]
		if !exists {
			result.Status = ExtensionConflict
			result.Reasons = append(result.Reasons, "extension-point-not-declared")
			extensions = append(extensions, result)
			continue
		}
		if extension.Path == "" {
			problems = append(problems, fmt.Sprintf("case extension for %q has no path", extension.Point))
			continue
		}
		if _, exists := artifact[extension.Path]; exists {
			result.Status = ExtensionConflict
			result.Reasons = append(result.Reasons, "extension-inside-generated-artifact")
		}
		if existing, exists := extensionPaths[extension.Path]; exists && existing {
			problems = append(problems, fmt.Sprintf("duplicate case extension path %q", extension.Path))
		}
		extensionPaths[extension.Path] = true
		result.Reasons = append(result.Reasons, contractDifferences(point, extension.Contract)...)
		if len(result.Reasons) > 0 {
			result.Status = ExtensionConflict
		}
		extensions = append(extensions, result)
	}
	sort.Slice(extensions, func(i, j int) bool {
		if extensions[i].Point == extensions[j].Point {
			return extensions[i].Path < extensions[j].Path
		}
		return extensions[i].Point < extensions[j].Point
	})
	report.Extensions = extensions

	pointResults := make([]PointResult, 0, len(points))
	for _, point := range points {
		pointResults = append(pointResults, PointResult{
			ID: point.ID, Version: point.Version, Order: point.Order,
			Inputs: point.Inputs, Outputs: point.Outputs, Default: point.Default,
		})
	}
	sort.Slice(pointResults, func(i, j int) bool {
		if pointResults[i].Order == pointResults[j].Order {
			return pointResults[i].ID < pointResults[j].ID
		}
		return pointResults[i].Order < pointResults[j].Order
	})
	report.Points = pointResults

	report.Regeneration = regeneration(input)
	report.DirectEdits = directEdits(input, artifact, extensionPaths)
	report.Outcomes = outcomeEvidence(input)

	if len(problems) > 0 {
		report.Problems = problems
		report.Status = StatusUnresolved
		return report
	}
	report.Status = StatusPassed
	for _, extension := range report.Extensions {
		if extension.Status == ExtensionConflict {
			report.Status = StatusConflict
		}
	}
	for _, edit := range report.DirectEdits {
		if edit.Status == FileDirectEdit || edit.Status == FileMissing {
			report.Status = StatusConflict
		}
	}
	return report
}

func contractDifferences(point ExtensionPoint, contract Contract) []string {
	var reasons []string
	if contract.Version != point.Version {
		reasons = append(reasons, fmt.Sprintf("contract-version-changed: %q != %q", contract.Version, point.Version))
	}
	if contract.Order != point.Order {
		reasons = append(reasons, fmt.Sprintf("contract-order-changed: %d != %d", contract.Order, point.Order))
	}
	if !sameStrings(contract.Inputs, point.Inputs) {
		reasons = append(reasons, fmt.Sprintf("contract-inputs-changed: %v != %v", contract.Inputs, point.Inputs))
	}
	if !sameStrings(contract.Outputs, point.Outputs) {
		reasons = append(reasons, fmt.Sprintf("contract-outputs-changed: %v != %v", contract.Outputs, point.Outputs))
	}
	return reasons
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// regeneration produces the generator output three times and checks that the
// artifact and the case extensions stay separate and byte-identical.
func regeneration(input Input) RegenerationResult {
	const rounds = 3
	result := RegenerationResult{
		Rounds:                    rounds,
		ArtifactStable:            true,
		ExtensionsStable:          true,
		ExtensionsOutsideArtifact: true,
	}
	var firstArtifact, firstExtensions map[string]string
	for round := 0; round < rounds; round++ {
		artifact, extensions := regenerate(input)
		if firstArtifact == nil {
			firstArtifact = artifact
			firstExtensions = extensions
			continue
		}
		if !sameFileMaps(firstArtifact, artifact) {
			result.ArtifactStable = false
		}
		if !sameFileMaps(firstExtensions, extensions) {
			result.ExtensionsStable = false
		}
	}
	for _, file := range input.Generator.Artifact {
		result.ArtifactPaths = append(result.ArtifactPaths, file.Path)
	}
	for _, extension := range input.CaseExtensions {
		result.ExtensionPaths = append(result.ExtensionPaths, extension.Path)
		if _, exists := firstArtifact[extension.Path]; exists {
			result.ExtensionsOutsideArtifact = false
		}
	}
	sort.Strings(result.ArtifactPaths)
	sort.Strings(result.ExtensionPaths)
	return result
}

func regenerate(input Input) (artifact, extensions map[string]string) {
	artifact = make(map[string]string, len(input.Generator.Artifact))
	for _, file := range input.Generator.Artifact {
		artifact[file.Path] = file.Content
	}
	extensions = make(map[string]string, len(input.CaseExtensions))
	for _, extension := range input.CaseExtensions {
		extensions[extension.Path] = extension.Content
	}
	return artifact, extensions
}

func sameFileMaps(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for path, content := range left {
		if right[path] != content {
			return false
		}
	}
	return true
}

func fileContent(files []ArtifactFile, path string) (string, bool) {
	for _, file := range files {
		if file.Path == path {
			return file.Content, true
		}
	}
	return "", false
}

func directEdits(input Input, artifact map[string]string, extensionPaths map[string]bool) []DirectEdit {
	var edits []DirectEdit
	seen := map[string]bool{}
	for _, file := range input.WorkspaceFiles {
		if seen[file.Path] {
			edits = append(edits, DirectEdit{Path: file.Path, Status: FileDirectEdit, Detail: "duplicate workspace path"})
			continue
		}
		seen[file.Path] = true
		if expected, exists := artifact[file.Path]; exists {
			if expected != file.Content {
				edits = append(edits, DirectEdit{Path: file.Path, Status: FileDirectEdit, Detail: "generated file differs from the generator output"})
			}
			continue
		}
		if extensionPaths[file.Path] {
			extensionContent := ""
			for _, extension := range input.CaseExtensions {
				if extension.Path == file.Path {
					extensionContent = extension.Content
					break
				}
			}
			if extensionContent != file.Content {
				edits = append(edits, DirectEdit{Path: file.Path, Status: FileExtensionChanged, Detail: "case extension changed; it is outside the artifact"})
			}
			continue
		}
		// Unknown paths are not part of this comparison.
	}
	for path := range artifact {
		if !seen[path] {
			edits = append(edits, DirectEdit{Path: path, Status: FileMissing, Detail: "generated file is absent from the workspace"})
		}
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].Path < edits[j].Path })
	return edits
}

func outcomeEvidence(input Input) []OutcomeResult {
	usedPoints := map[string]bool{}
	for _, extension := range input.CaseExtensions {
		usedPoints[extension.Point] = true
	}
	var outcomes []OutcomeResult
	for _, outcome := range input.Generator.Outcomes {
		outcomes = append(outcomes, OutcomeResult{
			Point:            outcome.Point,
			Outcome:          outcome.Outcome,
			Default:          outcome.Default,
			WithExtension:    outcome.WithExtension,
			ExtensionPresent: usedPoints[outcome.Point],
		})
	}
	sort.Slice(outcomes, func(i, j int) bool {
		if outcomes[i].Point == outcomes[j].Point {
			return outcomes[i].Outcome < outcomes[j].Outcome
		}
		return outcomes[i].Point < outcomes[j].Point
	})
	return outcomes
}

// DefaultEqualsNoExtension reports whether a generator run with no case
// extension returns exactly the declared default artifact.
func DefaultEqualsNoExtension(input Input) bool {
	empty := input
	empty.CaseExtensions = nil
	artifact, extensions := regenerate(empty)
	if len(extensions) != 0 || len(artifact) != len(input.Generator.Artifact) {
		return false
	}
	for _, file := range input.Generator.Artifact {
		if artifact[file.Path] != file.Content {
			return false
		}
	}
	return true
}

// ArtifactPaths returns the generator-owned paths in a stable order.
func ArtifactPaths(input Input) []string {
	paths := make([]string, 0, len(input.Generator.Artifact))
	for _, file := range input.Generator.Artifact {
		paths = append(paths, file.Path)
	}
	sort.Strings(paths)
	return paths
}
