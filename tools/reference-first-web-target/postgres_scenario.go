//go:build !tei_repair_only

package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"tei/internal/composition"
	"tei/internal/driver"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/tools/internal/postgresdb"
)

var errPostgresCommitUnknown = errors.New("PostgreSQL commit outcome is unknown; do not automatically retry")

// This host adapter owns one explicitly configured reference database. It
// never creates, drops or seeds objects, and never falls back to fixture data.
type postgresScenarioState struct {
	model  *scenarioModel
	db     *sql.DB
	world  ir.World
	digest string
	fence  int64
	// Internal transaction seam for actual-DB acknowledgement-loss tests.
	// Runtime hosts leave this nil and call sql.Tx.Commit directly.
	commit func(*sql.Tx) error
}

func openScenarioPostgres(ctx context.Context, path string) (*sql.DB, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("cannot read PostgreSQL connection file")
	}
	dsn := strings.TrimSuffix(strings.TrimSuffix(string(raw), "\n"), "\r")
	if dsn == "" || strings.ContainsAny(dsn, "\r\n\x00") {
		return nil, errors.New("PostgreSQL connection file must contain one nonempty line")
	}
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		return nil, errors.New("invalid PostgreSQL connection configuration")
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, errors.New("PostgreSQL connection check failed")
	}
	return db, nil
}

type grantReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Invocation identity does not grant permission. The host-owned grant row is
// checked against the pinned instance, current expiry and expected fencing.
// For effects, FOR SHARE holds the grant until commit; revocation/takeover
// updates must serialize on that row, even if issued by another connection.
func (s *postgresScenarioState) checkGrant(ctx context.Context, reader grantReader, lock bool) error {
	if err := s.model.Check(ctx); err != nil {
		return err
	}
	b, ok := composition.InvocationBinding(ctx)
	if !ok || b.ArtifactDigest != s.digest || s.fence <= 0 {
		return errors.New("PostgreSQL operation requires a pinned host instance")
	}
	query := `SELECT 1 FROM public.tei_reference_execution_grants
 WHERE scope = 'reference/order_entry/development' AND provider_id = $1
 AND contract_id = $2 AND contract_version = $3 AND artifact_digest = $4
 AND generation = $5 AND fencing_token = $6 AND NOT revoked
 AND expires_at > clock_timestamp()`
	args := []any{b.ProviderID, b.ContractID, b.ContractVersion, b.ArtifactDigest, b.Generation, s.fence}
	if s.model != nil {
		query += " AND model_revision = $7"
		args = append(args, s.model.revision)
	} else {
		query += " AND model_revision = ''"
	}
	if lock {
		query += " FOR SHARE"
	}
	var permitted int
	if err := reader.QueryRowContext(ctx, query, args...).Scan(&permitted); err != nil {
		return errors.New("PostgreSQL execution grant is absent, expired, revoked, fenced or unavailable")
	}
	return nil
}

func (s *postgresScenarioState) contracts(tx *sql.Tx) driver.ContractQueryExecutorMap {
	registry := referenceCustomerContractQueryExecutors()
	registry[driver.ContractQueryExecutorRuntimeKey{Environment: "development", Instance: "sales_core.dev", Capability: "database.query.postgres"}] = sqlquery.Executor{
		Provider:     environment.ProviderPostgres,
		Realizations: s.world.ContractQuerySQLRealizations,
		Runner:       postgresquery.Runner{DB: tx},
	}
	return registry
}

type postgresReadScenario struct{ state *postgresScenarioState }

func (*postgresReadScenario) ApplicationID() string { return "order_entry" }

func (r *postgresReadScenario) Run(ctx context.Context, input readScenarioInput) (readScenarioResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	s := r.state
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return readScenarioResult{}, errors.New("cannot start PostgreSQL read snapshot")
	}
	defer tx.Rollback()
	if err := s.checkGrant(ctx, tx, false); err != nil {
		return readScenarioResult{}, err
	}
	runner := newReferenceReadScenarioRunner(s.world)
	runner.browse = postgresquery.Executor{DB: tx, Database: "sales_core"}
	runner.contracts = s.contracts(tx)
	result, err := runner.Run(ctx, input)
	if err != nil {
		return readScenarioResult{}, errors.New("PostgreSQL-backed reference read failed; no synthetic fallback")
	}
	id, ok := semanticRowValueByAttribute(result.SelectedOrder, "sales.order.order_id")
	if !ok {
		return readScenarioResult{}, errors.New("selected order has no semantic identity")
	}
	if err := tx.QueryRowContext(ctx, "SELECT xmin::text FROM public.orders WHERE order_id = $1", id).Scan(&result.StorageRevision); err != nil {
		return readScenarioResult{}, errors.New("cannot bind PostgreSQL order snapshot")
	}
	if err := tx.Commit(); err != nil {
		return readScenarioResult{}, errors.New("cannot complete PostgreSQL read snapshot")
	}
	result.EvidenceSources = []scenarioEvidenceSource{
		{Part: "orders", Nature: "real-postgres-reference-data"},
		{Part: "order-lines", Nature: "real-postgres-reference-data"},
		{Part: "oracle-customer", Nature: "synthetic"},
	}
	result.AuthorityLimits = append(result.AuthorityLimits, "postgres-order-and-lines-share-one-repeatable-read-snapshot", "oracle-customer-remains-synthetic", "database-grant-is-checked-by-trusted-host-adapter")
	for i, limit := range result.AuthorityLimits {
		if limit == "synthetic-runtime-fixture-is-not-canonical-data" {
			result.AuthorityLimits[i] = "postgres-reference-rows-and-synthetic-oracle-customer-are-not-canonical-data"
		}
	}
	return result, nil
}

