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
