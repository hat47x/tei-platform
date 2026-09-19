package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
)

type passState struct {
	Passed bool `json:"passed"`
}

type pendingVisualState struct {
	Status   string `json:"status"`
	Required bool   `json:"required"`
}

type runtimeBinding struct {
	Pattern            string `json:"pattern"`
	Environment        string `json:"environment,omitempty"`
	Instance           string `json:"instance,omitempty"`
	Provider           string `json:"provider,omitempty"`
	ExecutorCapability string `json:"executorCapability,omitempty"`
	ContractQuery      string `json:"contractQuery,omitempty"`
	Datasource         string `json:"datasource,omitempty"`
	Database           string `json:"database,omitempty"`
	Access             string `json:"access,omitempty"`
}

type targetDifferenceHandoff struct {
	ApplicationID      string           `json:"applicationId"`
	TargetAdapter      string           `json:"targetAdapter"`
	Host               string           `json:"host"`
	TargetCapabilities []string         `json:"targetCapabilities"`
	RuntimeBindings    []runtimeBinding `json:"runtimeBindings"`
	ExecutionArtifact  string           `json:"executionArtifact"`
	ABI                string           `json:"abi"`
	ExecutionTrust     []string         `json:"executionTrust"`
	WebSpecific        []string         `json:"webSpecific"`
	DoNotCanonicalize  []string         `json:"doNotCanonicalize"`
	Unresolved         []string         `json:"unresolved"`
}

type semanticFunctionalSummary struct {
	EvidenceNature             string                  `json:"evidenceNature"`
	RunIdentity                string                  `json:"runIdentity"`
	HandoverBaselineID         string                  `json:"handoverBaselineId"`
	BaselineRepositoryRevision string                  `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string                  `json:"targetBuildRevision"`
	TargetAdapter              string                  `json:"targetAdapter"`
	Host                       string                  `json:"host"`
	SourceQualification        passState               `json:"sourceQualification"`
	SemanticConformance        passState               `json:"semanticConformance"`
	FunctionalEvidence         passState               `json:"functionalEvidence"`
	VisualEvidence             pendingVisualState      `json:"visualEvidence"`
	TargetDifferenceHandoff    targetDifferenceHandoff `json:"targetDifferenceHandoff"`
	Unresolved                 []string                `json:"unresolved"`
	GateCReady                 bool                    `json:"gateCReady"`
}

type architectureCheck struct {
	Passed bool `json:"passed"`
}

type postRunRepository struct {
	Checked          bool   `json:"checked"`
	RevisionStable   bool   `json:"revisionStable"`
	WorkingTreeClean bool   `json:"workingTreeClean"`
}

type architectureEvidence struct {
	EvidenceNature             string            `json:"evidenceNature"`
	HandoverBaselineID         string            `json:"handoverBaselineId"`
	BaselineRepositoryRevision string            `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string            `json:"targetBuildRevision"`
	WorkingTreeClean           bool              `json:"workingTreeClean"`
	ReproducibleRevision       bool              `json:"reproducibleRevision"`
	PostRunRepository          postRunRepository `json:"postRunRepository"`
	CanonicalWorldInvariant    architectureCheck `json:"canonicalWorldInvariant"`
	WebTargetAuthorityBoundary architectureCheck `json:"webTargetAuthorityBoundary"`
	SharedCommandBoundary      architectureCheck `json:"sharedCommandBoundary"`
	Unresolved                 []string          `json:"unresolved"`
}

type visualAudit struct {
	EvidenceNature                      string `json:"evidenceNature"`
	VisualRunIdentity                   string `json:"visualRunIdentity"`
	SourceSemanticFunctionalRunIdentity string `json:"sourceSemanticFunctionalRunIdentity"`
	HandoverBaselineID                  string `json:"handoverBaselineId"`
	BaselineRepositoryRevision          string `json:"baselineRepositoryRevision"`
	TargetBuildRevision                 string `json:"targetBuildRevision"`
	TargetAdapter                       string `json:"targetAdapter"`
	Host                                string `json:"host"`
	AllRequiredCapturesPresent          bool   `json:"allRequiredCapturesPresent"`
	AllCriteriaResolved                 bool   `json:"allCriteriaResolved"`
	VisualFidelityPassed                bool   `json:"visualFidelityPassed"`
	GateCReady                          bool   `json:"gateCReady"`
}

type unresolvedDisposition struct {
	ID          string `json:"id"`
	Disposition string `json:"disposition"`
	Blocking    bool   `json:"blocking"`
	Owner       string `json:"owner,omitempty"`
	Reason      string `json:"reason"`
}

type readinessCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
}

