package reconcile

import (
	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/database/schemadiff"
)

type FindingKind string

const (
	FindingPendingMigration  FindingKind = "pending-migration"
	FindingFailedMigration   FindingKind = "failed-migration"
	FindingDivergedMigration FindingKind = "diverged-migration"
	FindingOrphanApplied     FindingKind = "orphan-applied-migration"
	FindingSchemaDrift       FindingKind = "schema-drift"
)

type Finding struct {
	Kind      FindingKind `json:"kind"`
	Migration string      `json:"migration,omitempty"`
	Detail    string      `json:"detail,omitempty"`
}

type Result struct {
	Database         core.CanonicalID            `json:"database"`
	MigrationHistory migration.HistoryComparison `json:"migrationHistory"`
	SchemaDiff       schemadiff.Diff             `json:"schemaDiff"`
	Findings         []Finding                   `json:"findings,omitempty"`
}

func Build(database core.CanonicalID, history migration.HistoryComparison, diff schemadiff.Diff) Result {
	r := Result{Database: database, MigrationHistory: history, SchemaDiff: diff}
	for _, e := range history.Entries {
		switch e.State {
		case migration.RepoPending:
			r.Findings = append(r.Findings, Finding{Kind: FindingPendingMigration, Migration: e.ID})
		case migration.RepoFailed:
			r.Findings = append(r.Findings, Finding{Kind: FindingFailedMigration, Migration: e.ID})
		case migration.RepoDiverged:
			r.Findings = append(r.Findings, Finding{Kind: FindingDivergedMigration, Migration: e.ID, Detail: "repository checksum differs from applied history"})
		case migration.RepoOrphanApplied:
			r.Findings = append(r.Findings, Finding{Kind: FindingOrphanApplied, Migration: e.ID})
		}
	}
	if len(diff.Changes) > 0 {
		r.Findings = append(r.Findings, Finding{Kind: FindingSchemaDrift, Detail: "actual physical schema differs from desired physical schema"})
	}
	return r
}
