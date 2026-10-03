package knowledgecontext

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/subject"
)

type QualificationStatus string

const (
	QualificationApplicable   QualificationStatus = "applicable"
	QualificationIncompatible QualificationStatus = "incompatible"
	QualificationUnknown      QualificationStatus = "unknown"
)

type DependencyRequirement struct {
	Direction string `json:"direction"`
	Relation  string `json:"relation"`
	Kind      string `json:"kind"`
	ID        string `json:"id"`
	Alias     string `json:"alias,omitempty"`
}

type QualificationRequirements struct {
	ExpectedKind         string                  `json:"expectedKind"`
	Application          string                  `json:"application,omitempty"`
	RequiredDependencies []DependencyRequirement `json:"requiredDependencies,omitempty"`
}

type QualificationReason struct {
	Code   string `json:"code"`
	Detail string `json:"detail,omitempty"`
}

type QualificationEvidence struct {
	SubjectKind       string `json:"subjectKind"`
	Subject           string `json:"subject"`
	Source            string `json:"source"`
	SourceFingerprint string `json:"sourceFingerprint"`
}

type QualificationResult struct {
	Candidate          subject.Subject           `json:"candidate"`
	Status             QualificationStatus       `json:"status"`
	Reasons            []QualificationReason     `json:"reasons,omitempty"`
	Requirements       QualificationRequirements `json:"requirements"`
	Evidence           []QualificationEvidence   `json:"evidence,omitempty"`
	DependencyEvidence []QualificationEvidence   `json:"dependencyEvidence,omitempty"`
	AuthorityLimits    []string                  `json:"authorityLimits"`
}

func Qualify(ctx Context, requirements QualificationRequirements) (QualificationResult, error) {
	requirements, err := NormalizeQualificationRequirements(requirements)
	if err != nil {
		return QualificationResult{}, err
	}

	result := QualificationResult{
		Candidate:    ctx.Subject,
		Requirements: requirements,
		AuthorityLimits: []string{
			"only-explicit-requirements-were-evaluated",
			"qualification-is-not-acceptance",
			"qualification-is-not-write-or-runtime-authority",
		},
	}
	incompatible := make([]QualificationReason, 0)
	unknown := make([]QualificationReason, 0)

	if ctx.Subject.Kind != requirements.ExpectedKind {
		incompatible = append(incompatible, QualificationReason{
			Code:   "expected-kind-mismatch",
			Detail: fmt.Sprintf("have=%s want=%s", ctx.Subject.Kind, requirements.ExpectedKind),
		})
	}
	authoritativeSource := hasAuthoritativeSourceReference(ctx.References)
	if !authoritativeSource {
		unknown = append(unknown, QualificationReason{Code: "authoritative-source-reference-unavailable"})
	}
	for _, unresolved := range ctx.Unresolved {
		unknown = append(unknown, QualificationReason{Code: "context-unresolved", Detail: unresolved})
	}

	if requirements.Application != "" {
		switch {
		case ctx.ApplicationPattern == nil:
			unknown = append(unknown, QualificationReason{Code: "application-scope-unavailable"})
		case !authoritativeSource:
			unknown = append(unknown, QualificationReason{Code: "application-scope-unverified"})
		case string(ctx.ApplicationPattern.Application) != requirements.Application:
			incompatible = append(incompatible, QualificationReason{
				Code:   "application-scope-mismatch",
				Detail: fmt.Sprintf("have=%s want=%s", ctx.ApplicationPattern.Application, requirements.Application),
			})
		}
	}

	for _, requirement := range requirements.RequiredDependencies {
		if hasRequiredDependency(ctx, requirement) {
			continue
		}
		reason := QualificationReason{Detail: dependencyRequirementKey(requirement)}
		if !authoritativeSource || len(ctx.Unresolved) != 0 {
			reason.Code = "required-dependency-unresolved"
			unknown = append(unknown, reason)
		} else {
			reason.Code = "required-dependency-missing"
			incompatible = append(incompatible, reason)
		}
	}

	result.Reasons = append(result.Reasons, incompatible...)
	result.Reasons = append(result.Reasons, unknown...)
	sortQualificationReasons(result.Reasons)
	switch {
	case len(incompatible) != 0:
		result.Status = QualificationIncompatible
	case len(unknown) != 0:
		result.Status = QualificationUnknown
	default:
		result.Evidence = qualificationEvidence(ctx)
		if len(result.Evidence) == 0 {
			result.Status = QualificationUnknown
			result.Reasons = append(result.Reasons, QualificationReason{Code: "exact-authoritative-evidence-unavailable"})
			sortQualificationReasons(result.Reasons)
		} else {
			result.Status = QualificationApplicable
		}
	}
	return result, nil
}

