package main

import (
	"fmt"
	"strings"

	"tei/internal/knowledgecontext"
	"tei/internal/subject"
	"tei/internal/workspace"
)

func buildKnowledgeContext(root, subjectKind, subjectID string) (knowledgecontext.Context, error) {
	if strings.TrimSpace(root) == "" {
		return knowledgecontext.Context{}, fmt.Errorf("knowledge-context requires -workspace")
	}
	kind := strings.TrimSpace(subjectKind)
	id := strings.TrimSpace(subjectID)
	if kind == "" || id == "" {
		return knowledgecontext.Context{}, fmt.Errorf("knowledge-context requires -subject-kind and -subject-id")
	}

	loaded, err := workspace.Load(root)
	if err != nil {
		return knowledgecontext.Context{}, err
	}
	sources, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return knowledgecontext.Context{}, err
	}
	return knowledgecontext.Build(
		loaded.World,
		sources,
		subject.Subject{Kind: kind, ID: id},
	), nil
}
func buildKnowledgeDisclosure(
	ctx knowledgecontext.Context,
	selection string,
) (knowledgecontext.DisclosureProjection, error) {
	return knowledgecontext.BuildDisclosure(ctx, []string{selection})
}

func checkKnowledgeDisclosureFingerprint(
	disclosure knowledgecontext.DisclosureProjection,
	expected string,
) error {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return nil
	}
	if disclosure.Status != "complete" {
		return fmt.Errorf(
			"Knowledge Disclosure status is %q; cached disclosure reuse requires complete selected Knowledge",
			disclosure.Status,
		)
	}
	if disclosure.Fingerprint != expected {
		return fmt.Errorf(
			"Knowledge Disclosure fingerprint mismatch: expected=%s current=%s",
			expected,
			disclosure.Fingerprint,
		)
	}
	return nil
}

func checkDataDesignFingerprint(ctx knowledgecontext.Context, expected string) error {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return nil
	}
	if ctx.DataDesign == nil {
		return fmt.Errorf("expected Data Design fingerprint %q, but this subject has no Data Design Context", expected)
	}
	if ctx.DataDesign.ResolutionStatus != "resolved" {
		return fmt.Errorf(
			"Data Design Context resolution status is %q; cached Context reuse requires structurally resolved identity",
			ctx.DataDesign.ResolutionStatus,
		)
	}
	if ctx.DataDesign.EvidenceCoverageStatus != "complete" {
		return fmt.Errorf(
			"Data Design Context evidence coverage is %q; cached Context reuse requires complete Evidence coverage",
			ctx.DataDesign.EvidenceCoverageStatus,
		)
	}
	if ctx.DataDesign.Fingerprint != expected {
		return fmt.Errorf(
			"Data Design Context fingerprint mismatch: expected=%s current=%s",
			expected,
			ctx.DataDesign.Fingerprint,
		)
	}
	return nil
}

func buildKnowledgeQualification(root, subjectKind, subjectID string, requirements knowledgecontext.QualificationRequirements) (knowledgecontext.QualificationResult, error) {
	if strings.TrimSpace(root) == "" {
		return knowledgecontext.QualificationResult{}, fmt.Errorf("knowledge-qualify requires -workspace")
	}
	kind := strings.TrimSpace(subjectKind)
	id := strings.TrimSpace(subjectID)
	if kind == "" || id == "" {
		return knowledgecontext.QualificationResult{}, fmt.Errorf("knowledge-qualify requires -subject-kind and -subject-id")
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return knowledgecontext.QualificationResult{}, err
	}
	sources, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return knowledgecontext.QualificationResult{}, err
	}
	ctx := knowledgecontext.Build(
		loaded.World,
		sources,
		subject.Subject{Kind: kind, ID: id},
	)
	return knowledgecontext.QualifyWithSources(ctx, sources, requirements)
}

func buildKnowledgeQualificationFromFile(root, subjectKind, subjectID, requirementsPath string) (knowledgecontext.QualificationResult, error) {
	if strings.TrimSpace(requirementsPath) == "" {
		return knowledgecontext.QualificationResult{}, fmt.Errorf("knowledge-qualify requires -qualification-requirements")
	}
	var requirements knowledgecontext.QualificationRequirements
	if err := decodeJSONFile(requirementsPath, &requirements); err != nil {
		return knowledgecontext.QualificationResult{}, err
	}
	return buildKnowledgeQualification(root, subjectKind, subjectID, requirements)
}
