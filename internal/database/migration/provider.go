package migration

import "tei/internal/core"

type Tool string

const (
	ToolNative        Tool = "native"
	ToolAtlas         Tool = "atlas"
	ToolFlyway        Tool = "flyway"
	ToolLiquibase     Tool = "liquibase"
	ToolGolangMigrate Tool = "golang-migrate"
)

func (t Tool) Valid() bool {
	return t == ToolNative || t == ToolAtlas || t == ToolFlyway || t == ToolLiquibase || t == ToolGolangMigrate
}

type Mode string

const (
	ModeVersioned Mode = "versioned"
)

func (m Mode) Valid() bool { return m == ModeVersioned }

// ProviderConfig selects the external/versioning mechanism for one Database
// Unit. It does not alter Migration Definition semantics and can therefore be
// replaced without changing canonical schema meaning.
type ProviderConfig struct {
	Database core.CanonicalID `json:"database"`
	Tool     Tool             `json:"tool"`
	Mode     Mode             `json:"mode"`
	Source   string           `json:"source,omitempty"`
}
