package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type semanticFunctionalSummary struct {
	EvidenceNature             string             `json:"evidenceNature"`
	RunIdentity                string             `json:"runIdentity"`
	HandoverBaselineID         string             `json:"handoverBaselineId"`
	BaselineRepositoryRevision string             `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string             `json:"targetBuildRevision"`
	TargetAdapter              string             `json:"targetAdapter"`
	Host                       string             `json:"host"`
	SourceQualification        passState          `json:"sourceQualification"`
	SemanticConformance        passState          `json:"semanticConformance"`
	FunctionalEvidence         passState          `json:"functionalEvidence"`
	VisualEvidence             pendingVisualState `json:"visualEvidence"`
	GateCReady                 bool               `json:"gateCReady"`
}

type passState struct {
	Passed bool `json:"passed"`
}

type pendingVisualState struct {
	Status   string `json:"status"`
	Required bool   `json:"required"`
}

type visualManifest struct {
	EvidenceNature                      string            `json:"evidenceNature"`
	VisualRunIdentity                   string            `json:"visualRunIdentity"`
	SourceSemanticFunctionalRunIdentity string            `json:"sourceSemanticFunctionalRunIdentity"`
	HandoverBaselineID                  string            `json:"handoverBaselineId"`
	BaselineRepositoryRevision          string            `json:"baselineRepositoryRevision"`
	TargetBuildRevision                 string            `json:"targetBuildRevision"`
	TargetAdapter                       string            `json:"targetAdapter"`
	Host                                string            `json:"host"`
	Evaluator                           visualEvaluator   `json:"evaluator"`
	Captures                            []visualCapture   `json:"captures"`
	Criteria                            []visualCriterion `json:"criteria"`
}

type visualEvaluator struct {
	Identity string `json:"identity"`
	Version  string `json:"version"`
	Method   string `json:"method"`
}

type visualCapture struct {
	ID             string `json:"id"`
	Surface        string `json:"surface"`
	State          string `json:"state"`
	File           string `json:"file"`
	MediaType      string `json:"mediaType"`
	ViewportWidth  int    `json:"viewportWidth"`
	ViewportHeight int    `json:"viewportHeight"`
}

type visualCriterion struct {
	ID          string   `json:"id"`
	Result      string   `json:"result"`
	CaptureRefs []string `json:"captureRefs"`
	Notes       string   `json:"notes,omitempty"`
}

type captureDigest struct {
	ID       string `json:"id"`
	File     string `json:"file"`
	SHA256   string `json:"sha256"`
	ByteSize int64  `json:"byteSize"`
}

type criterionAudit struct {
	ID          string   `json:"id"`
	Result      string   `json:"result"`
	CaptureRefs []string `json:"captureRefs"`
}

type visualAuditSummary struct {
	EvidenceNature                      string           `json:"evidenceNature"`
	VisualRunIdentity                   string           `json:"visualRunIdentity"`
	SourceSemanticFunctionalRunIdentity string           `json:"sourceSemanticFunctionalRunIdentity"`
	HandoverBaselineID                  string           `json:"handoverBaselineId"`
	BaselineRepositoryRevision          string           `json:"baselineRepositoryRevision"`
	TargetBuildRevision                 string           `json:"targetBuildRevision"`
	TargetAdapter                       string           `json:"targetAdapter"`
	Host                                string           `json:"host"`
	Evaluator                           visualEvaluator  `json:"evaluator"`
	AllRequiredCapturesPresent          bool             `json:"allRequiredCapturesPresent"`
	AllCriteriaResolved                 bool             `json:"allCriteriaResolved"`
	VisualFidelityPassed                bool             `json:"visualFidelityPassed"`
	Captures                            []captureDigest  `json:"captures"`
	Criteria                            []criterionAudit `json:"criteria"`
	GateCReady                          bool             `json:"gateCReady"`
	AuthorityLimits                     []string         `json:"authorityLimits"`
}

var requiredCaptures = map[string]struct {
	Surface string
	State   string
}{
	"projection-index":                       {Surface: "standard-projection", State: "initial"},
	"custom-submit-initial":                  {Surface: "custom-submit", State: "initial"},
	"custom-submit-success":                  {Surface: "custom-submit", State: "success"},
	"custom-submit-authorization-denied":     {Surface: "custom-submit", State: "authorization-denied"},
	"custom-submit-validation-not-satisfied": {Surface: "custom-submit", State: "validation-not-satisfied"},
	"custom-submit-execution-error":          {Surface: "custom-submit", State: "execution-error"},
}

var requiredCriteria = []string{
	"field-grouping-readability",
	"relation-reference-comprehensibility",
	"header-detail-locality",
	"validation-feedback-distinguishable",
	"authorization-denial-vs-system-error-distinguishable",
	"business-result-vs-observation-distinguishable",
	"standard-vs-custom-ui-difference-explainable",
}

func main() {
	var (
		semanticFunctional = flag.String("semantic-functional-summary", "", "Semantic / Functional summary JSON")
		manifestPath       = flag.String("visual-manifest", "", "Visual Evidence manifest JSON")
		artifactDir        = flag.String("artifact-dir", "", "Directory containing screenshot artifacts")
	)
	flag.Parse()
	if *semanticFunctional == "" || *manifestPath == "" || *artifactDir == "" {
		fatalf("-semantic-functional-summary, -visual-manifest and -artifact-dir are required")
	}

	summary, err := auditVisualEvidence(*semanticFunctional, *manifestPath, *artifactDir)
	if err != nil {
		fatalf("%v", err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		fatalf("encode Visual Evidence audit: %v", err)
	}
}

func auditVisualEvidence(
	semanticFunctionalPath string,
	manifestPath string,
	artifactDir string,
) (visualAuditSummary, error) {
	var semanticFunctional semanticFunctionalSummary
	if err := readJSON(semanticFunctionalPath, &semanticFunctional); err != nil {
		return visualAuditSummary{}, err
	}
	if semanticFunctional.EvidenceNature != "first-web-target-semantic-functional-evidence" ||
		!semanticFunctional.SourceQualification.Passed ||
		!semanticFunctional.SemanticConformance.Passed ||
		!semanticFunctional.FunctionalEvidence.Passed ||
		semanticFunctional.VisualEvidence.Status != "pending" ||
		!semanticFunctional.VisualEvidence.Required ||
		semanticFunctional.GateCReady {
		return visualAuditSummary{}, fmt.Errorf(
			"Semantic / Functional summary is not a valid Visual intake source: %+v",
			semanticFunctional,
		)
	}

	var manifest visualManifest
	if err := readJSON(manifestPath, &manifest); err != nil {
		return visualAuditSummary{}, err
	}
	if manifest.EvidenceNature != "first-web-target-visual-evidence" {
		return visualAuditSummary{}, fmt.Errorf("visual evidenceNature=%q", manifest.EvidenceNature)
	}
	if manifest.VisualRunIdentity == "" {
		return visualAuditSummary{}, fmt.Errorf("visualRunIdentity is required")
	}
	if manifest.SourceSemanticFunctionalRunIdentity != semanticFunctional.RunIdentity {
		return visualAuditSummary{}, fmt.Errorf(
			"visual source run=%q, want %q",
			manifest.SourceSemanticFunctionalRunIdentity,
			semanticFunctional.RunIdentity,
		)
	}
	if err := requireSameTargetProvenance(manifest, semanticFunctional); err != nil {
		return visualAuditSummary{}, err
	}
	if manifest.Evaluator.Identity == "" ||
		manifest.Evaluator.Version == "" ||
		manifest.Evaluator.Method == "" {
		return visualAuditSummary{}, fmt.Errorf("evaluator identity, version and method are required")
	}

	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return visualAuditSummary{}, err
	}
	for _, forbidden := range []string{
		"editor-secret",
		"viewer-secret",
		"locked-secret",
		"disabled-secret",
		"credential-ref://legacy/",
	} {
		if strings.Contains(string(manifestBytes), forbidden) {
			return visualAuditSummary{}, fmt.Errorf("visual manifest contains credential material %q", forbidden)
		}
	}

	capturesByID := make(map[string]visualCapture, len(manifest.Captures))
	for _, capture := range manifest.Captures {
		if capture.ID == "" {
			return visualAuditSummary{}, fmt.Errorf("visual capture ID is required")
		}
		if _, exists := capturesByID[capture.ID]; exists {
			return visualAuditSummary{}, fmt.Errorf("visual capture %s is duplicated", capture.ID)
		}
		if capture.ViewportWidth <= 0 || capture.ViewportHeight <= 0 {
			return visualAuditSummary{}, fmt.Errorf("visual capture %s has invalid viewport", capture.ID)
		}
		if capture.MediaType != "image/png" {
			return visualAuditSummary{}, fmt.Errorf(
				"visual capture %s mediaType=%q; first Reference accepts image/png",
				capture.ID,
				capture.MediaType,
			)
		}
		if filepath.Base(capture.File) != capture.File || capture.File == "." || capture.File == "" {
			return visualAuditSummary{}, fmt.Errorf("visual capture %s has unsafe file %q", capture.ID, capture.File)
		}
		capturesByID[capture.ID] = capture
	}

	for id, expected := range requiredCaptures {
		capture, ok := capturesByID[id]
		if !ok {
			return visualAuditSummary{}, fmt.Errorf("required visual capture %s is missing", id)
		}
		if capture.Surface != expected.Surface || capture.State != expected.State {
			return visualAuditSummary{}, fmt.Errorf(
				"visual capture %s surface/state=%s/%s, want %s/%s",
				id,
				capture.Surface,
				capture.State,
				expected.Surface,
				expected.State,
			)
		}
	}

	resolvedArtifactDir, err := filepath.Abs(artifactDir)
	if err != nil {
		return visualAuditSummary{}, fmt.Errorf("resolve artifact directory: %w", err)
	}
	resolvedArtifactDir, err = filepath.EvalSymlinks(resolvedArtifactDir)
	if err != nil {
		return visualAuditSummary{}, fmt.Errorf("resolve artifact directory symlinks: %w", err)
	}

	digests := make([]captureDigest, 0, len(manifest.Captures))
	for _, capture := range manifest.Captures {
		path := filepath.Join(resolvedArtifactDir, capture.File)
		resolvedPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return visualAuditSummary{}, fmt.Errorf("resolve visual capture %s: %w", capture.ID, err)
		}
		if filepath.Dir(resolvedPath) != resolvedArtifactDir {
			return visualAuditSummary{}, fmt.Errorf(
				"visual capture %s resolves outside artifact directory",
				capture.ID,
			)
		}
		data, err := os.ReadFile(resolvedPath)
		if err != nil {
			return visualAuditSummary{}, fmt.Errorf("read visual capture %s: %w", capture.ID, err)
		}
		if len(data) < 8 || string(data[:8]) != "\x89PNG\r\n\x1a\n" {
			return visualAuditSummary{}, fmt.Errorf("visual capture %s is not a PNG artifact", capture.ID)
		}
		info, err := os.Stat(resolvedPath)
		if err != nil {
			return visualAuditSummary{}, fmt.Errorf("stat visual capture %s: %w", capture.ID, err)
		}
		if !info.Mode().IsRegular() {
			return visualAuditSummary{}, fmt.Errorf("visual capture %s is not a regular file", capture.ID)
		}
		sum := sha256.Sum256(data)
		digests = append(digests, captureDigest{
			ID:       capture.ID,
			File:     capture.File,
			SHA256:   hex.EncodeToString(sum[:]),
			ByteSize: info.Size(),
		})
	}
	sort.Slice(digests, func(i, j int) bool { return digests[i].ID < digests[j].ID })

	criteriaByID := make(map[string]visualCriterion, len(manifest.Criteria))
	for _, criterion := range manifest.Criteria {
		if criterion.ID == "" {
			return visualAuditSummary{}, fmt.Errorf("visual criterion ID is required")
		}
		if _, exists := criteriaByID[criterion.ID]; exists {
			return visualAuditSummary{}, fmt.Errorf("visual criterion %s is duplicated", criterion.ID)
		}
		if criterion.Result != "pass" &&
			criterion.Result != "fail" &&
			criterion.Result != "unresolved" {
			return visualAuditSummary{}, fmt.Errorf(
				"visual criterion %s has invalid result %q",
				criterion.ID,
				criterion.Result,
			)
		}
		if len(criterion.CaptureRefs) == 0 {
			return visualAuditSummary{}, fmt.Errorf("visual criterion %s has no captureRefs", criterion.ID)
		}
		for _, ref := range criterion.CaptureRefs {
			if _, ok := capturesByID[ref]; !ok {
				return visualAuditSummary{}, fmt.Errorf(
					"visual criterion %s references unknown capture %s",
					criterion.ID,
					ref,
				)
			}
		}
		criteriaByID[criterion.ID] = criterion
	}

	allResolved := true
	allPassed := true
	auditedCriteria := make([]criterionAudit, 0, len(requiredCriteria))
	for _, id := range requiredCriteria {
		criterion, ok := criteriaByID[id]
		if !ok {
			return visualAuditSummary{}, fmt.Errorf("required visual criterion %s is missing", id)
		}
		if criterion.Result == "unresolved" {
			allResolved = false
		}
		if criterion.Result != "pass" {
			allPassed = false
		}
		refs := append([]string(nil), criterion.CaptureRefs...)
		sort.Strings(refs)
		auditedCriteria = append(auditedCriteria, criterionAudit{
			ID:          id,
			Result:      criterion.Result,
			CaptureRefs: refs,
		})
	}

	return visualAuditSummary{
		EvidenceNature:                      "first-web-target-visual-evidence-audit",
		VisualRunIdentity:                   manifest.VisualRunIdentity,
		SourceSemanticFunctionalRunIdentity: manifest.SourceSemanticFunctionalRunIdentity,
		HandoverBaselineID:                  manifest.HandoverBaselineID,
		BaselineRepositoryRevision:          manifest.BaselineRepositoryRevision,
		TargetBuildRevision:                 manifest.TargetBuildRevision,
		TargetAdapter:                       manifest.TargetAdapter,
		Host:                                manifest.Host,
		Evaluator:                           manifest.Evaluator,
		AllRequiredCapturesPresent:          true,
		AllCriteriaResolved:                 allResolved,
		VisualFidelityPassed:                allResolved && allPassed,
		Captures:                            digests,
		Criteria:                            auditedCriteria,
		GateCReady:                          false,
		AuthorityLimits: []string{
			"visual-evaluator-result-is-attributed-evidence-not-canonical-meaning",
			"visual-pass-does-not-auto-declare-gate-c-ready",
			"screenshot-files-are-derived-evidence-not-source-of-truth",
			"capture-tool-and-evaluator-remain-exchangeable",
		},
	}, nil
}

func requireSameTargetProvenance(
	visual visualManifest,
	semantic semanticFunctionalSummary,
) error {
	if visual.HandoverBaselineID != semantic.HandoverBaselineID ||
		visual.BaselineRepositoryRevision != semantic.BaselineRepositoryRevision ||
		visual.TargetBuildRevision != semantic.TargetBuildRevision ||
		visual.TargetAdapter != semantic.TargetAdapter ||
		visual.Host != semantic.Host {
		return fmt.Errorf(
			"Visual provenance B/R/T/Target does not match Semantic / Functional source",
		)
	}
	return nil
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
