package operations

import (
	"fmt"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// SubjectSources is a transport-neutral, read-only source trace. It makes no
// claim that the subject is unique or exists in the canonical World.
type SubjectSources struct {
	Subject    sourceindex.Subject     `json:"subject"`
	Artifacts  []sourceindex.Artifact  `json:"artifacts"`
	References []sourceindex.Reference `json:"references"`
}

func TraceSubjectSources(root string, subject sourceindex.Subject) (SubjectSources, error) {
	if strings.TrimSpace(subject.Kind) == "" || strings.TrimSpace(subject.ID) == "" {
		return SubjectSources{}, fmt.Errorf("subject kind and id are required")
	}
	loaded, err := workspace.Load(root)
	if err != nil { return SubjectSources{}, err }
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil { return SubjectSources{}, err }
	refs := idx.ReferencesFor(subject)
	paths := make(map[string]bool, len(refs))
	for _, ref := range refs { paths[ref.ArtifactPath] = true }
	artifacts := make([]sourceindex.Artifact, 0, len(paths))
	for _, artifact := range idx.Artifacts {
		if paths[artifact.Path] { artifacts = append(artifacts, artifact) }
	}
	return SubjectSources{Subject:subject, Artifacts:artifacts, References:refs}, nil
}
