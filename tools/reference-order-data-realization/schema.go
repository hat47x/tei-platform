package orderdatarealization

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
	dbrealization "tei/internal/database/realization"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
	"tei/internal/ir"
)

const databaseID = "sales_core"

// mappedSchemas are the schemas the storage mapping uses. The comparison is
// projected to them; the sample's bookkeeping lives in odr_meta.
var mappedSchemas = []string{"public"}

var (
	ErrObservationChanged = errors.New("the database structure changed after the migration was planned")
	ErrNeedsDecision      = errors.New("the difference contains changes that are not applied automatically")
	ErrOldWritersRemain   = errors.New("old-version writers remain; the contract stage is refused")
)

func observe(ctx context.Context, db *sql.DB, instance environment.DatabaseInstance) (postgres.SchemaObservation, error) {
	obs, err := postgres.Provider{Catalog: postgres.SQLCatalogReader{DB: db}}.ObserveSchema(ctx, instance)
	if err != nil {
		return obs, err
	}
	var tables []physical.Table
	for _, t := range obs.Snapshot.Tables {
		if slices.Contains(mappedSchemas, t.Schema) {
			tables = append(tables, t)
		}
	}
	obs.Snapshot.Tables = tables
	var parts []postgres.PartitionRelation
	for _, p := range obs.Partitions {
		if slices.Contains(mappedSchemas, p.Child.Schema) || slices.Contains(mappedSchemas, p.Parent.Schema) {
			parts = append(parts, p)
		}
	}
	obs.Partitions = parts
	return obs, nil
}

// observationFingerprint hashes the normalized structure without the capture
// time.
func observationFingerprint(s physical.SchemaSnapshot) string {
	n := s.Normalize()
	n.CapturedAt, n.Fingerprint = "", ""
	raw, _ := json.Marshal(n)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func desiredOf(world ir.World) (physical.SchemaSnapshot, error) {
	return dbrealization.RealizeDesired(world.Persistence, databaseID, environment.ProviderPostgres)
}

// PlannedStage is one declared stage of the migration.
type PlannedStage struct {
	Name                  string              `json:"name"`
	Changes               []schemadiff.Change `json:"changes"`
	Statements            []string            `json:"statements"`
	RenderedBy            string              `json:"renderedBy"`
	ProviderRenderError   string              `json:"providerRenderError,omitempty"`
	RefuseWhileOldWriters bool                `json:"refuseWhileOldWriters"`
}

// MigrationPlan binds the migration to the final model revision and to the
// fingerprint of the observation it was planned from.
type MigrationPlan struct {
	ID                    string               `json:"id"`
	ModelRevision         string               `json:"modelRevision"`
	BasedOnObservation    string               `json:"basedOnObservation"`
	Diff                  schemadiff.Diff      `json:"diff"`
	Stages                []PlannedStage       `json:"stages"`
	NeedsDecision         []schemadiff.Change  `json:"needsDecision,omitempty"`
	Definition            migration.Definition `json:"expandDefinition"`
	DefinitionFingerprint string               `json:"expandDefinitionFingerprint"`
}

func (p *MigrationPlan) stage(name string) *PlannedStage {
	for i := range p.Stages {
		if p.Stages[i].Name == name {
			return &p.Stages[i]
		}
	}
	return nil
}

func qualifiedTable(table string) string {
	parts := strings.Split(table, ".")
	for i, p := range parts {
		parts[i] = quoteIdent(p)
	}
	return strings.Join(parts, ".")
}

// renderSampleChange renders what the PostgreSQL provider does not render.
// Only adding a unique constraint is supported.
func renderSampleChange(c schemadiff.Change) (string, error) {
	if c.Kind != schemadiff.AddUniqueConstraint || c.UniqueConstraint == nil || c.UniqueConstraint.Name == "" {
		return "", fmt.Errorf("the sample renders only named add-unique-constraint changes, not %s", c.Kind)
	}
	cols := make([]string, len(c.UniqueConstraint.Columns))
	for i, col := range c.UniqueConstraint.Columns {
		cols[i] = quoteIdent(col)
	}
	return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s)", qualifiedTable(c.Table), quoteIdent(c.UniqueConstraint.Name), strings.Join(cols, ", ")), nil
}

