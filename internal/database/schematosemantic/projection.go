package schematosemantic

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/semantic"
)

// Provenance tracks the origin and derivation basis of a generated semantic
// model from a physical schema snapshot.
type Provenance struct {
	Source             string `json:"source"`                   // e.g. "postgres-introspection", "sqlite-introspection"
	CapturedAt         string `json:"capturedAt"`               // ISO8601 timestamp when schema was read
	SourceRevision     string `json:"sourceRevision,omitempty"` // DB version, migration version, or empty for live
	ContentFingerprint string `json:"contentFingerprint"`       // SHA256 of normalized schema snapshot
	ReasoningBasis     string `json:"reasoningBasis,omitempty"` // How structural candidates were derived
}

// Diagnostic records information that could not be determined or is ambiguous.
type Diagnostic struct {
	Code    string `json:"code"`    // e.g. "ambiguous-relation", "missing-business-key"
	Table   string `json:"table"`   // affected table, empty for global
	Column  string `json:"column"`  // affected column, empty for table-level
	Message string `json:"message"` // human-readable description
}

// SchemaDiffImpact describes the impact of a schema change on the semantic model.
type SchemaDiffImpact struct {
	AddedTables           []string `json:"addedTables,omitempty"`
	DroppedTables         []string `json:"droppedTables,omitempty"`
	AddedColumns          []string `json:"addedColumns,omitempty"`
	DroppedColumns        []string `json:"droppedColumns,omitempty"`
	AlteredColumns        []string `json:"alteredColumns,omitempty"`
	AddedRelations        []string `json:"addedRelations,omitempty"`
	DroppedRelations      []string `json:"droppedRelations,omitempty"`
	AffectedCandidateKeys []string `json:"affectedCandidateKeys,omitempty"`
}

// ProjectionResult contains the generated semantic model plus provenance and diagnostics.
type ProjectionResult struct {
	Namespace   semantic.Namespace `json:"namespace"`
	Provenance  Provenance         `json:"provenance"`
	Diagnostics []Diagnostic       `json:"diagnostics,omitempty"`
}

// Options control aspects of the schema-to-semantic projection.
type Options struct {
	// RequireOwnerAcceptance marks candidate keys derived from physical constraints
	// as needing explicit Owner acceptance rather than treating them as definitive.
	RequireOwnerAcceptance bool

	// GenerateRelationsFromFK creates semantic Relations for every foreign key.
	// When false, FKs are recorded but no Relations are emitted.
	GenerateRelationsFromFK bool
}

// DefaultOptions returns the default projection options.
func DefaultOptions() Options {
	return Options{
		RequireOwnerAcceptance:  true,
		GenerateRelationsFromFK: true,
	}
}

// Project converts a physical schema snapshot into a TEI semantic model namespace.
// It does not mutate the input snapshot.
func Project(snapshot physical.SchemaSnapshot, opts Options) (ProjectionResult, error) {
	normalized := snapshot.Normalize()
	fp, err := fingerprintSchema(normalized)
	if err != nil {
		return ProjectionResult{}, fmt.Errorf("schema fingerprint: %w", err)
	}

	provenance := Provenance{
		Source:             "live-introspection",
		CapturedAt:         normalized.CapturedAt,
		ContentFingerprint: fp,
		ReasoningBasis:     "primary-key-and-unique-constraint-to-candidate-key",
	}

	ns := semantic.Namespace{
		ID: sanitizeNamespaceID(normalized.Database),
	}

	var diagnostics []Diagnostic

	for _, table := range normalized.Tables {
		data, diags := projectTable(table, normalized, opts)
		ns.Data = append(ns.Data, data)
		diagnostics = append(diagnostics, diags...)
	}

	return ProjectionResult{
		Namespace:   ns,
		Provenance:  provenance,
		Diagnostics: diagnostics,
	}, nil
}

