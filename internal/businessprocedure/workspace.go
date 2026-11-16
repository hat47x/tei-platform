package businessprocedure

import (
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// DeclaredFromIndex takes the declared targets from a Source Index. Only
// definition-role references are targets: an authoring or derived reference is
// not where the identity is declared, so it must not resolve a step's Function.
//
// Two references for one identity are both returned. The caller's resolver then
// reports the identity as ambiguous, which is the point: the index does not
// choose between declarations and neither does the projection.
func DeclaredFromIndex(index sourceindex.Index) []Declared {
	// An artifact's digest belongs to its path, so index it once.
	fingerprints := make(map[string]string, len(index.Artifacts))
	for _, artifact := range index.Artifacts {
		fingerprints[artifact.Path] = artifact.Fingerprint
	}
	declared := make([]Declared, 0, len(index.References))
	for _, reference := range index.References {
		if reference.Role != sourceindex.RoleDefinition {
			continue
		}
		kind, ok := referenceKindForSubject(reference.Subject.Kind)
		if !ok {
			continue
		}
		declared = append(declared, Declared{
			Kind:     kind,
			ID:       reference.Subject.ID,
			Subject:  reference.Subject.Kind + ":" + reference.Subject.ID,
			Artifact: reference.ArtifactPath,
			Line:     reference.Range.Start.Line,
			Column:   reference.Range.Start.Column,
			Role:     string(reference.Role),
			// Carry the artifact digest, so a resolved reference names not only
			// a path but the bytes the index read at that path.
			Fingerprint: fingerprints[reference.ArtifactPath],
		})
	}
	return declared
}

// DeclaredFromWorkspace builds the declared targets for a loaded Workspace from
// its own Source Index. It reads the bytes Load captured, so a file changed on
// disk after Load cannot change what a reference resolves to.
func DeclaredFromWorkspace(root string, result workspace.Result) ([]Declared, error) {
	index, err := workspace.BuildSourceIndexFromLoad(root, result)
	if err != nil {
		return nil, err
	}
	return DeclaredFromIndex(index), nil
}
