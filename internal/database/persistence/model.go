package persistence

import (
	"tei/internal/core"
	"tei/internal/environment"
)

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
	RelationStorage   []RelationStorageMapping  `json:"relationStorage,omitempty"`
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

// RelationStorageMappingは、target Semantic Dataへ新しい属性を追加せず、
// Semantic Relationをmanaged-here Databaseのtableへ保存するための補助列群を表す。
// physical column名やForeign KeyをSemantic Relationのidentityとして扱わない。
type RelationStorageMapping struct {
	Relation   core.CanonicalID                  `json:"relation"`
	Components []RelationStorageComponentMapping `json:"components"`
}

// RelationStorageComponentMappingは、Relation source Data上のSemantic attributeを、
// target Mappingのtable内にあるDatabase-ownedな物理support columnへ対応付ける。
// SourceAttributeはtarget Semantic Dataの属性ではないため、通常のColumnMappingとは分離する。
type RelationStorageComponentMapping struct {
	SourceAttribute core.CanonicalID `json:"sourceAttribute"`
	Column          string           `json:"column"`
	Storage         StorageType      `json:"storage"`
	Nullability     Nullability      `json:"nullability"`
}

// UniqueConstraintMapping records a relational uniqueness constraint as a
// database-owned persistence decision. It is intentionally separate from
// IndexMapping so constraint intent is not collapsed into an access path.
type UniqueConstraintMapping struct {
	Name       string             `json:"name,omitempty"`
	Attributes []core.CanonicalID `json:"attributes"`
}

type IndexOrder string

const (
	IndexOrderDefault IndexOrder = "default"
	IndexOrderAsc     IndexOrder = "asc"
	IndexOrderDesc    IndexOrder = "desc"
)

func (o IndexOrder) Valid() bool {
	return o == "" || o == IndexOrderDefault || o == IndexOrderAsc || o == IndexOrderDesc
}

type IndexNullsOrder string

const (
	IndexNullsOrderDefault IndexNullsOrder = "default"
	IndexNullsOrderFirst   IndexNullsOrder = "first"
	IndexNullsOrderLast    IndexNullsOrder = "last"
)

func (o IndexNullsOrder) Valid() bool {
	return o == "" || o == IndexNullsOrderDefault || o == IndexNullsOrderFirst || o == IndexNullsOrderLast
}

type UniqueNullTreatment string

const (
	UniqueNullTreatmentDistinct    UniqueNullTreatment = "distinct"
	UniqueNullTreatmentNotDistinct UniqueNullTreatment = "not-distinct"
)

func (t UniqueNullTreatment) Valid() bool {
	return t == "" || t == UniqueNullTreatmentDistinct || t == UniqueNullTreatmentNotDistinct
}

// ProviderExpressionMapping is database-owned provider-native authoring. Source
// remains opaque to the persistence parser; Provider and Dialect preserve the
// provenance required to avoid comparing equal text across unrelated DBMSs.
type ProviderExpressionMapping struct {
	Provider environment.DatabaseProvider `json:"provider"`
	Dialect  string                       `json:"dialect"`
	Source   string                       `json:"source"`
}

// IndexKeyMapping is one ordered Advanced Index search key. Attribute and
// Expression are mutually exclusive. Attribute keys are resolved through this
// Mapping's ColumnMapping; expression keys remain provider-native.
type IndexKeyMapping struct {
	Attribute     core.CanonicalID           `json:"attribute,omitempty"`
	Expression    *ProviderExpressionMapping `json:"expression,omitempty"`
	Collation     string                     `json:"collation,omitempty"`
	OperatorClass string                     `json:"operatorClass,omitempty"`
	Order         IndexOrder                 `json:"order,omitempty"`
	NullsOrder    IndexNullsOrder            `json:"nullsOrder,omitempty"`
}

// IndexMapping retains Attributes for the existing simple authoring syntax and
// adds structured fields for Advanced Index authoring. Simple and advanced key
// representations must not be populated at the same time.
type IndexMapping struct {
	Name                string                     `json:"name"`
	Attributes          []core.CanonicalID         `json:"attributes,omitempty"`
	Keys                []IndexKeyMapping          `json:"keys,omitempty"`
	IncludedAttributes  []core.CanonicalID         `json:"includedAttributes,omitempty"`
	Predicate           *ProviderExpressionMapping `json:"predicate,omitempty"`
	Unique              bool                       `json:"unique,omitempty"`
	UniqueNullTreatment UniqueNullTreatment        `json:"uniqueNullTreatment,omitempty"`
	Method              string                     `json:"method,omitempty"`
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

func (m Mapping) RelationStorageByRelation(id core.CanonicalID) (RelationStorageMapping, bool) {
	for _, storage := range m.RelationStorage {
		if storage.Relation == id {
			return storage, true
		}
	}
	return RelationStorageMapping{}, false
}
