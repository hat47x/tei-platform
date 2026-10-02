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
func buildKnowledgeQualification(root, subjectKind, subjectID string, requirements knowledgecontext.QualificationRequirements) (knowledgecontext.QualificationResult, error) {
	ctx, err := buildKnowledgeContext(root, subjectKind, subjectID)
	if err != nil {
		return knowledgecontext.QualificationResult{}, err
	}
	return knowledgecontext.Qualify(ctx, requirements)
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