func projectTable(table physical.Table, snapshot physical.SchemaSnapshot, opts Options) (semantic.Data, []Diagnostic) {
	data := semantic.Data{
		ID:         sanitizeDataID(table.QualifiedName()),
		Symbol:     table.Name,
		Attributes: make([]semantic.Attribute, 0, len(table.Columns)),
		Relations:  make([]semantic.Relation, 0),
	}

	var diagnostics []Diagnostic

	// Project columns to attributes
	for _, col := range table.Columns {
		attr := semantic.Attribute{
			ID:       sanitizeAttributeID(table.QualifiedName(), col.Name),
			Symbol:   col.Name,
			TypeRef:  core.MustCanonicalID(mapPhysicalType(col.PhysicalType)),
			Required: !col.Nullable,
		}
		data.Attributes = append(data.Attributes, attr)
	}

	// Project primary key to candidate key
	if len(table.PrimaryKey) > 0 {
		ck := semantic.CandidateKey{
			ID:         sanitizeCandidateKeyID(table.QualifiedName(), "pk"),
			Symbol:     "PK_" + table.Name,
			Attributes: make([]core.CanonicalID, 0, len(table.PrimaryKey)),
		}
		for _, col := range table.PrimaryKey {
			ck.Attributes = append(ck.Attributes, sanitizeAttributeID(table.QualifiedName(), col))
		}
		if opts.RequireOwnerAcceptance {
			ck.Symbol = ck.Symbol + "_structural_candidate"
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "physical-pk-requires-owner-acceptance",
				Table:   table.QualifiedName(),
				Message: "primary key was derived from physical constraint; Owner must accept as business key",
			})
		}
		data.CandidateKeys = append(data.CandidateKeys, ck)
	}

	// Project unique constraints to candidate keys
	for i, uc := range table.UniqueConstraints {
		ck := semantic.CandidateKey{
			ID:         sanitizeCandidateKeyID(table.QualifiedName(), fmt.Sprintf("uk_%d", i)),
			Symbol:     "UK_" + uc.Name,
			Attributes: make([]core.CanonicalID, 0, len(uc.Columns)),
		}
		for _, col := range uc.Columns {
			ck.Attributes = append(ck.Attributes, sanitizeAttributeID(table.QualifiedName(), col))
		}
		if opts.RequireOwnerAcceptance {
			ck.Symbol = ck.Symbol + "_structural_candidate"
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "unique-constraint-requires-owner-acceptance",
				Table:   table.QualifiedName(),
				Message: fmt.Sprintf("unique constraint %s was derived from physical constraint; Owner must accept as business key", uc.Name),
			})
		}
		data.CandidateKeys = append(data.CandidateKeys, ck)
	}

	// Project foreign keys to relations
	if opts.GenerateRelationsFromFK {
		for i, fk := range table.ForeignKeys {
			rel, diags := projectForeignKey(table, fk, i, snapshot)
			data.Relations = append(data.Relations, rel)
			diagnostics = append(diagnostics, diags...)
		}
	}

	// Warn if no candidate keys
	if len(data.CandidateKeys) == 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "no-candidate-key",
			Table:   table.QualifiedName(),
			Message: "table has no primary key or unique constraint; cannot derive candidate keys",
		})
	}

	return data, diagnostics
}

func projectForeignKey(table physical.Table, fk physical.ForeignKey, index int, snapshot physical.SchemaSnapshot) (semantic.Relation, []Diagnostic) {
	targetName := fk.ReferencedTable
	if strings.TrimSpace(fk.ReferencedSchema) != "" {
		targetName = fk.ReferencedSchema + "." + fk.ReferencedTable
	}
	// If the referenced schema is not declared, try to resolve it from the snapshot.
	if strings.TrimSpace(fk.ReferencedSchema) == "" {
		if ref, ok := snapshot.Table(fk.ReferencedTable); ok {
			targetName = ref.QualifiedName()
		}
	}

	rel := semantic.Relation{
		ID:     sanitizeRelationID(table.QualifiedName(), fk.Name, index),
		Symbol: fk.Name,
		Target: sanitizeDataID(targetName),
	}

	var diagnostics []Diagnostic

	// Map FK columns to relation keys
	if len(fk.Columns) == len(fk.ReferencedColumns) {
		rel.Keys = make([]semantic.RelationKey, 0, len(fk.Columns))
		for i := range fk.Columns {
			rel.Keys = append(rel.Keys, semantic.RelationKey{
				SourceAttribute: sanitizeAttributeID(table.QualifiedName(), fk.Columns[i]),
				TargetAttribute: sanitizeAttributeID(fk.ReferencedSchema+"."+fk.ReferencedTable, fk.ReferencedColumns[i]),
			})
		}
	} else {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "foreign-key-column-count-mismatch",
			Table:   table.QualifiedName(),
			Message: fmt.Sprintf("foreign key %s has %d columns but referenced table has %d columns", fk.Name, len(fk.Columns), len(fk.ReferencedColumns)),
		})
	}

	// Check if referenced table exists in snapshot
	if _, ok := snapshot.Table(fk.ReferencedSchema + "." + fk.ReferencedTable); !ok {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "dangling-foreign-key",
			Table:   table.QualifiedName(),
			Message: fmt.Sprintf("foreign key %s references %s.%s which is not in the schema snapshot", fk.Name, fk.ReferencedSchema, fk.ReferencedTable),
		})
	}

	return rel, diagnostics
}

func fingerprintSchema(snapshot physical.SchemaSnapshot) (string, error) {
	bytes, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:]), nil
}

func sanitizeNamespaceID(id core.CanonicalID) core.CanonicalID {
	return core.MustCanonicalID(strings.ToLower(string(id)))
}

