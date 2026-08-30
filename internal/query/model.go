package query

import (
	"tei/internal/core"
	"tei/internal/environment"
)

type EmbeddedSQL struct {
	ID          core.CanonicalID             `json:"id"`
	Application core.CanonicalID             `json:"application"`
	Datasource  core.LocalName               `json:"datasource"`
	Dialect     environment.DatabaseProvider `json:"dialect"`
	Source      string                       `json:"source"`
}

type PhysicalReference struct {
	Database core.CanonicalID `json:"database"`
	Object   string           `json:"object"`
}
