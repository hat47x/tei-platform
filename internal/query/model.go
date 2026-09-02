package query

import (
	"tei/internal/core"
	"tei/internal/environment"
)

// ResultColumnMappingは、native Queryが返す一つのraw列labelをSemantic attributeへ結び付ける。
//
// Persistence Mappingは保存構造との対応を所有するのに対し、このmappingはQuery固有の結果射影を所有する。
// DriverやProviderは、列名の類似やPersistence Mappingからこの意味を推測しない。
type ResultColumnMapping struct {
	Column    string           `json:"column"`
	Attribute core.CanonicalID `json:"attribute"`
}

type EmbeddedSQL struct {
	ID            core.CanonicalID             `json:"id"`
	Application   core.CanonicalID             `json:"application"`
	Datasource    core.LocalName               `json:"datasource"`
	Dialect       environment.DatabaseProvider `json:"dialect"`
	Source        string                       `json:"source"`
	ResultColumns []ResultColumnMapping        `json:"resultColumns,omitempty"`
}

type PhysicalReference struct {
	Database core.CanonicalID `json:"database"`
	Object   string           `json:"object"`
}