func sanitizeDataID(qualifiedName string) core.CanonicalID {
	s := strings.ToLower(strings.ReplaceAll(qualifiedName, ".", "_"))
	s = strings.TrimPrefix(s, "_")
	return core.MustCanonicalID(s)
}

func sanitizeAttributeID(table, column string) core.CanonicalID {
	s := strings.ToLower(table + "_" + column)
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.Trim(s, "_")
	return core.MustCanonicalID(s)
}

func sanitizeCandidateKeyID(table, suffix string) core.CanonicalID {
	return core.MustCanonicalID(strings.ToLower(table + "_" + suffix))
}

func sanitizeRelationID(table, fkName string, index int) core.CanonicalID {
	return core.MustCanonicalID(strings.ToLower(table + "_" + fkName + "_" + fmt.Sprintf("%d", index)))
}

// ComputeDiffImpact compares two schema snapshots and returns the semantic impact.
func ComputeDiffImpact(oldSnap, newSnap physical.SchemaSnapshot) SchemaDiffImpact {
	oldTables := make(map[string]physical.Table)
	newTables := make(map[string]physical.Table)
	for _, t := range oldSnap.Tables {
		oldTables[t.QualifiedName()] = t
	}
	for _, t := range newSnap.Tables {
		newTables[t.QualifiedName()] = t
	}

	impact := SchemaDiffImpact{}

	// Find added/dropped tables
	for name := range newTables {
		if _, ok := oldTables[name]; !ok {
			impact.AddedTables = append(impact.AddedTables, name)
		}
	}
	for name := range oldTables {
		if _, ok := newTables[name]; !ok {
			impact.DroppedTables = append(impact.DroppedTables, name)
		}
	}

	// Find added/dropped/altered columns and relations
	for name, newTable := range newTables {
		if oldTable, ok := oldTables[name]; ok {
			oldCols := make(map[string]physical.Column)
			for _, c := range oldTable.Columns {
				oldCols[c.Name] = c
			}
			newCols := make(map[string]physical.Column)
			for _, c := range newTable.Columns {
				newCols[c.Name] = c
			}

			for _, c := range newTable.Columns {
				if _, ok := oldCols[c.Name]; !ok {
					impact.AddedColumns = append(impact.AddedColumns, fmt.Sprintf("%s.%s", name, c.Name))
				} else if oldCols[c.Name] != c {
					impact.AlteredColumns = append(impact.AlteredColumns, fmt.Sprintf("%s.%s", name, c.Name))
				}
			}
			for _, c := range oldTable.Columns {
				if _, ok := newCols[c.Name]; !ok {
					impact.DroppedColumns = append(impact.DroppedColumns, fmt.Sprintf("%s.%s", name, c.Name))
				}
			}

			oldFKs := make(map[string]physical.ForeignKey)
			for _, fk := range oldTable.ForeignKeys {
				oldFKs[fk.Name] = fk
			}
			newFKs := make(map[string]physical.ForeignKey)
			for _, fk := range newTable.ForeignKeys {
				newFKs[fk.Name] = fk
			}
			for _, fk := range newTable.ForeignKeys {
				if _, ok := oldFKs[fk.Name]; !ok {
					impact.AddedRelations = append(impact.AddedRelations, fmt.Sprintf("%s.%s", name, fk.Name))
				}
			}
			for _, fk := range oldTable.ForeignKeys {
				if _, ok := newFKs[fk.Name]; !ok {
					impact.DroppedRelations = append(impact.DroppedRelations, fmt.Sprintf("%s.%s", name, fk.Name))
				}
			}

			// Check candidate key changes
			if len(oldTable.PrimaryKey) != len(newTable.PrimaryKey) {
				impact.AffectedCandidateKeys = append(impact.AffectedCandidateKeys, name)
			} else {
				for i := range oldTable.PrimaryKey {
					if oldTable.PrimaryKey[i] != newTable.PrimaryKey[i] {
						impact.AffectedCandidateKeys = append(impact.AffectedCandidateKeys, name)
						break
					}
				}
			}
		}
	}

	return impact
}

func mapPhysicalType(physicalType string) string {
	normalized := strings.ToLower(strings.TrimSpace(physicalType))
	switch normalized {
	case "integer", "int", "int4", "int8", "smallint", "bigint":
		return "db.integer"
	case "real", "double precision", "float", "float8", "numeric", "decimal":
		return "db.decimal"
	case "text", "varchar", "char", "string":
		return "db.text"
	case "boolean", "bool":
		return "db.boolean"
	case "date", "timestamp", "timestamptz", "datetime":
		return "db.timestamp"
	case "blob", "bytea", "binary":
		return "db.blob"
	default:
		return "db.unknown"
	}
}