// planMigration classifies the difference into the declared stages. A
// difference with a change that is declared not to be applied automatically
// (dropping a column or a table) is returned with ErrNeedsDecision.
func planMigration(id, modelRevision string, decl MigrationDecl, desired, actual physical.SchemaSnapshot) (*MigrationPlan, error) {
	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		return nil, err
	}
	p := &MigrationPlan{ID: id, ModelRevision: modelRevision, BasedOnObservation: observationFingerprint(actual), Diff: diff}
	if len(diff.Unresolved) > 0 {
		return p, fmt.Errorf("unresolved comparisons: %+v", diff.Unresolved)
	}
	byStage := map[string][]schemadiff.Change{}
	for _, c := range diff.Changes {
		if slices.Contains(decl.NotAppliedAutomatically, string(c.Kind)) {
			p.NeedsDecision = append(p.NeedsDecision, c)
			continue
		}
		s, ok := decl.stageOf(string(c.Kind))
		if !ok {
			return p, fmt.Errorf("change %s %s has no declared stage", c.Kind, c.Table)
		}
		byStage[s.Name] = append(byStage[s.Name], c)
	}
	if len(p.NeedsDecision) > 0 {
		return p, ErrNeedsDecision
	}
	for _, s := range decl.Stages {
		changes := byStage[s.Name]
		st := PlannedStage{Name: s.Name, Changes: changes, RefuseWhileOldWriters: s.RefuseWhileOldWriters}
		sub := schemadiff.Diff{Database: diff.Database, Changes: changes}
		if len(changes) == 0 {
			st.RenderedBy = "nothing-to-do"
		} else if stmts, rerr := migration.Render(environment.ProviderPostgres, sub); rerr == nil {
			st.Statements, st.RenderedBy = stmts, "provider:postgres"
		} else {
			st.ProviderRenderError = rerr.Error()
			st.RenderedBy = "sample"
			for _, c := range changes {
				stmt, err := renderSampleChange(c)
				if err != nil {
					return p, fmt.Errorf("stage %s: provider: %v; sample: %w", s.Name, rerr, err)
				}
				st.Statements = append(st.Statements, stmt)
			}
		}
		if s.Name == "expand" {
			def, err := migration.Plan(id+"-expand", core.CanonicalID(databaseID), p.BasedOnObservation, sub, environment.ProviderPostgres)
			if err != nil {
				return p, fmt.Errorf("expand definition: %w", err)
			}
			if !slices.Equal(def.Realizations[0].Statements, st.Statements) {
				return p, fmt.Errorf("expand definition statements differ from the stage")
			}
			p.Definition = def
			if p.DefinitionFingerprint, err = migration.Fingerprint(def); err != nil {
				return p, err
			}
		}
		p.Stages = append(p.Stages, st)
	}
	return p, nil
}

// execWithLockTimeout runs statements in one transaction with the declared
// lock timeout and returns the elapsed time on the monotonic clock.
func execWithLockTimeout(ctx context.Context, db *sql.DB, timeoutMillis int, stmts []string) (time.Duration, error) {
	start := time.Now()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL lock_timeout = '%dms'", timeoutMillis)); err != nil {
		return 0, err
	}
	for _, s := range stmts {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			return time.Since(start), fmt.Errorf("%s: %w", s, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return time.Since(start), err
	}
	return time.Since(start), nil
}

// StageApply records the application of one stage.
type StageApply struct {
	Stage                string   `json:"stage"`
	FreshObservation     string   `json:"freshObservation,omitempty"`
	Statements           []string `json:"statements,omitempty"`
	ElapsedMillis        float64  `json:"elapsedMillis"`
	LockTimeoutMillis    int      `json:"lockTimeoutMillis"`
	WithinLockTimeout    bool     `json:"withinLockTimeout"`
	AppliedThroughCore   string   `json:"appliedThrough"`
}

