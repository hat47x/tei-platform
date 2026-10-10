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

// TraceSubjectSources obtains source references from one Workspace load and
// one Source Index projection. It does not infer a Canonical Subject from paths.
func TraceSubjectSources(root string, subject sourceindex.Subject) (SubjectSources, error) {
	if err := validateTraceSubject(subject); err != nil {
		return SubjectSources{}, err
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return SubjectSources{}, err
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return SubjectSources{}, err
	}
	return TraceSubjectSourcesFromIndex(idx, subject)
}

// TraceSubjectSourcesFromIndex is the same read-only operation over a
// previously projected index. The CLI may request the full index or one
// subject without loading and rebuilding the Workspace a second time.
func TraceSubjectSourcesFromIndex(idx sourceindex.Index, subject sourceindex.Subject) (SubjectSources, error) {
	if err := validateTraceSubject(subject); err != nil {
		return SubjectSources{}, err
	}
	refs := idx.ReferencesFor(subject)
	paths := make(map[string]bool, len(refs))
	for _, ref := range refs {
		paths[ref.ArtifactPath] = true
	}
	artifacts := make([]sourceindex.Artifact, 0, len(paths))
	for _, artifact := range idx.Artifacts {
		if paths[artifact.Path] {
			artifacts = append(artifacts, artifact)
		}
	}
	return SubjectSources{Subject: subject, Artifacts: artifacts, References: refs}, nil
}

func validateTraceSubject(subject sourceindex.Subject) error {
	if strings.TrimSpace(subject.Kind) == "" || strings.TrimSpace(subject.ID) == "" {
		return fmt.Errorf("subject kind and id are required")
	}
	return nil
}
