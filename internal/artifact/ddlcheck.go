package artifact

import (
	"fmt"

	"tei/internal/core"

	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
	"tei/internal/database/tabledef"
	"tei/internal/semantic"
)

// TableDefinitionSnapshot renders one persistence mapping as a physical schema
// snapshot. The derivation lives in internal/database/tabledef so a projection
// that must sit in internal/operations (which artifact depends on, and therefore
// cannot import artifact) reads the same source instead of a second copy.
func TableDefinitionSnapshot(mapping persistence.Mapping, element semantic.Data) physical.SchemaSnapshot {
	return tabledef.TableDefinitionSnapshot(mapping, element)
}

// TableDefinitionSnapshotWith also resolves the referenced mappings, so foreign
// keys carry the table and columns they point at. A nil resolver leaves them to
// the declared identifiers.
func TableDefinitionSnapshotWith(mapping persistence.Mapping, element semantic.Data, resolve func(core.CanonicalID) (persistence.Mapping, bool)) physical.SchemaSnapshot {
	return tabledef.TableDefinitionSnapshotWith(mapping, element, resolve)
}

// CompareTableDefinition compares the schema the table definition implies
// against a target snapshot (a desired schema or an observed one). No changes
// means the artifact and the schema agree; the diff is returned, never silently
// repaired.
func CompareTableDefinition(mapping persistence.Mapping, element semantic.Data, target physical.SchemaSnapshot) (schemadiff.Diff, error) {
	return CompareTableDefinitionWith(mapping, element, target, nil)
}

// CompareTableDefinitionWith resolves referenced mappings, so foreign keys are
// compared by the table and columns they point at.
func CompareTableDefinitionWith(mapping persistence.Mapping, element semantic.Data, target physical.SchemaSnapshot, resolve func(core.CanonicalID) (persistence.Mapping, bool)) (schemadiff.Diff, error) {
	diff, err := schemadiff.Compare(TableDefinitionSnapshotWith(mapping, element, resolve).Normalize(), target.Normalize())
	if err != nil {
		return schemadiff.Diff{}, fmt.Errorf("compare table definition with the target schema: %w", err)
	}
	return diff, nil
}