// applyExpand checks the fresh observation against the plan before anything
// is executed, then applies the expand stage through the provider's
// apply-with-fresh-observation entry.
func applyExpand(ctx context.Context, db *sql.DB, instance environment.DatabaseInstance, plan *MigrationPlan, lockMillis int) (StageApply, error) {
	out := StageApply{Stage: "expand", LockTimeoutMillis: lockMillis, AppliedThroughCore: "migration.ApplyPostgresWithFreshObservation"}
	fresh, err := observe(ctx, db, instance)
	if err != nil {
		return out, err
	}
	out.FreshObservation = observationFingerprint(fresh.Snapshot)
	if out.FreshObservation != plan.BasedOnObservation {
		return out, fmt.Errorf("%w: planned %s, fresh %s", ErrObservationChanged, plan.BasedOnObservation, out.FreshObservation)
	}
	st := plan.stage("expand")
	if st == nil || len(st.Statements) == 0 {
		return out, nil
	}
	var elapsed time.Duration
	err = migration.ApplyPostgresWithFreshObservation(ctx, plan.Definition,
		func(ctx context.Context) (postgres.SchemaObservation, error) { return observe(ctx, db, instance) },
		func(ctx context.Context, stmts []string) error {
			out.Statements = stmts
			var err error
			elapsed, err = execWithLockTimeout(ctx, db, lockMillis, stmts)
			return err
		})
	out.ElapsedMillis = float64(elapsed.Microseconds()) / 1000
	out.WithinLockTimeout = err == nil && elapsed < time.Duration(lockMillis)*time.Millisecond
	return out, err
}

func activeOldWriters(ctx context.Context, db *sql.DB, final string) ([]string, error) {
	return listStringsArgs(ctx, db, `SELECT writer_id || '@' || model_revision FROM `+metaSchema+`.writer WHERE state = 'active' AND model_revision <> $1 ORDER BY 1`, final)
}

func listStringsArgs(ctx context.Context, db *sql.DB, query string, args ...any) ([]string, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// applyContract refuses while old-version writers are registered as active,
// then requires the fresh difference to be exactly the planned contract
// changes before it executes them.
func applyContract(ctx context.Context, db *sql.DB, instance environment.DatabaseInstance, plan *MigrationPlan, desired physical.SchemaSnapshot, final string, lockMillis int) (StageApply, error) {
	out := StageApply{Stage: "contract", LockTimeoutMillis: lockMillis, AppliedThroughCore: "sample (provider does not render add-unique-constraint)"}
	st := plan.stage("contract")
	if st == nil {
		return out, fmt.Errorf("no contract stage")
	}
	if st.RefuseWhileOldWriters {
		writers, err := activeOldWriters(ctx, db, final)
		if err != nil {
			return out, err
		}
		if len(writers) > 0 {
			return out, fmt.Errorf("%w: %v", ErrOldWritersRemain, writers)
		}
	}
	fresh, err := observe(ctx, db, instance)
	if err != nil {
		return out, err
	}
	out.FreshObservation = observationFingerprint(fresh.Snapshot)
	diff, err := schemadiff.Compare(desired, fresh.Snapshot)
	if err != nil {
		return out, err
	}
	want, _ := json.Marshal(st.Changes)
	got, _ := json.Marshal(diff.Changes)
	if string(want) != string(got) || len(diff.Unresolved) > 0 {
		return out, fmt.Errorf("%w: remaining difference %s differs from the planned contract stage %s", ErrObservationChanged, got, want)
	}
	elapsed, err := execWithLockTimeout(ctx, db, lockMillis, st.Statements)
	out.Statements = st.Statements
	out.ElapsedMillis = float64(elapsed.Microseconds()) / 1000
	out.WithinLockTimeout = err == nil && elapsed < time.Duration(lockMillis)*time.Millisecond
	return out, err
}

func converged(ctx context.Context, db *sql.DB, instance environment.DatabaseInstance, desired physical.SchemaSnapshot) (schemadiff.Diff, error) {
	fresh, err := observe(ctx, db, instance)
	if err != nil {
		return schemadiff.Diff{}, err
	}
	return schemadiff.Compare(desired, fresh.Snapshot)
}
