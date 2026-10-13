package main

// A read-only consumer of TEI's existing Workspace, Source Index and typed
// subject/knowledge context. It never promotes a Source declaration to
// Shared Acceptance, business-rule meaning, or runtime execution authority.

import (
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/knowledgecontext"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/internal/workspace"
)

type SourceWitness struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
	Role        string `json:"role"`
}

type WorkspaceAssessment struct {
	SchemaVersion    string          `json:"schemaVersion"`
	Authority        string          `json:"authority"`
	Application      string          `json:"application"`
	SubjectKind      string          `json:"subjectKind"`
	SubjectID        string          `json:"subjectId"`
	SourceStatus     string          `json:"sourceStatus"`
	RuleMeaning      string          `json:"ruleMeaning"`
	ExecutionAllowed bool            `json:"executionAllowed"`
	Witnesses        []SourceWitness `json:"witnesses"`
	Unresolved       []string        `json:"unresolved"`
}

// InspectWorkspace only verifies whether a real application-pattern has one
// unambiguous declaration and current Source Index provenance. The subject is
// typed using TEI's existing subject contract. It deliberately does not infer
// order approval from the existence/name of a submit-order Pattern.
func InspectWorkspace(root, application, kind, id string) (WorkspaceAssessment, error) {
	result := WorkspaceAssessment{
		SchemaVersion:    "tei.reference.purpose-preflight-source/v0",
		Authority:        "read-only-derived-projection",
		Application:      application,
		SubjectKind:      kind,
		SubjectID:        id,
		SourceStatus:     "unresolved",
		RuleMeaning:      "not-established",
		ExecutionAllowed: false,
		Witnesses:        []SourceWitness{},
		Unresolved:       []string{},
	}
	if root == "" || application == "" || kind != "application-pattern" {
		return result, fmt.Errorf("workspace mode requires -workspace, -application and -subject-kind application-pattern")
	}
	if _, err := core.ParseCanonicalID(application); err != nil {
		return result, fmt.Errorf("invalid Application ID: %w", err)
	}
	if _, err := core.ParseCanonicalID(id); err != nil {
		return result, fmt.Errorf("invalid application Pattern ID: %w", err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return result, fmt.Errorf("load workspace: %w", err)
	}
	sources, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return result, fmt.Errorf("build source index: %w", err)
	}
	count := 0
	for _, app := range loaded.World.Applications {
		for _, pattern := range app.Patterns {
			if string(pattern.ID) == id {
				count++
			}
		}
	}
	switch {
	case count == 0:
		result.Unresolved = append(result.Unresolved, "application-pattern-not-declared")
		return result, nil
	case count > 1:
		result.Unresolved = append(result.Unresolved, "ambiguous-application-pattern")
		return result, nil
	}

	ctx := knowledgecontext.Build(loaded.World, sources, subject.Subject{Kind: kind, ID: id})
	qualification, err := knowledgecontext.QualifyWithSources(ctx, sources, knowledgecontext.QualificationRequirements{
		ExpectedKind: kind,
		Application:  application,
	})
	if err != nil {
		return result, fmt.Errorf("qualify declared subject: %w", err)
	}
	if ctx.ApplicationPattern == nil {
		result.Unresolved = append(result.Unresolved, "pattern-not-resolved-in-derived-index")
		return result, nil
	}
	fingerprints := map[string]string{}
	for _, artifact := range ctx.Artifacts {
		fingerprints[artifact.Path] = artifact.Fingerprint
	}
	for _, reference := range ctx.References {
		if reference.Role != sourceindex.RoleDefinition {
			continue
		}
		fingerprint := fingerprints[reference.ArtifactPath]
		if fingerprint == "" {
			continue
		}
		result.Witnesses = append(result.Witnesses, SourceWitness{
			Path: reference.ArtifactPath, Fingerprint: fingerprint, Role: string(reference.Role),
		})
	}
	sort.Slice(result.Witnesses, func(i, j int) bool {
		a, b := result.Witnesses[i], result.Witnesses[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Role != b.Role {
			return a.Role < b.Role
		}
		return a.Fingerprint < b.Fingerprint
	})
	if qualification.Status != knowledgecontext.QualificationApplicable {
		for _, reason := range qualification.Reasons {
			result.Unresolved = append(result.Unresolved, reason.Code)
		}
		return result, nil
	}
	if len(result.Witnesses) == 0 {
		result.Unresolved = append(result.Unresolved, "current-definition-source-not-established")
		return result, nil
	}
	result.SourceStatus = "traceable-source-declaration"
	// Source Index fingerprints attest bytes, not human acceptance or
	// equivalence of a proposed Business Procedure/approval rule.
	result.Unresolved = append(result.Unresolved, "business-rule-acceptance-not-established")
	return result, nil
}