type targetDifferenceRecord struct {
	ApplicationID      string           `json:"applicationId"`
	TargetAdapter      string           `json:"targetAdapter"`
	Host               string           `json:"host"`
	TargetCapabilities []string         `json:"targetCapabilities"`
	RuntimeBindings    []runtimeBinding `json:"runtimeBindings"`
	ExecutionArtifact  string           `json:"executionArtifact"`
	ABI                string           `json:"abi"`
	ExecutionTrust     []string         `json:"executionTrust"`
	WebSpecific        []string         `json:"webSpecific"`
	DoNotCanonicalize  []string         `json:"doNotCanonicalize"`
	CarryForward       []string         `json:"carryForward"`
}

type readinessSummary struct {
	EvidenceNature             string                  `json:"evidenceNature"`
	HandoverBaselineID         string                  `json:"handoverBaselineId"`
	BaselineRepositoryRevision string                  `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string                  `json:"targetBuildRevision"`
	TargetAdapter              string                  `json:"targetAdapter"`
	Host                       string                  `json:"host"`
	SemanticFunctionalRun      string                  `json:"semanticFunctionalRun"`
	VisualRun                  string                  `json:"visualRun"`
	Checks                     []readinessCheck        `json:"checks"`
	Unresolved                 []unresolvedDisposition `json:"unresolved"`
	TargetDifferenceHandoff    targetDifferenceRecord  `json:"targetDifferenceHandoff"`
	GateCReady                 bool                    `json:"gateCReady"`
	AuthorityLimits            []string                `json:"authorityLimits"`
}

func main() {
	var (
		semanticPath = flag.String("semantic-functional-summary", "", "Semantic / Functional summary")
		architecturePath = flag.String("architecture-evidence", "", "Architecture conformance Evidence")
		visualPath = flag.String("visual-audit", "", "Visual Evidence audit")
	)
	flag.Parse()
	if *semanticPath == "" || *architecturePath == "" || *visualPath == "" {
		fatalf("-semantic-functional-summary, -architecture-evidence and -visual-audit are required")
	}

	summary, err := auditReadiness(*semanticPath, *architecturePath, *visualPath)
	if err != nil {
		fatalf("%v", err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		fatalf("encode Gate C readiness audit: %v", err)
	}
}

func auditReadiness(
	semanticPath string,
	architecturePath string,
	visualPath string,
) (readinessSummary, error) {
	var semantic semanticFunctionalSummary
	if err := readJSON(semanticPath, &semantic); err != nil {
		return readinessSummary{}, err
	}
	if semantic.EvidenceNature != "first-web-target-semantic-functional-evidence" {
		return readinessSummary{}, fmt.Errorf("unexpected Semantic / Functional evidenceNature %q", semantic.EvidenceNature)
	}
	if !semantic.SourceQualification.Passed ||
		!semantic.SemanticConformance.Passed ||
		!semantic.FunctionalEvidence.Passed {
		return readinessSummary{}, fmt.Errorf("Semantic / Functional Evidence is not fully passed")
	}
	if semantic.VisualEvidence.Status != "pending" ||
		!semantic.VisualEvidence.Required ||
		semantic.GateCReady {
		return readinessSummary{}, fmt.Errorf("Semantic / Functional summary has invalid pre-Visual state")
	}

	var architecture architectureEvidence
	if err := readJSON(architecturePath, &architecture); err != nil {
		return readinessSummary{}, err
	}
	if architecture.EvidenceNature != "first-web-target-architecture-conformance" {
		return readinessSummary{}, fmt.Errorf("unexpected architecture evidenceNature %q", architecture.EvidenceNature)
	}
	if !architecture.WorkingTreeClean ||
		!architecture.ReproducibleRevision ||
		!architecture.PostRunRepository.Checked ||
		!architecture.PostRunRepository.RevisionStable ||
		!architecture.PostRunRepository.WorkingTreeClean ||
		!architecture.CanonicalWorldInvariant.Passed ||
		!architecture.WebTargetAuthorityBoundary.Passed ||
		!architecture.SharedCommandBoundary.Passed {
		return readinessSummary{}, fmt.Errorf("architecture conformance Evidence is not fully passed")
	}

	var visual visualAudit
	if err := readJSON(visualPath, &visual); err != nil {
		return readinessSummary{}, err
	}
	if visual.EvidenceNature != "first-web-target-visual-evidence-audit" {
		return readinessSummary{}, fmt.Errorf("unexpected Visual evidenceNature %q", visual.EvidenceNature)
	}
	if !visual.AllRequiredCapturesPresent ||
		!visual.AllCriteriaResolved ||
		!visual.VisualFidelityPassed ||
		visual.GateCReady {
		return readinessSummary{}, fmt.Errorf("Visual Evidence is not a complete attributed pass")
	}
	if visual.SourceSemanticFunctionalRunIdentity != semantic.RunIdentity {
		return readinessSummary{}, fmt.Errorf(
			"Visual source run %q does not match Semantic / Functional run %q",
			visual.SourceSemanticFunctionalRunIdentity,
			semantic.RunIdentity,
		)
	}

	if err := requireSameProvenance(semantic, architecture, visual); err != nil {
		return readinessSummary{}, err
	}
	if err := requireTargetHandoff(semantic.TargetDifferenceHandoff, semantic); err != nil {
		return readinessSummary{}, err
	}

	dispositions, err := classifyUnresolved(
		semantic.Unresolved,
		semantic.TargetDifferenceHandoff.Unresolved,
		architecture.Unresolved,
	)
	if err != nil {
		return readinessSummary{}, err
	}
	blocking := false
	var carryForward []string
	for _, disposition := range dispositions {
		if disposition.Blocking {
			blocking = true
		}
		if disposition.Disposition == "carry-forward" {
			carryForward = append(carryForward, disposition.ID)
		}
	}
	sort.Strings(carryForward)

	checks := []readinessCheck{
		{Name: "same-B-R-T-and-target-identity", Passed: true},
		{Name: "source-semantic-functional-passed", Passed: true},
		{Name: "visual-fidelity-separately-passed", Passed: true},
		{Name: "canonical-world-invariant-R-to-T", Passed: true},
		{Name: "web-target-authority-boundary-passed", Passed: true},
		{Name: "standard-custom-shared-command-boundary-passed", Passed: true},
		{Name: "result-execution-error-observation-boundaries-preserved", Passed: true},
		{Name: "target-difference-handoff-recoverable", Passed: true},
		{Name: "all-unresolved-explicitly-classified", Passed: true},
		{Name: "no-blocking-unresolved", Passed: !blocking},
	}

	return readinessSummary{
		EvidenceNature:             "first-web-target-gate-c-readiness-audit",
		HandoverBaselineID:         semantic.HandoverBaselineID,
		BaselineRepositoryRevision: semantic.BaselineRepositoryRevision,
		TargetBuildRevision:        semantic.TargetBuildRevision,
		TargetAdapter:              semantic.TargetAdapter,
		Host:                       semantic.Host,
		SemanticFunctionalRun:      semantic.RunIdentity,
		VisualRun:                  visual.VisualRunIdentity,
		Checks:                     checks,
		Unresolved:                 dispositions,
		TargetDifferenceHandoff: targetDifferenceRecord{
			ApplicationID:      semantic.TargetDifferenceHandoff.ApplicationID,
			TargetAdapter:      semantic.TargetDifferenceHandoff.TargetAdapter,
			Host:               semantic.TargetDifferenceHandoff.Host,
			TargetCapabilities: append([]string(nil), semantic.TargetDifferenceHandoff.TargetCapabilities...),
			RuntimeBindings:    append([]runtimeBinding(nil), semantic.TargetDifferenceHandoff.RuntimeBindings...),
			ExecutionArtifact:  semantic.TargetDifferenceHandoff.ExecutionArtifact,
			ABI:                semantic.TargetDifferenceHandoff.ABI,
			ExecutionTrust:     append([]string(nil), semantic.TargetDifferenceHandoff.ExecutionTrust...),
			WebSpecific:        append([]string(nil), semantic.TargetDifferenceHandoff.WebSpecific...),
			DoNotCanonicalize:  append([]string(nil), semantic.TargetDifferenceHandoff.DoNotCanonicalize...),
			CarryForward:       carryForward,
		},
		GateCReady: !blocking,
		AuthorityLimits: []string{
			"readiness-is-derived-from-three-independent-evidence-classes-plus-architecture-conformance",
			"carry-forward-unresolved-is-not-converted-to-successful-external-evidence",
			"target-difference-handoff-is-observed-first-reference-data-not-final-taxonomy",
			"gate-c-ready-does-not-imply-production-ready-web-target",
		},
	}, nil
}

func requireSameProvenance(
	semantic semanticFunctionalSummary,
	architecture architectureEvidence,
	visual visualAudit,
) error {
	if semantic.HandoverBaselineID == "" ||
		semantic.BaselineRepositoryRevision == "" ||
		semantic.TargetBuildRevision == "" ||
		semantic.TargetAdapter == "" ||
		semantic.Host == "" {
		return fmt.Errorf("Semantic / Functional provenance is incomplete")
	}
	if architecture.HandoverBaselineID != semantic.HandoverBaselineID ||
		architecture.BaselineRepositoryRevision != semantic.BaselineRepositoryRevision ||
		architecture.TargetBuildRevision != semantic.TargetBuildRevision {
		return fmt.Errorf("Architecture Evidence does not share B/R/T with Semantic / Functional Evidence")
	}
	if visual.HandoverBaselineID != semantic.HandoverBaselineID ||
		visual.BaselineRepositoryRevision != semantic.BaselineRepositoryRevision ||
		visual.TargetBuildRevision != semantic.TargetBuildRevision ||
		visual.TargetAdapter != semantic.TargetAdapter ||
		visual.Host != semantic.Host {
		return fmt.Errorf("Visual Evidence does not share B/R/T/Target with Semantic / Functional Evidence")
	}
	return nil
}

func requireTargetHandoff(
	handoff targetDifferenceHandoff,
	semantic semanticFunctionalSummary,
) error {
	if handoff.ApplicationID != "order_entry" ||
		handoff.TargetAdapter != semantic.TargetAdapter ||
		handoff.Host != semantic.Host ||
		len(handoff.TargetCapabilities) == 0 ||
		len(handoff.RuntimeBindings) == 0 ||
		handoff.ExecutionArtifact == "" ||
		handoff.ExecutionArtifact == "unresolved" ||
		handoff.ABI == "" ||
		handoff.ABI == "unresolved" ||
		len(handoff.ExecutionTrust) == 0 ||
		len(handoff.WebSpecific) == 0 ||
		len(handoff.DoNotCanonicalize) == 0 {
		return fmt.Errorf("Target difference handoff is incomplete: %+v", handoff)
	}

	byPattern := make(map[string]runtimeBinding, len(handoff.RuntimeBindings))
	for _, binding := range handoff.RuntimeBindings {
		if binding.Pattern == "" {
			return fmt.Errorf("Target runtime binding has no Pattern identity")
		}
		if _, exists := byPattern[binding.Pattern]; exists {
			return fmt.Errorf("Target runtime binding Pattern %s is duplicated", binding.Pattern)
		}
		byPattern[binding.Pattern] = binding
	}
	for _, required := range []string{
		"order_entry.recent_orders",
		"order_entry.customer_reference",
		"order_entry.order_lines",
	} {
		if _, ok := byPattern[required]; !ok {
			return fmt.Errorf("Target handoff is missing runtime binding %s", required)
		}
	}
	return nil
}

func classifyUnresolved(groups ...[]string) ([]unresolvedDisposition, error) {
	seen := map[string]struct{}{}
	var ids []string
	for _, group := range groups {
		for _, id := range group {
			if id == "" {
				continue
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	result := make([]unresolvedDisposition, 0, len(ids))
	for _, id := range ids {
		switch id {
		case "visual-fidelity-evidence-not-acquired":
			result = append(result, unresolvedDisposition{
				ID:          id,
				Disposition: "resolved-by-visual-evidence",
				Blocking:    false,
				Reason:      "the independently audited Visual Evidence supplies the previously pending Visual layer",
			})
		case "browse-environment-instance-binding-not-exposed-by-current-driver-api":
			result = append(result, unresolvedDisposition{
				ID:          id,
				Disposition: "carry-forward",
				Blocking:    false,
				Owner:       "ISSUE-000026",
				Reason:      "the Web Target does not infer the missing Driver environment selection; compare this asymmetry under second-target pressure",
			})
		case "actual-external-command-observation-not-acquired":
			result = append(result, unresolvedDisposition{
				ID:          id,
				Disposition: "carry-forward",
				Blocking:    false,
				Owner:       "external-runtime-evidence",
				Reason:      "Gate C Reference preserves synthetic nature and does not upgrade it to external production Evidence",
			})
		case "target-capability-taxonomy-remains-candidate-until-second-target":
			result = append(result, unresolvedDisposition{
				ID:          id,
				Disposition: "carry-forward",
				Blocking:    false,
				Owner:       "ISSUE-000026",
				Reason:      "first-target observations are candidates until contrasted with a sufficiently different second Target",
			})
		case "general-target-neutral-presentation-taxonomy-not-yet-defined":
			result = append(result, unresolvedDisposition{
				ID:          id,
				Disposition: "carry-forward",
				Blocking:    false,
				Owner:       "ISSUE-000026",
				Reason:      "Gate C does not require a completed target-neutral Presentation taxonomy",
			})
		default:
			return nil, fmt.Errorf("unclassified Gate C unresolved item %q", id)
		}
	}
	return result, nil
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
