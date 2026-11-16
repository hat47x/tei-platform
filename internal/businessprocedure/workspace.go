package businessprocedure

import (
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// referenceKindForSubject maps a Source Index subject kind to the reference
// kind this projection resolves. A subject kind with no entry is not a
// reference target here, so a procedure cannot point at it.
func referenceKindForSubject(kind string) (ReferenceKind, bool) {
	switch kind {
	case "semantic-capability":
		return ReferenceCapability, true
	case "semantic-validation":
		return ReferenceValidation, true
	case "application-pattern":
		return ReferencePattern, true
	case "semantic-data":
		return ReferenceData, true
	}
	return "", false
}

// DeclaredFromIndex takes the declared targets from a Source Index. Only
// definition-role references are targets: an authoring or derived reference is
// not where the identity is declared, so it must not resolve a step's Function.
//
// Two references for one identity are both returned. The caller's resolver then
// reports the identity as ambiguous, which is the point: the index does not
// choose between declarations and neither does the projection.
func DeclaredFromIndex(index sourceindex.Index) []Declared {
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