func lineSnapshotDigest(rows []semanticRowProjection) string {
	raw, _ := json.Marshal(rows)
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

// Submit rejects stale snapshots and repeated state transitions. The coarse
// table lock prevents line insert/update/delete between revalidation and commit
// in this small reference; it is not a production concurrency strategy.
func (s *postgresScenarioState) Submit(ctx context.Context, id int64, snapshot readScenarioResult, authorize func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if snapshot.StorageRevision == "" || authorize == nil {
		return errors.New("PostgreSQL submit requires snapshot and current business authorization")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("cannot start PostgreSQL submit transaction")
	}
	defer tx.Rollback()
	if err := s.checkGrant(ctx, tx, true); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "LOCK TABLE public.order_lines IN SHARE MODE"); err != nil {
		return errors.New("cannot protect PostgreSQL line snapshot")
	}
	var version, status string
	if err := tx.QueryRowContext(ctx, "SELECT xmin::text, status FROM public.orders WHERE order_id = $1 FOR UPDATE", id).Scan(&version, &status); err != nil {
		return errors.New("cannot lock selected PostgreSQL order")
	}
	if version != snapshot.StorageRevision || status != "draft" {
		return errors.New("PostgreSQL order snapshot is stale or order is no longer draft")
	}
	resolver := driver.NewResolver(s.world)
	resolved, err := resolver.ResolveHeaderDetailContractQuery("order_entry", "order_entry.order_lines")
	if err != nil {
		return err
	}
	// Resolve the canonical target and mapping again; physical column names never
	// become a replacement semantic contract or leak the parent support column.
	executor := sqlquery.Executor{Provider: environment.ProviderPostgres, Realizations: s.world.ContractQuerySQLRealizations, Runner: postgresquery.Runner{DB: tx}}
	rows, err := executor.ExecuteContractQuery(ctx, resolved.Binding.Target, resolved.Query, map[string]any{"order_id": id})
	if err != nil {
		return errors.New("cannot revalidate PostgreSQL line snapshot")
	}
	if lineSnapshotDigest(projectContractRows(rows.Rows)) != lineSnapshotDigest(snapshot.OrderLineRows) {
		return errors.New("PostgreSQL order lines changed after validation")
	}
	if err := authorize(ctx); err != nil {
		return err
	}
	// Recheck expiry at the mutation boundary. The shared grant lock already
	// serializes grant revocation and fencing changes until transaction completion.
	if err := s.checkGrant(ctx, tx, false); err != nil {
		return err
	}
	changed, err := tx.ExecContext(ctx, "UPDATE public.orders SET status = 'submitted' WHERE order_id = $1 AND status = 'draft'", id)
	if err != nil {
		return errors.New("PostgreSQL order update failed")
	}
	count, err := changed.RowsAffected()
	if err != nil || count != 1 {
		return errors.New("PostgreSQL order update did not affect exactly one row")
	}
	commit := tx.Commit
	if s.commit != nil {
		commit = func() error { return s.commit(tx) }
	}
	if err := commit(); err != nil {
		return errPostgresCommitUnknown
	}
	return nil
}

func (s *postgresScenarioState) Observe(ctx context.Context, id int64) (driver.ObservationReadExecution, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return driver.ExecuteObservationRead(ctx, "postgres.order_state.read", driver.ObservationEvidenceExternalObserved, func(ctx context.Context) (any, error) {
		if err := s.checkGrant(ctx, s.db, false); err != nil {
			return nil, err
		}
		var status string
		if err := s.db.QueryRowContext(ctx, "SELECT status FROM public.orders WHERE order_id = $1", id).Scan(&status); err != nil {
			return nil, errors.New("PostgreSQL order observation failed")
		}
		return status, nil
	})
}

// Keep a compile-time check that these adapters do not replace Driver APIs.
var _ commandPersistence = (*postgresScenarioState)(nil)
var _ readScenarioRunner = (*postgresReadScenario)(nil)
