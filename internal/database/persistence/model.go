package persistence

import "tei/internal/core"

type StorageType string

const (
	StorageInt64      StorageType = "int64"
	StorageVarchar    StorageType = "varchar"
	StorageText       StorageType = "text"
	StorageDateTimeTZ StorageType = "datetime-tz"
	StorageDecimal    StorageType = "decimal"
	StorageJSON       StorageType = "json"
)

func (s StorageType) Valid() bool {
	switch s {
	case StorageInt64, StorageVarchar, StorageText, StorageDateTimeTZ, StorageDecimal, StorageJSON:
		return true
	default:
		return false
	}
}

type Nullability string

const (
	Required Nullability = "required"
	Optional Nullability = "optional"
)

func (n Nullability) Valid() bool { return n == Required || n == Optional }

// Mapping is a database-owned persistence realization of one semantic data
// element. It is intentionally separate from the semantic model so the same
// domain meaning can be realized by different databases and physical layouts.
type Mapping struct {
	ID                core.CanonicalID          `json:"id"`
	Database          core.CanonicalID          `json:"database"`
	SemanticData      core.CanonicalID          `json:"semanticData"`
	Table             string                    `json:"table"`
	Columns           []ColumnMapping           `json:"columns"`
	PrimaryKey        []core.CanonicalID        `json:"primaryKey,omitempty"`
	UniqueConstraints []UniqueConstraintMapping `json:"uniqueConstraints,omitempty"`
	Indexes           []IndexMapping            `json:"indexes,omitempty"`
	ForeignKeys       []ForeignKeyMapping       `json:"foreignKeys,omitempty"`
}

type ColumnMapping struct {
	Attribute   core.CanonicalID `json:"attribute"`
	Column      string           `json:"column"`
	Storage     StorageType      `json:"storage"`
	Nullability Nullability      `json:"nullability"`
}

// UniqueConstraintMapping records a relational uniqueness constraint as a
// database-owned persistence decision. It is intentionally separate from
// IndexMapping so constraint intent is not collapsed into an access path.
type UniqueConstraintMapping struct {
	Name       string             `json:"name,omitempty"`
	Attributes []core.CanonicalID `json:"attributes"`
}

type IndexMapping struct {
	Name       string             `json:"name"`
	Attributes []core.CanonicalID `json:"attributes"`
	Unique     bool               `json:"unique,omitempty"`
}

// ForeignKeyMapping records an explicit database-owned decision to realize a
// relational foreign key. It refers to semantic attributes through Mapping
// identities rather than embedding physical target table/column names.
type ForeignKeyMapping struct {
	Name                 string             `json:"name,omitempty"`
	Attributes           []core.CanonicalID `json:"attributes"`
	ReferencedMapping    core.CanonicalID   `json:"referencedMapping"`
	ReferencedAttributes []core.CanonicalID `json:"referencedAttributes"`
}

func (m Mapping) ColumnByAttribute(id core.CanonicalID) (ColumnMapping, bool) {
	for _, c := range m.Columns {
		if c.Attribute == id {
			return c, true
		}
	}
	return ColumnMapping{}, false
}
