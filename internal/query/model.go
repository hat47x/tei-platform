package query

import (
	"tei/internal/core"
	"tei/internal/environment"
)

// ResultColumnMappingは、native Queryが返す一意なresult列名をSemantic attributeへ結び付ける。
//
// SQLの列名やPersistence Mappingから意味を推測せず、Query authoringがresult shapeの意味を明示する。
// v1では同名のresult列を複数持つQueryを暗黙にordinalで区別しない。必要な場合はSQL側で一意なaliasを付ける。
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
