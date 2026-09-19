package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"tei/tools/internal/repositoryevidence"
)

const evidenceNature = "first-web-target-architecture-conformance"

type checkEvidence struct {
	Name    string   `json:"name"`
	Passed  bool     `json:"passed"`
	Details []string `json:"details,omitempty"`
}

type evidence struct {
	EvidenceNature             string                     `json:"evidenceNature"`
	HandoverBaselineID         string                     `json:"handoverBaselineId"`
	BaselineRepositoryRevision string                     `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string                     `json:"targetBuildRevision"`
	WorkingTreeClean           bool                       `json:"workingTreeClean"`
	ReproducibleRevision       bool                       `json:"reproducibleRevision"`
	PostRunRepository          repositoryevidence.PostRun `json:"postRunRepository"`
	CanonicalWorldInvariant    checkEvidence              `json:"canonicalWorldInvariant"`
	WebTargetAuthorityBoundary checkEvidence              `json:"webTargetAuthorityBoundary"`
	SharedCommandBoundary      checkEvidence              `json:"sharedCommandBoundary"`
	AuthorityLimits            []string                   `json:"authorityLimits"`
	Unresolved                 []string                   `json:"unresolved"`
}

var nativeSQLPattern = regexp.MustCompile("(?is)\\bSELECT\\b.{0,240}\\bFROM\\b|\\bINSERT\\s+INTO\\b|\\bUPDATE\\s+[A-Za-z0-9_\\.\\\"]+\\s+SET\\b|\\bDELETE\\s+FROM\\b|\\bCREATE\\s+(TABLE|INDEX|VIEW)\\b|\\bALTER\\s+TABLE\\b|\\bDROP\\s+(TABLE|INDEX|VIEW)\\b|\\bMERGE\\s+INTO\\b")

func main() {
	var (
		baselineID = flag.String("baseline-id", "B", "Handover Baseline ID")
		baselineRev = flag.String(
			"baseline-revision",
			"e5231839a5dcc62cce1181a2731fcb7c70d16736",
			"Baseline Repository revision R",
		)
		targetRev = flag.String("target-revision", "", "Target build revision T; defaults to current clean HEAD")
	)
	flag.Parse()

	result, err := run(*baselineID, *baselineRev, *targetRev)
	encoded, encodeErr := json.MarshalIndent(result, "", "  ")
	if encodeErr != nil {
		fmt.Fprintln(os.Stderr, encodeErr)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(baselineID, baselineRevision, requestedTargetRevision string) (evidence, error) {
	result := evidence{
		EvidenceNature:             evidenceNature,
		HandoverBaselineID:         strings.TrimSpace(baselineID),
		BaselineRepositoryRevision: strings.TrimSpace(baselineRevision),
		AuthorityLimits: []string{
			"first-reference-uses-coarse-workspace-world-invariance-not-production-semantic-diff-taxonomy",
			"web-source-scan-proves-absence-of-authoritative-files-and-native-sql-not-all-possible-business-duplication",
			"shared-command-boundary-is-proven-by-reference-source-test",
			"architecture-evidence-is-derived-not-canonical",
		},
		Unresolved: []string{
			"browse-environment-instance-binding-not-exposed-by-current-driver-api",
			"actual-external-command-observation-not-acquired",
			"general-target-neutral-presentation-taxonomy-not-yet-defined",
		},
	}
	if result.HandoverBaselineID == "" {
		return result, fmt.Errorf("handover baseline ID is required")
	}
	if result.BaselineRepositoryRevision == "" {
		return result, fmt.Errorf("baseline repository revision is required")
	}
	if err := requireFixedGoEnvironment(); err != nil {
		return result, err
	}

	source, err := repositoryevidence.InspectStart()
	if err != nil {
		return result, err
	}
	result.TargetBuildRevision = source.RepositoryRevision
	result.WorkingTreeClean = source.WorkingTreeClean
	if !source.WorkingTreeClean {
		return result, fmt.Errorf("Gate C architecture Evidence requires a clean full checkout")
	}
	if requested := strings.TrimSpace(requestedTargetRevision); requested != "" &&
		requested != source.RepositoryRevision {
		return result, fmt.Errorf(
			"requested target revision %s does not match clean HEAD %s",
			requested,
			source.RepositoryRevision,
		)
	}

	if err := requireAncestor(result.BaselineRepositoryRevision, result.TargetBuildRevision); err != nil {
		return result, err
	}

	changed, err := gitLines(
		"diff",
		"--name-only",
		result.BaselineRepositoryRevision+".."+result.TargetBuildRevision,
		"--",
		"reference/workspace",
		"reference/world.json",
	)
	if err != nil {
		return result, fmt.Errorf("compare baseline Canonical inputs: %w", err)
	}
	result.CanonicalWorldInvariant = checkEvidence{
		Name:   "baseline-workspace-world-unchanged",
		Passed: len(changed) == 0,
	}
	if len(changed) != 0 {
		result.CanonicalWorldInvariant.Details = append(
			result.CanonicalWorldInvariant.Details,
			changed...,
		)
		return result, fmt.Errorf(
			"baseline Canonical Workspace / generated World changed between R and T; successor Handover is required",
		)
	}
	result.CanonicalWorldInvariant.Details = []string{
		"reference/workspace unchanged between R and T",
		"reference/world.json unchanged between R and T",
	}

	violations, err := scanWebTargetAuthority("tools/reference-first-web-target")
	if err != nil {
		return result, err
	}
	result.WebTargetAuthorityBoundary = checkEvidence{
		Name:    "web-target-has-no-authoritative-tei-sql-or-native-sql",
		Passed:  len(violations) == 0,
		Details: violations,
	}
	if len(violations) != 0 {
		return result, fmt.Errorf(
			"Web Target authority boundary violation: %s",
			strings.Join(violations, "; "),
		)
	}
	result.WebTargetAuthorityBoundary.Details = []string{
		"no .tei files under tools/reference-first-web-target",
		"no .sql files under tools/reference-first-web-target",
		"no native SQL statement text in Web Target Go/HTML source",
	}

	command := []string{
		"test",
		"./tools/reference-first-web-target",
		"-run",
		"^(TestCustomSubmitUIUsesSameCommandScenarioRunnerAsJSONEndpoint|TestSubmitScenarioHTTPPreservesExecutionErrorProvenanceAndPartialStage|TestReferenceCommandScenarioUsesRuntimeFixtureForValidationNotSatisfied)$",
		"-count=1",
	}
	if err := runGo(command...); err != nil {
		result.SharedCommandBoundary = checkEvidence{
			Name:    "shared-command-and-result-boundary-pressure",
			Passed:  false,
			Details: []string{"go " + strings.Join(command, " ")},
		}
		return result, err
	}
	result.SharedCommandBoundary = checkEvidence{
		Name:   "shared-command-and-result-boundary-pressure",
		Passed: true,
		Details: []string{
			"go " + strings.Join(command, " "),
			"standard JSON interaction and Custom UI share commandScenarioRunner",
			"Validation not-satisfied stops before Authorization / Command",
			"Command Execution Error retains B/R/T and remains distinct from Business Result / Observation",
		},
	}

	post, err := repositoryevidence.InspectPost(source.RepositoryRevision)
	result.PostRunRepository = post
	if err != nil {
		return result, err
	}
	result.ReproducibleRevision = repositoryevidence.IsReproducible(source, post)
	if !result.ReproducibleRevision {
		return result, fmt.Errorf("Gate C architecture Evidence did not finish on the same clean Target revision")
	}
	return result, nil
}

func requireFixedGoEnvironment() error {
	if os.Getenv("GOENV") != "off" {
		return fmt.Errorf("GOENV must be exact off")
	}
	if os.Getenv("GOWORK") != "off" {
		return fmt.Errorf("GOWORK must be exact off")
	}
	if os.Getenv("GOFLAGS") != "" {
		return fmt.Errorf("GOFLAGS must be empty")
	}
	return nil
}

func requireAncestor(baselineRevision, targetRevision string) error {
	cmd := exec.Command(
		"git",
		"merge-base",
		"--is-ancestor",
		baselineRevision,
		targetRevision,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf(
			"baseline revision %s is not an ancestor of target revision %s: %w: %s",
			baselineRevision,
			targetRevision,
			err,
			strings.TrimSpace(string(output)),
		)
	}
	return nil
}

func gitLines(args ...string) ([]string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"git %s: %w: %s",
			strings.Join(args, " "),
			err,
			strings.TrimSpace(string(out)),
		)
	}
	var result []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	sort.Strings(result)
	return result, nil
}

func scanWebTargetAuthority(root string) ([]string, error) {
	var violations []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		lower := strings.ToLower(entry.Name())
		if strings.HasSuffix(lower, ".tei") {
			violations = append(violations, path+": authoritative .tei file is not allowed in Web Target")
			return nil
		}
		if strings.HasSuffix(lower, ".sql") {
			violations = append(violations, path+": native .sql file is not allowed in Web Target")
			return nil
		}
		if !strings.HasSuffix(lower, ".go") &&
			!strings.HasSuffix(lower, ".html") &&
			!strings.HasSuffix(lower, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if nativeSQLPattern.Find(data) != nil {
			violations = append(violations, path+": native SQL statement text detected")
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan Web Target authority boundary: %w", err)
	}
	sort.Strings(violations)
	return violations, nil
}

func runGo(args ...string) error {
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go %s: %w", strings.Join(args, " "), err)
	}
	return nil
}