func NormalizeQualificationRequirements(requirements QualificationRequirements) (QualificationRequirements, error) {
	requirements.ExpectedKind = strings.TrimSpace(requirements.ExpectedKind)
	requirements.Application = strings.TrimSpace(requirements.Application)
	if requirements.ExpectedKind == "" {
		return QualificationRequirements{}, fmt.Errorf("qualification expectedKind is required")
	}
	if requirements.Application != "" && requirements.ExpectedKind != "application-pattern" {
		return QualificationRequirements{}, fmt.Errorf("qualification application scope requires expectedKind=application-pattern")
	}

	seen := map[string]struct{}{}
	for i := range requirements.RequiredDependencies {
		dep := &requirements.RequiredDependencies[i]
		dep.Direction = strings.ToLower(strings.TrimSpace(dep.Direction))
		dep.Relation = strings.TrimSpace(dep.Relation)
		dep.Kind = strings.TrimSpace(dep.Kind)
		dep.ID = strings.TrimSpace(dep.ID)
		dep.Alias = strings.TrimSpace(dep.Alias)
		if dep.Direction != "outgoing" && dep.Direction != "incoming" {
			return QualificationRequirements{}, fmt.Errorf("qualification dependency direction must be outgoing or incoming")
		}
		if dep.Relation == "" || dep.Kind == "" || dep.ID == "" {
			return QualificationRequirements{}, fmt.Errorf("qualification dependency requires relation, kind, and id")
		}
		key := dependencyRequirementKey(*dep)
		if _, ok := seen[key]; ok {
			return QualificationRequirements{}, fmt.Errorf("qualification dependency is duplicated: %s", key)
		}
		seen[key] = struct{}{}
	}
	sort.Slice(requirements.RequiredDependencies, func(i, j int) bool {
		return dependencyRequirementKey(requirements.RequiredDependencies[i]) < dependencyRequirementKey(requirements.RequiredDependencies[j])
	})
	return requirements, nil
}

func hasAuthoritativeSourceReference(references []sourceindex.Reference) bool {
	for _, ref := range references {
		if ref.Role == sourceindex.RoleDefinition || ref.Role == sourceindex.RoleAuthoring {
			return true
		}
	}
	return false
}

func hasRequiredDependency(ctx Context, requirement DependencyRequirement) bool {
	edges := ctx.Outgoing
	if requirement.Direction == "incoming" {
		edges = ctx.Incoming
	}
	for _, edge := range edges {
		if edge.Relation != requirement.Relation {
			continue
		}
		other := edge.To
		if requirement.Direction == "incoming" {
			other = edge.From
		}
		if other.Kind != requirement.Kind || other.ID != requirement.ID {
			continue
		}
		if requirement.Alias != "" && string(edge.Alias) != requirement.Alias {
			continue
		}
		return true
	}
	return false
}

func dependencyRequirementKey(requirement DependencyRequirement) string {
	return strings.Join([]string{
		requirement.Direction,
		requirement.Relation,
		requirement.Kind,
		requirement.ID,
		requirement.Alias,
	}, "|")
}

func sortQualificationReasons(reasons []QualificationReason) {
	sort.Slice(reasons, func(i, j int) bool {
		if reasons[i].Code != reasons[j].Code {
			return reasons[i].Code < reasons[j].Code
		}
		return reasons[i].Detail < reasons[j].Detail
	})
}

