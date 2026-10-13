package main

// This is a read-only witness for a declared semantic Validation. A declaration
// and its Source Index provenance are not a human Acceptance Decision or a
// capability to authorize the corresponding business operation.

import (
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/knowledgecontext"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/internal/workspace"
)

// InspectValidation demonstrates the distinction between a machine-readable,
// source-traceable business constraint and verified human acceptance of that
// constraint. Only existing shared Workspace/Knowledge Context primitives are
// used. No second Decision Index or Source Registry is introduced.
func InspectValidation(root, applicationID, validationID string) (WorkspaceAssessment, error) {
	result := WorkspaceAssessment{
		SchemaVersion:    "tei.reference.purpose-preflight-source/v0",
		Authority:        "read-only-derived-projection",
		Application:      applicationID,
		SubjectKind:      "semantic-validation",
		SubjectID:        validationID,
		SourceStatus:     "unresolved",
		RuleMeaning:      "not-established",
		ExecutionAllowed: false,
		Witnesses:        []SourceWitness{},
		Unresolved:       []string{},
	}
	if root == "" {
		return result, fmt.Errorf("workspace root is required")
	}
	if _, err := core.ParseCanonicalID(applicationID); err != nil {
		return result, fmt.Errorf("invalid Application ID: %w", err)
	}
	if _, err := core.ParseCanonicalID(validationID); err != nil {
		return result, fmt.Errorf("invalid semantic Validation ID: %w", err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return result, fmt.Errorf("load workspace: %w", err)
	}
	sources, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return result, fmt.Errorf("build source index: %w", err)
	}

	validationCount := 0
	var namespaceID string
	for _, ns := range loaded.World.Semantics {
		for _, v := range ns.Validations {
			if string(v.ID) == validationID {
				validationCount++
				namespaceID = string(ns.ID)
			}
		}
	}
	if validationCount != 1 {
		if validationCount == 0 {
			result.Unresolved = append(result.Unresolved, "semantic-validation-not-declared")
		} else {
			result.Unresolved = append(result.Unresolved, "ambiguous-semantic-validation")
		}
		return result, nil
	}
	applicationCount := 0
	usesRule := false
	usesNamespace := false
	for _, app := range loaded.World.Applications {
		if string(app.ID) != applicationID {
			continue
		}
		applicationCount++
		for _, ref := range app.Validations {
			if string(ref) == validationID {
				usesRule = true
			}
		}
		for _, ref := range app.SemanticUses {
			if string(ref) == namespaceID {
				usesNamespace = true
			}
		}
	}
	if applicationCount != 1 || !usesRule || !usesNamespace {
		result.Unresolved = append(result.Unresolved, "application-validation-use-not-established")
		return result, nil
	}
	ctx := knowledgecontext.Build(loaded.World, sources, subject.Subject{Kind: "semantic-validation", ID: validationID})
	if ctx.SemanticElement == nil {
		result.Unresolved = append(result.Unresolved, "validation-not-resolved-in-derived-index")
		return result, nil
	}
	// The shared qualification API accepts Application scope only for
	// application-pattern. Application Validation usage was verified above.
	qualification, err := knowledgecontext.QualifyWithSources(ctx, sources, knowledgecontext.QualificationRequirements{
		ExpectedKind: "semantic-validation",
	})
	if err != nil {
		return result, fmt.Errorf("qualify validation source: %w", err)
	}
	if qualification.Status != knowledgecontext.QualificationApplicable {
		for _, reason := range qualification.Reasons {
			result.Unresolved = append(result.Unresolved, reason.Code)
		}
		return result, nil
	}
	fingerprints := map[string]string{}
	for _, artifact := range ctx.Artifacts {
		fingerprints[artifact.Path] = artifact.Fingerprint
	}
	for _, ref := range ctx.References {
		if ref.Role != sourceindex.RoleDefinition {
			continue
		}
		fp := fingerprints[ref.ArtifactPath]
		if fp == "" {
			continue
		}
		result.Witnesses = append(result.Witnesses, SourceWitness{Path: ref.ArtifactPath, Fingerprint: fp, Role: string(ref.Role)})
	}
	sort.Slice(result.Witnesses, func(i, j int) bool {
		if result.Witnesses[i].Path != result.Witnesses[j].Path {
			return result.Witnesses[i].Path < result.Witnesses[j].Path
		}
		return result.Witnesses[i].Fingerprint < result.Witnesses[j].Fingerprint
	})
	if len(result.Witnesses) == 0 {
		result.Unresolved = append(result.Unresolved, "current-definition-source-not-established")
		return result, nil
	}
	result.SourceStatus = "traceable-source-declaration"
	result.RuleMeaning = "canonical-validation-declared"
	result.Unresolved = append(result.Unresolved, "human-acceptance-not-verified", "runtime-validation-not-evaluated")
	return result, nil
}
