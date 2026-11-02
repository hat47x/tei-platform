package artifact

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

const (
	// TableDefinitionRuleID and TableDefinitionRuleVersion identify the rules
	// this producer applies. A change of output shape or of the unresolved
	// policy must change the version.
	TableDefinitionRuleID      = "tei.artifact.table-definition"
	TableDefinitionRuleVersion = "v1"

	// TableDefinitionGenerator identifies the implementation. It is part of the
	// artifact identity, so two implementations of the same rules stay distinct.
	TableDefinitionGenerator = "tei-artifact-tabledef/v1"
)

const tableDefinitionRulePolicy = `table-definition/v1
lines: table, data, column (<column> <attribute> <storage> <nullability> [pk]), primary-key, unique, foreign-key, index
order: columns in the authored persistence order, indexes by name
unresolved: unmapped semantic attribute, missing primary key`

// TableDefinitionInput is everything the producer reads. It holds no time and no
// environment value, so the same input always renders the same bytes.
type TableDefinitionInput struct {
	InputRevision string
	Mapping       persistence.Mapping
	Data          semantic.Data
}

// RenderTableDefinition renders one table definition from one persistence
// mapping. A missing column or key is reported as unresolved instead of being
// filled with a default.
func RenderTableDefinition(input TableDefinitionInput) (Artifact, error) {
	return RenderTableDefinitionWith(input, nil)
}

// RenderTableDefinitionWith also resolves referenced tables, so the definition
// shows the foreign keys the mapping declares. A nil resolver keeps the
// definition to one table.
func RenderTableDefinitionWith(input TableDefinitionInput, tableOf func(core.CanonicalID) (string, bool)) (Artifact, error) {
	content, unresolved := renderTableDefinition(input.Mapping, input.Data, tableOf)
	return New(
		KindTableDefinition,
		Rule{
			ID:          TableDefinitionRuleID,
			Version:     TableDefinitionRuleVersion,
			Fingerprint: sourceindex.Fingerprint([]byte(tableDefinitionRulePolicy)),
		},
		TableDefinitionGenerator,
		input.InputRevision,
		content,
		unresolved,
	)
}

func renderTableDefinition(mapping persistence.Mapping, element semantic.Data, tableOf func(core.CanonicalID) (string, bool)) (string, []Unresolved) {
	var b strings.Builder
	fmt.Fprintf(&b, "table %s\n", mapping.Table)
	fmt.Fprintf(&b, "data %s\n", mapping.SemanticData)

	mapped := map[core.CanonicalID]struct{}{}
	for _, column := range mapping.Columns {
		mapped[column.Attribute] = struct{}{}
		fmt.Fprintf(&b, "column %s %s %s %s%s\n",
			column.Column,
			column.Attribute,
			column.Storage,
			column.Nullability,
			primaryKeyMarker(mapping.PrimaryKey, column.Attribute),
		)
	}

	if len(mapping.PrimaryKey) == 0 {
		b.WriteString("primary-key (none)\n")
	} else {
		keys := make([]string, 0, len(mapping.PrimaryKey))
		for _, attribute := range mapping.PrimaryKey {
			keys = append(keys, string(attribute))
		}
		fmt.Fprintf(&b, "primary-key %s\n", strings.Join(keys, ","))
	}

	uniques := append([]persistence.UniqueConstraintMapping(nil), mapping.UniqueConstraints...)
	sort.Slice(uniques, func(i, j int) bool { return uniques[i].Name < uniques[j].Name })
	for _, unique := range uniques {
		columns := make([]string, 0, len(unique.Attributes))
		for _, attribute := range unique.Attributes {
			columns = append(columns, string(attribute))
		}
		fmt.Fprintf(&b, "unique %s %s\n", unique.Name, strings.Join(columns, ","))
	}

	foreignKeys := append([]persistence.ForeignKeyMapping(nil), mapping.ForeignKeys...)
	sort.Slice(foreignKeys, func(i, j int) bool { return foreignKeys[i].Name < foreignKeys[j].Name })
	for _, foreignKey := range foreignKeys {
		table := string(foreignKey.ReferencedMapping)
		if tableOf != nil {
			if resolved, ok := tableOf(foreignKey.ReferencedMapping); ok {
				table = resolved
			}
		}
		columns := make([]string, 0, len(foreignKey.Attributes))
		for _, attribute := range foreignKey.Attributes {
			columns = append(columns, string(attribute))
		}
		referenced := make([]string, 0, len(foreignKey.ReferencedAttributes))
		for _, attribute := range foreignKey.ReferencedAttributes {
			referenced = append(referenced, string(attribute))
		}
		fmt.Fprintf(&b, "foreign-key %s %s -> %s(%s)\n", foreignKey.Name, strings.Join(columns, ","), table, strings.Join(referenced, ","))
	}

	indexes := append([]persistence.IndexMapping(nil), mapping.Indexes...)
	sort.Slice(indexes, func(i, j int) bool { return indexes[i].Name < indexes[j].Name })
	for _, index := range indexes {
		attributes := make([]string, 0, len(index.Attributes))
		for _, attribute := range index.Attributes {
			attributes = append(attributes, string(attribute))
		}
		fmt.Fprintf(&b, "index %s %s\n", index.Name, strings.Join(attributes, ","))
	}

	var unresolved []Unresolved
	for _, attribute := range element.Attributes {
		if _, ok := mapped[attribute.ID]; ok {
			continue
		}
		unresolved = append(unresolved, Unresolved{
			Subject: string(attribute.ID),
			Reason:  "missing-input",
			Detail:  "semantic attribute is not mapped to a column",
		})
	}
	if len(mapping.PrimaryKey) == 0 {
		unresolved = append(unresolved, Unresolved{
			Subject: string(mapping.SemanticData),
			Reason:  "missing-semantics",
			Detail:  "no primary key is declared for the table",
		})
	}
	return b.String(), unresolved
}

func primaryKeyMarker(primaryKey []core.CanonicalID, attribute core.CanonicalID) string {
	for _, key := range primaryKey {
		if key == attribute {
			return " pk"
		}
	}
	return ""
}
