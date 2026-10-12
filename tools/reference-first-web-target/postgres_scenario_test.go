//go:build !tei_repair_only

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"tei/internal/composition"
	"tei/internal/driver"
	"tei/internal/workspace"
)

type observationFailurePersistence struct{ submits int }

func (p *observationFailurePersistence) Submit(ctx context.Context, _ int64, _ readScenarioResult, authorize func(context.Context) error) error {
	if err := authorize(ctx); err != nil {
		return err
	}
	p.submits++
	return nil
}
func (*observationFailurePersistence) Observe(context.Context, int64) (driver.ObservationReadExecution, error) {
	return driver.ObservationReadExecution{}, errors.New("observation unavailable")
}

func TestPersistentCommandObservationFailureKeepsCommandAndDoesNotRetry(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := newReferenceCommandScenarioRunner(loaded.World, legacyFixturePath(t, "users.json"), legacyFixturePath(t, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	persistence := &observationFailurePersistence{}
	runner.persistence = persistence
	result, err := runner.RunCommand(context.Background(), postgresSubmitInput())
	if err == nil || result.Stage != "observation" || result.Command == nil || result.Observation != nil || persistence.submits != 1 {
		t.Fatalf("observation failure hid committed command or retried: %+v %v", result, err)
	}
}

type hookedReadScenario struct {
	read  readScenarioRunner
	after func()
}

func (*hookedReadScenario) ApplicationID() string { return "order_entry" }
func (r *hookedReadScenario) Run(ctx context.Context, input readScenarioInput) (readScenarioResult, error) {
	result, err := r.read.Run(ctx, input)
	if err == nil && r.after != nil {
		r.after()
	}
	return result, err
}

// Actual SQL tests require explicit opt-in and the repository fixture ownership
// marker on every table they modify. Normal repository health does not substitute
// skipped tests for evidence of a real PostgreSQL engine.
func postgresTestState(t *testing.T) *postgresScenarioState {
	t.Helper()
	path := os.Getenv("TEI_REFERENCE_POSTGRES_TEST_DSN_FILE")
	if path == "" {
		t.Skip("actual PostgreSQL reference database not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := openScenarioPostgres(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, table := range []string{"public.orders", "public.order_lines", "public.tei_reference_execution_grants"} {
		var marker string
		if err := db.QueryRowContext(ctx, "SELECT obj_description($1::regclass, 'pg_class')", table).Scan(&marker); err != nil || marker != "tei.reference.postgres-order-execution.v1" {
			t.Fatal("actual SQL tests require owned disposable fixture tables")
		}
	}
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	digest, err := runningExecutableDigest()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE public.orders SET status = 'draft', total_amount = CASE order_id WHEN 1001 THEN 12500.00 WHEN 1002 THEN 8300.00 ELSE 6200.00 END"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE public.order_lines SET quantity = CASE order_line_id WHEN 5001 THEN 2 WHEN 5003 THEN 0 ELSE 1 END"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE public.tei_reference_execution_grants SET artifact_digest = $1, generation = 1, fencing_token = 1, revoked = false, expires_at = clock_timestamp() + interval '10 minutes'", digest); err != nil {
		t.Fatal(err)
	}
	return &postgresScenarioState{db: db, world: loaded.World, digest: digest, fence: 1}
}

func postgresTestRunners(t *testing.T, state *postgresScenarioState, after func()) (*composedReadScenario, *composedCommandScenario) {
	t.Helper()
	command, err := newReferenceCommandScenarioRunner(state.world, legacyFixturePath(t, "users.json"), legacyFixturePath(t, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	command.persistence = state
	read := &hookedReadScenario{read: &postgresReadScenario{state: state}, after: after}
	r, c, err := wireComposedScenarios(context.Background(), composition.NewRegistry(), read, command, state.digest, func(composition.Binding) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = c.service.Withdraw(1)
		_ = c.service.Reap(context.Background())
		_ = r.service.Withdraw(1)
		_ = r.service.Reap(context.Background())
	})
	return r, c
}

func postgresSubmitInput() commandScenarioInput {
	return commandScenarioInput{LoginID: "legacy-order-editor", Credential: "editor-secret", From: "2026-09-18T08:30:00Z", RowIndex: 0}
}

func postgresStatus(t *testing.T, db *sql.DB) string {
	t.Helper()
	var status string
	if err := db.QueryRow("SELECT status FROM public.orders WHERE order_id = 1001").Scan(&status); err != nil {
		t.Fatal(err)
	}
	return status
}

func TestPostgresScenarioActualReadCommitAndSeparateObservation(t *testing.T) {
	s := postgresTestState(t)
	r, c := postgresTestRunners(t, s, nil)
	read, err := r.Run(context.Background(), readScenarioInput{From: postgresSubmitInput().From})
	if err != nil {
		t.Fatal(err)
	}
	if read.StorageRevision == "" || len(read.OrderLineRows) != 2 || len(read.EvidenceSources) != 3 {
		t.Fatalf("snapshot: %+v", read)
	}
	result, err := c.RunCommand(context.Background(), postgresSubmitInput())
	if err != nil {
		t.Fatal(err)
	}
	if result.Stage != "completed" || result.EffectAuthorization == nil || result.Command == nil || result.Observation == nil || result.Observation.Nature != "external-observed" || result.Observation.Result != "submitted" {
		t.Fatalf("result: %+v", result)
	}
	if result.Composition.ContractID != postgresSubmitServiceContract.ID || postgresStatus(t, s.db) != "submitted" {
		t.Fatal("persistent command or exact contract missing")
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "editor-secret") || strings.Contains(string(raw), "sales.order_line.order_id") {
		t.Fatal("credential or physical support column leaked")
	}
	if _, err := c.RunCommand(context.Background(), postgresSubmitInput()); err == nil {
		t.Fatal("repeated submit accepted")
	}
}

func TestPostgresScenarioRejectsChangedSnapshotsAndCurrentGrant(t *testing.T) {
	for name, change := range map[string]string{
		"order-changed":  "UPDATE public.orders SET total_amount = total_amount + 1 WHERE order_id = 1001",
		"line-changed":   "UPDATE public.order_lines SET quantity = quantity + 1 WHERE order_id = 1001",
		"revoked":        "UPDATE public.tei_reference_execution_grants SET revoked = true WHERE provider_id = 'order-submit'",
		"fenced":         "UPDATE public.tei_reference_execution_grants SET fencing_token = 2 WHERE provider_id = 'order-submit'",
		"expired":        "UPDATE public.tei_reference_execution_grants SET expires_at = clock_timestamp() - interval '1 second' WHERE provider_id = 'order-submit'",
		"wrong-artifact": "UPDATE public.tei_reference_execution_grants SET artifact_digest = 'wrong-artifact' WHERE provider_id = 'order-submit'",
	} {
		t.Run(name, func(t *testing.T) {
			s := postgresTestState(t)
			_, c := postgresTestRunners(t, s, func() {
				if _, err := s.db.Exec(change); err != nil {
					t.Fatal(err)
				}
			})
			if _, err := c.RunCommand(context.Background(), postgresSubmitInput()); err == nil {
				t.Fatal("stale or unauthorized effect accepted")
			}
			if postgresStatus(t, s.db) != "draft" {
				t.Fatal("rejected operation changed persistent state")
			}
		})
	}
}

func TestPostgresScenarioConcurrentSubmissionCommitsOnce(t *testing.T) {
	s := postgresTestState(t)
	arrived := make(chan struct{}, 2)
	release := make(chan struct{})
	_, c := postgresTestRunners(t, s, func() { arrived <- struct{}{}; <-release })
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.RunCommand(context.Background(), postgresSubmitInput())
			results <- err
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-arrived:
		case <-time.After(10 * time.Second):
			close(release)
			wg.Wait()
			t.Fatal("concurrent read did not arrive")
		}
	}
	close(release)
	wg.Wait()
	close(results)
	committed := 0
	for err := range results {
		if err == nil {
			committed++
		}
	}
	if committed != 1 || postgresStatus(t, s.db) != "submitted" {
		t.Fatalf("successful transitions: %d", committed)
	}
}

func TestPostgresScenarioGrantRevocationAndConnectionFailureDoNotFallback(t *testing.T) {
	s := postgresTestState(t)
	r, _ := postgresTestRunners(t, s, nil)
	if _, err := s.db.Exec("UPDATE public.tei_reference_execution_grants SET revoked = true WHERE provider_id = 'order-read'"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run(context.Background(), readScenarioInput{From: postgresSubmitInput().From}); err == nil {
		t.Fatal("read bypassed current grant")
	}
	_ = s.db.Close()
	if _, err := r.Run(context.Background(), readScenarioInput{From: postgresSubmitInput().From}); err == nil {
		t.Fatal("unavailable PostgreSQL fell back to synthetic rows")
	}
}

func TestPostgresScenarioBusinessDenialAndValidationDoNotCommit(t *testing.T) {
	s := postgresTestState(t)
	_, c := postgresTestRunners(t, s, nil)
	viewer := postgresSubmitInput()
	viewer.LoginID = "legacy-order-viewer"
	viewer.Credential = "viewer-secret"
	result, err := c.RunCommand(context.Background(), viewer)
	if err != nil || result.Stage != "authorization-denied" || result.Command != nil || postgresStatus(t, s.db) != "draft" {
		t.Fatalf("business denial: %+v %v", result, err)
	}
	invalid := postgresSubmitInput()
	invalid.From = "2026-09-18T07:00:00Z"
	invalid.RowIndex = 1
	result, err = c.RunCommand(context.Background(), invalid)
	if err != nil || result.Stage != "validation-not-satisfied" || result.Command != nil {
		t.Fatalf("canonical validation: %+v %v", result, err)
	}
	var status string
	if err := s.db.QueryRow("SELECT status FROM public.orders WHERE order_id = 1002").Scan(&status); err != nil || status != "draft" {
		t.Fatal("invalid order was changed")
	}
}

func TestPostgresScenarioGrantLockSerializesExternalRevocation(t *testing.T) {
	s := postgresTestState(t)
	r, _ := postgresTestRunners(t, s, nil)
	snapshot, err := r.Run(context.Background(), readScenarioInput{From: postgresSubmitInput().From})
	if err != nil {
		t.Fatal(err)
	}
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, []composition.Provider{{ID: "order-submit", Scope: "reference/order_entry/development", Offers: postgresSubmitServiceContract}}, nil); err != nil {
		t.Fatal(err)
	}
	selected, err := registry.Select("order-submit", "reference/order_entry/development")
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(release) }) }
	defer finish()
	service, err := composition.NewService(context.Background(), registry, selected,
		composition.Binding{ContractID: postgresSubmitServiceContract.ID, ContractVersion: "1", ProviderID: "order-submit", ArtifactDigest: s.digest, Generation: 1},
		func(composition.Binding) error { return nil },
		func(ctx context.Context, _ int) (int, error) {
			err := s.Submit(ctx, 1001, snapshot, func(context.Context) error { close(entered); <-release; return nil })
			return 0, err
		}, composition.Lifecycle{Prepare: func(context.Context, *composition.Scope) error { return nil }, Ready: func(context.Context, composition.Binding) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { finish(); _ = service.Withdraw(1); _ = service.Reap(context.Background()) })
	committed := make(chan error, 1)
	go func() { _, _, err := service.Invoke(context.Background(), 0); committed <- err }()
	select {
	case <-entered:
	case err := <-committed:
		t.Fatalf("submit failed before authorization barrier: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("submit did not reach authorization barrier")
	}
	conn, err := s.db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { finish(); _ = conn.Close() }()
	if _, err := conn.ExecContext(context.Background(), "SET application_name = 'tei-reference-revoke-check'"); err != nil {
		t.Fatal(err)
	}
	revoked := make(chan error, 1)
	go func() {
		_, err := conn.ExecContext(context.Background(), "UPDATE public.tei_reference_execution_grants SET revoked = true WHERE provider_id = 'order-submit'")
		revoked <- err
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting bool
		if err := s.db.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE application_name = 'tei-reference-revoke-check' AND wait_event_type = 'Lock')").Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("external revocation did not wait for commit grant lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	finish()
	if err := <-committed; err != nil {
		t.Fatal(err)
	}
	if err := <-revoked; err != nil {
		t.Fatal(err)
	}
	if postgresStatus(t, s.db) != "submitted" {
		t.Fatal("authorized earlier commit did not complete")
	}
	if _, _, err := service.Invoke(context.Background(), 0); err == nil {
		t.Fatal("later invocation ignored completed revocation")
	}
}
