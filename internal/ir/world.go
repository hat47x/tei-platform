package ir

import (
	"tei/internal/application"
	apprelease "tei/internal/application/release"
	"tei/internal/database"
	"tei/internal/database/migration"
	"tei/internal/database/persistence"
	"tei/internal/database/release"
	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/query"
	"tei/internal/realization"
	"tei/internal/releaseplan"
	"tei/internal/semantic"
)

// World is the normalized, parser-independent M0 compilation unit.
// It is intentionally serializable so the semantic/compiler layers can be
// tested before the surface DSL parser is complete.
type World struct {
	Plugins                           []plugin.Definition                            `json:"plugins,omitempty"`
	Bundles                           []plugin.Bundle                                `json:"bundles,omitempty"`
	Semantics                         []semantic.Namespace                           `json:"semantics,omitempty"`
	Applications                      []application.Application                      `json:"applications"`
	Databases                         []database.Database                            `json:"databases"`
	Persistence                       []persistence.Mapping                          `json:"persistenceMappings,omitempty"`
	Migrations                        []migration.Definition                         `json:"migrations,omitempty"`
	MigrationProviders                []migration.ProviderConfig                     `json:"migrationProviders,omitempty"`
	ApplicationReleases               []apprelease.Definition                        `json:"applicationReleases,omitempty"`
	DatabaseReleases                  []release.Definition                           `json:"databaseReleases,omitempty"`
	SolutionReleases                  []releaseplan.Definition                       `json:"solutionReleases,omitempty"`
	Environments                      []environment.Environment                      `json:"environments"`
	Queries                           []query.EmbeddedSQL                            `json:"queries"`
	Bindings                          []realization.CustomGoBinding                  `json:"customGoBindings,omitempty"`
	HeaderDetailContractQueryBindings []realization.HeaderDetailContractQueryBinding `json:"headerDetailContractQueryBindings,omitempty"`
	ContractQueryExecutorBindings     []realization.ContractQueryExecutorBinding     `json:"contractQueryExecutorBindings,omitempty"`
	ContractQuerySQLRealizations      []realization.ContractQuerySQLRealization      `json:"contractQuerySQLRealizations,omitempty"`
}
