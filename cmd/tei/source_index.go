package main

import (
	"fmt"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

type sourceIndexSubjectView struct {
	Subject    sourceindex.Subject     `json:"subject"`
	Artifacts  []sourceindex.Artifact  `json:"artifacts,omitempty"`
	References []sourceindex.Reference `json:"references"`
}

func buildSourceIndexInspection(root, subjectKind, subjectID string) (any, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("source-index requires -workspace")
	}
	kind := strings.TrimSpace(subjectKind)
	id := strings.TrimSpace(subjectID)
	if (kind == "") != (id == "") {
		return nil, fmt.Errorf("-subject-kind and -subject-id must be supplied together")
	}

	result, err := workspace.Load(root)
	if err != nil {
		return nil, err
	}
	index, err := workspace.BuildSourceIndex(root, result)
	if err != nil {
		return nil, err
	}
	if kind == "" {
		return index, nil
	}

	subject := sourceindex.Subject{Kind: kind, ID: id}
	references := index.ReferencesFor(subject)
	paths := make(map[string]struct{}, len(references))
	for _, ref := range references {
		paths[ref.ArtifactPath] = struct{}{}
	}
	artifacts := make([]sourceindex.Artifact, 0, len(paths))
	for _, artifact := range index.Artifacts {
		if _, ok := paths[artifact.Path]; ok {
			artifacts = append(artifacts, artifact)
		}
	}
	return sourceIndexSubjectView{
		Subject:    subject,
		Artifacts:  artifacts,
		References: references,
	}, nil
}