func qualificationEvidence(ctx Context) []QualificationEvidence {
	artifacts := make(map[string]string, len(ctx.Artifacts))
	for _, artifact := range ctx.Artifacts {
		artifacts[artifact.Path] = artifact.Fingerprint
	}
	var out []QualificationEvidence
	seen := map[string]struct{}{}
	for _, ref := range ctx.References {
		if ref.Role != sourceindex.RoleDefinition && ref.Role != sourceindex.RoleAuthoring {
			continue
		}
		fingerprint := artifacts[ref.ArtifactPath]
		if fingerprint == "" {
			continue
		}
		item := QualificationEvidence{
			SubjectKind:       ctx.Subject.Kind,
			Subject:           ctx.Subject.ID,
			Source:            ref.ArtifactPath,
			SourceFingerprint: fingerprint,
		}
		key := fmt.Sprintf("%q|%q|%q|%q", item.SubjectKind, item.Subject, item.Source, item.SourceFingerprint)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		if out[i].SubjectKind != out[j].SubjectKind {
			return out[i].SubjectKind < out[j].SubjectKind
		}
		return out[i].Subject < out[j].Subject
	})
	return out
}


func QualifyWithSources(ctx Context, sources sourceindex.Index, requirements QualificationRequirements) (QualificationResult, error) {
	result, err := Qualify(ctx, requirements)
	if err != nil || result.Status != QualificationApplicable {
		return result, err
	}

	var dependencyEvidence []QualificationEvidence
	for _, requirement := range result.Requirements.RequiredDependencies {
		target := subject.Subject{Kind: requirement.Kind, ID: requirement.ID}
		current := authoritativeEvidenceForSubject(sources, target)
		if len(current) == 0 {
			result.Status = QualificationUnknown
			result.Reasons = append(result.Reasons, QualificationReason{
				Code:   "required-dependency-authoritative-evidence-unavailable",
				Detail: dependencyRequirementKey(requirement),
			})
			sortQualificationReasons(result.Reasons)
			result.Evidence = nil
			result.DependencyEvidence = nil
			return result, nil
		}
		dependencyEvidence = append(dependencyEvidence, current...)
	}
	result.DependencyEvidence = normalizeQualificationEvidence(dependencyEvidence)
	return result, nil
}

func authoritativeEvidenceForSubject(sources sourceindex.Index, target subject.Subject) []QualificationEvidence {
	artifacts := make(map[string]string, len(sources.Artifacts))
	for _, artifact := range sources.Artifacts {
		artifacts[artifact.Path] = artifact.Fingerprint
	}
	var out []QualificationEvidence
	for _, ref := range sources.ReferencesFor(target) {
		if ref.Role != sourceindex.RoleDefinition && ref.Role != sourceindex.RoleAuthoring {
			continue
		}
		fingerprint := artifacts[ref.ArtifactPath]
		if fingerprint == "" {
			continue
		}
		out = append(out, QualificationEvidence{
			SubjectKind:       target.Kind,
			Subject:           target.ID,
			Source:            ref.ArtifactPath,
			SourceFingerprint: fingerprint,
		})
	}
	return normalizeQualificationEvidence(out)
}

func normalizeQualificationEvidence(values []QualificationEvidence) []QualificationEvidence {
	seen := map[string]struct{}{}
	out := make([]QualificationEvidence, 0, len(values))
	for _, item := range values {
		key := fmt.Sprintf("%q|%q|%q|%q", item.SubjectKind, item.Subject, item.Source, item.SourceFingerprint)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SubjectKind != out[j].SubjectKind {
			return out[i].SubjectKind < out[j].SubjectKind
		}
		if out[i].Subject != out[j].Subject {
			return out[i].Subject < out[j].Subject
		}
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		return out[i].SourceFingerprint < out[j].SourceFingerprint
	})
	return out
}
