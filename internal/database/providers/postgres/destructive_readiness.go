package postgres

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/database/schemadiff"
)

type DropTableReadinessReason string

const (
	DropTablePartitionTreePresent        DropTableReadinessReason = "partition-tree-present"
	DropTableTargetIsPartitionChild      DropTableReadinessReason = "target-is-partition-child"
	DropTablePartitionObservationMissing DropTableReadinessReason = "partition-observation-unavailable"
)

type DropTableReadiness struct {
	Table            string                   `json:"table"`
	Executable       bool                     `json:"executable"`
	Reason           DropTableReadinessReason `json:"reason,omitempty"`
	RelatedRelations []RelationIdentity       `json:"relatedRelations,omitempty"`
}

type DestructiveReadiness struct {
	Executable bool                 `json:"executable"`
	DropTables []DropTableReadiness `json:"dropTables,omitempty"`
}

type DestructiveReadinessError struct {
	Readiness DestructiveReadiness
}

func (e *DestructiveReadinessError) Error() string {
	blocked := 0
	for _, drop := range e.Readiness.DropTables {
		if !drop.Executable {
			blocked++
		}
	}
	return fmt.Sprintf("postgres destructive migration readiness blocked %d drop-table operation(s)", blocked)
}

// AssessDestructiveReadiness evaluates PostgreSQL-specific safety facts without
// changing the provider-neutral Schema Diff. Only destructive table removal is
// assessed here; non-destructive changes remain the renderer's responsibility.
func (o SchemaObservation) AssessDestructiveReadiness(diff schemadiff.Diff) (DestructiveReadiness, error) {
	readiness := DestructiveReadiness{Executable: true}
	hasDropTable := false
	for _, change := range diff.Changes {
		if change.Kind == schemadiff.DropTable {
			hasDropTable = true
			break
		}
	}
	if !hasDropTable {
		return readiness, nil
	}
	if o.PartitionObservationComplete && string(o.Snapshot.Database) != diff.Database {
		return DestructiveReadiness{}, fmt.Errorf("postgres partition observation database %s differs from diff database %s", o.Snapshot.Database, diff.Database)
	}

	childToParent := map[string]PartitionRelation{}
	parentToChildren := map[string][]RelationIdentity{}
	for _, edge := range o.Partitions {
		childKey := key(edge.Child.Schema, edge.Child.Name)
		parentKey := key(edge.Parent.Schema, edge.Parent.Name)
		if prior, exists := childToParent[childKey]; exists {
			if key(prior.Parent.Schema, prior.Parent.Name) != parentKey {
				return DestructiveReadiness{}, fmt.Errorf("postgres partition child %s has multiple observed parents", childKey)
			}
			continue
		}
		childToParent[childKey] = edge
		parentToChildren[parentKey] = append(parentToChildren[parentKey], edge.Child)
	}

	for _, change := range diff.Changes {
		if change.Kind != schemadiff.DropTable {
			continue
		}
		assessment := DropTableReadiness{Table: change.Table, Executable: true}
		if !o.PartitionObservationComplete {
			assessment.Executable = false
			assessment.Reason = DropTablePartitionObservationMissing
			readiness.Executable = false
			readiness.DropTables = append(readiness.DropTables, assessment)
			continue
		}

		target, err := relationIdentityFromQualifiedName(change.Table)
		if err != nil {
			return DestructiveReadiness{}, fmt.Errorf("drop-table %s: %w", change.Table, err)
		}
		targetKey := key(target.Schema, target.Name)
		if edge, isChild := childToParent[targetKey]; isChild {
			assessment.Executable = false
			assessment.Reason = DropTableTargetIsPartitionChild
			assessment.RelatedRelations = []RelationIdentity{edge.Parent}
			readiness.Executable = false
			readiness.DropTables = append(readiness.DropTables, assessment)
			continue
		}

		descendants := partitionDescendants(targetKey, parentToChildren)
		if len(descendants) > 0 {
			assessment.Executable = false
			assessment.Reason = DropTablePartitionTreePresent
			assessment.RelatedRelations = descendants
			readiness.Executable = false
		}
		readiness.DropTables = append(readiness.DropTables, assessment)
	}

	return readiness, nil
}

// RenderMigrationSQLWithObservation is the destructive-safe PostgreSQL render
// entry point. A DropTable reaches the existing SQL renderer only after a
// complete provider-owned partition observation proves that the target is not
// a partition child and has no attached descendants.
func RenderMigrationSQLWithObservation(diff schemadiff.Diff, observation SchemaObservation) ([]string, error) {
	readiness, err := observation.AssessDestructiveReadiness(diff)
	if err != nil {
		return nil, err
	}
	if !readiness.Executable {
		return nil, &DestructiveReadinessError{Readiness: readiness}
	}
	return RenderMigrationSQL(diff)
}

func relationIdentityFromQualifiedName(value string) (RelationIdentity, error) {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return RelationIdentity{}, fmt.Errorf("relation identity %q must be schema-qualified", value)
	}
	return RelationIdentity{Schema: strings.TrimSpace(parts[0]), Name: strings.TrimSpace(parts[1])}, nil
}

func partitionDescendants(root string, byParent map[string][]RelationIdentity) []RelationIdentity {
	seen := map[string]struct{}{root: {}}
	queue := []string{root}
	var out []RelationIdentity
	for len(queue) > 0 {
		parent := queue[0]
		queue = queue[1:]
		children := append([]RelationIdentity(nil), byParent[parent]...)
		sort.Slice(children, func(i, j int) bool {
			return key(children[i].Schema, children[i].Name) < key(children[j].Schema, children[j].Name)
		})
		for _, child := range children {
			childKey := key(child.Schema, child.Name)
			if _, exists := seen[childKey]; exists {
				continue
			}
			seen[childKey] = struct{}{}
			out = append(out, child)
			queue = append(queue, childKey)
		}
	}
	return out
}
