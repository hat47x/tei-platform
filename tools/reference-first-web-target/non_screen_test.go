//go:build !tei_repair_only && !tei_frozen_world

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/composition"
	renderer "tei/plugins/report-renderer"
)

func TestReportStorageRejectsWorkspaceAliasesAndExistingFiles(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	doc := renderer.Document{Bytes: []byte("reference"), MediaType: "text/plain"}
	if _, err := publishReport(context.Background(), root, filepath.Join(root, "output.csv"), doc); err == nil {
		t.Fatal("wrote into model workspace")
	}
	alias := filepath.Join(outside, "alias")
	if err := os.Symlink(root, alias); err == nil {
		if _, err := publishReport(context.Background(), root, filepath.Join(alias, "output.csv"), doc); err == nil {
			t.Fatal("workspace symlink bypass")
		}
	}
	path := filepath.Join(outside, "output.csv")
	if _, err := publishReport(context.Background(), root, path, doc); err != nil {
		t.Fatal(err)
	}
	if _, err := publishReport(context.Background(), root, path, renderer.Document{Bytes: []byte("overwrite")}); err == nil {
		t.Fatal("overwrote existing report")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "reference" {
		t.Fatal("changed previous report")
	}
}

func postgresNonScreen(t *testing.T, state *postgresScenarioState, format, path string, progress func(batchItem), registries ...**composition.Registry) *nonScreenServices {
	t.Helper()
	command, err := newReferenceCommandScenarioRunner(state.world, legacyFixturePath(t, "users.json"), legacyFixturePath(t, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	command.persistence = state
	command.model = state.model
	registry := composition.NewRegistry()
	if len(registries) > 0 {
		*registries[0] = registry
	}
	read, submit, err := wireComposedScenarios(context.Background(), registry, &postgresReadScenario{state: state}, command, state.digest, func(composition.Binding) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	s, err := wireNonScreen(context.Background(), registry, read, command, state, referenceWorkspacePath(t), path, format, progress)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		s.Close()
		_ = submit.service.Withdraw(1)
		_ = submit.service.Reap(context.Background())
		_ = read.service.Withdraw(1)
		_ = read.service.Reap(context.Background())
	})
	return s
}
func nonScreenTestInput() nonScreenInput {
	return nonScreenInput{From: "2026-09-18T00:00:00Z", Rows: []int{0, 1, 2}, LoginID: "legacy-order-editor", Credential: "editor-secret"}
}

func TestPostgresScenarioBatchDeclaresPersistentSubmitDependency(t *testing.T) {
	state := postgresTestState(t)
	var registry *composition.Registry
	s := postgresNonScreen(t, state, "csv", "", nil, &registry)
	selection, err := registry.Select("order-batch", "reference/order_entry/development")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, epoch := range selection.DependencyEpochs {
		if epoch.ProviderID == "order-submit" {
			found = true
		}
	}
	if !found {
		t.Fatal("batch ignores persistent submit declaration")
	}
	snapshot, err := registry.Apply(registry.Snapshot().Revision, nil, []string{"order-submit"})
	if err != nil {
		t.Fatal(err)
	}
	if assessment, exists := snapshot.Assessment("order-batch"); !exists || assessment.Eligible {
		t.Fatal("batch declaration did not remain present and become ineligible on submit loss")
	}
	if _, _, err := s.batch.Invoke(context.Background(), nonScreenTestInput()); err == nil || statusOf(t, state, 1001) != "draft" {
		t.Fatal("removed submit declaration still allowed a batch effect")
	}
}
func statusOf(t *testing.T, state *postgresScenarioState, id int64) string {
	t.Helper()
	var status string
	if err := state.db.QueryRow("SELECT status FROM public.orders WHERE order_id=$1", id).Scan(&status); err != nil {
		t.Fatal(err)
	}
	return status
}
func TestPostgresScenarioNonScreenReportExchangeAndRevocation(t *testing.T) {
	state := postgresTestState(t)
	path := filepath.Join(t.TempDir(), "orders.csv")
	s := postgresNonScreen(t, state, "csv", path, nil)
	r, binding, err := s.report.Invoke(context.Background(), nonScreenTestInput())
	if err != nil || r.Rows != 4 || binding.ProviderID != "order-report" || r.Renderer.ProviderID != "report-csv" {
		t.Fatalf("report: %+v %v", r, err)
	}
	b, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(b), "1001,1,ITEM-A,2,4000") {
		t.Fatalf("CSV artifact: %s %v", b, err)
	}
	jsonPath := filepath.Join(t.TempDir(), "orders.json")
	j := postgresNonScreen(t, state, "json", jsonPath, nil)
	r, _, err = j.report.Invoke(context.Background(), nonScreenTestInput())
	if err != nil || r.Rows != 4 || r.Renderer.ProviderID != "report-json" || r.MediaType != "application/json" {
		t.Fatalf("exchanged renderer: %+v %v", r, err)
	}
	if _, err := state.db.Exec("UPDATE public.tei_reference_execution_grants SET revoked=true WHERE provider_id='order-report'"); err != nil {
		t.Fatal(err)
	}
	deniedPath := filepath.Join(t.TempDir(), "denied.csv")
	d := postgresNonScreen(t, state, "csv", deniedPath, nil)
	if _, _, err := d.report.Invoke(context.Background(), nonScreenTestInput()); err == nil {
		t.Fatal("revoked report published")
	}
	if _, err := os.Stat(deniedPath); !os.IsNotExist(err) {
		t.Fatal("revoked report created artifact")
	}
}

func TestPostgresScenarioCompletedBatchSurvivesReportedPluginCleanupFailure(t *testing.T) {
	state := postgresTestState(t)
	s := postgresNonScreen(t, state, "csv", "", nil)
	if err := s.render.Shutdown(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, []composition.Provider{{ID: "report-csv", Scope: "reference/order_entry/development", Offers: renderer.Contract}}, nil); err != nil {
		t.Fatal(err)
	}
	selected, err := registry.Select("report-csv", "reference/order_entry/development")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.CreateTemp(t.TempDir(), "renderer-resource-")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	cleanupCalls := 0
	lifecycle := composition.Lifecycle{Prepare: func(_ context.Context, scope *composition.Scope) error {
		return scope.Track(func() error { cleanupCalls++; return file.Close() })
	}, Ready: func(context.Context, composition.Binding) error { return nil }}
	binding := composition.Binding{ContractID: renderer.Contract.ID, ContractVersion: renderer.Contract.Version, ProviderID: "report-csv", ArtifactDigest: state.digest, Generation: 1}
	s.render, err = composition.NewService(context.Background(), registry, selected, binding, func(composition.Binding) error { return nil }, renderer.CSV, lifecycle)
	if err != nil {
		t.Fatal(err)
	}
	input := nonScreenTestInput()
	input.Rows = []int{0, 2}
	result, _, err := s.batch.Invoke(context.Background(), input)
	if err != nil || result.Stage != "completed" {
		t.Fatalf("batch: %+v %v", result, err)
	}
	closeErr := s.Close()
	if !errors.Is(closeErr, os.ErrClosed) || cleanupCalls != 1 {
		t.Fatalf("cleanup failure lost: %v", closeErr)
	}
	if !errors.Is(s.Close(), os.ErrClosed) || cleanupCalls != 1 {
		t.Fatal("cleanup failure forgotten or retried")
	}
	if _, _, err := s.report.Invoke(context.Background(), input); !errors.Is(err, composition.ErrUnbound) {
		t.Fatal("other services still callable after cleanup failure")
	}
	if statusOf(t, state, 1001) != "submitted" || statusOf(t, state, 1003) != "submitted" {
		t.Fatal("cleanup failure rolled back durable business result")
	}
}
func TestPostgresScenarioNonScreenBatchCancelReplayAndExplicitRemainder(t *testing.T) {
	state := postgresTestState(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := postgresNonScreen(t, state, "csv", "", func(batchItem) { cancel() })
	input := nonScreenTestInput()
	input.Rows = []int{0, 2}
	r, _, err := s.batch.Invoke(ctx, input)
	if !errors.Is(err, context.Canceled) || r.Stage != "cancelled" || len(r.Items) != 1 || r.Items[0].Result.Stage != "completed" || r.Items[0].Result.Observation.Nature != "external-observed" {
		t.Fatalf("cancel receipt: %+v %v", r, err)
	}
	if statusOf(t, state, 1001) != "submitted" || statusOf(t, state, 1003) != "draft" {
		t.Fatal("cancellation lost completed effect or ran remaining item")
	}
	plain := postgresNonScreen(t, state, "csv", "", nil)
	r, _, err = plain.batch.Invoke(context.Background(), input)
	if err == nil || r.Stage != "execution-failed" || len(r.Items) != 1 || statusOf(t, state, 1003) != "draft" {
		t.Fatalf("replay retried or silently resumed: %+v %v", r, err)
	}
	input.Rows = []int{2}
	r, _, err = plain.batch.Invoke(context.Background(), input)
	if err != nil || r.Stage != "completed" || statusOf(t, state, 1003) != "submitted" {
		t.Fatalf("explicit remainder failed: %+v %v", r, err)
	}
}
func TestPostgresScenarioNonScreenBatchBusinessFailureAndDuplicatePlan(t *testing.T) {
	state := postgresTestState(t)
	s := postgresNonScreen(t, state, "csv", "", nil)
	input := nonScreenTestInput()
	input.Rows = []int{0, 0}
	if _, _, err := s.batch.Invoke(context.Background(), input); err == nil || statusOf(t, state, 1001) != "draft" {
		t.Fatal("duplicate plan had an effect")
	}
	input.Rows = []int{0, 1, 2}
	r, _, err := s.batch.Invoke(context.Background(), input)
	if err != nil || r.Stage != "business-stopped" || len(r.Items) != 2 || r.Items[1].Result.Stage != "validation-not-satisfied" || statusOf(t, state, 1001) != "submitted" || statusOf(t, state, 1002) != "draft" || statusOf(t, state, 1003) != "draft" {
		t.Fatalf("partial business failure: %+v %v", r, err)
	}
}

type failingBatchJournal struct {
	kind   string
	events []batchJournalEvent
}

func (j *failingBatchJournal) Append(e batchJournalEvent) error {
	if e.Kind == j.kind {
		return errors.New("injected progress storage failure")
	}
	j.events = append(j.events, e)
	return nil
}

func TestPostgresScenarioBatchJournalFailureStopsAtEffectBoundary(t *testing.T) {
	for _, kind := range []string{"planned", "starting", "result"} {
		t.Run(kind, func(t *testing.T) {
			state := postgresTestState(t)
			s := postgresNonScreen(t, state, "csv", "", nil)
			journal := &failingBatchJournal{kind: kind}
			input := nonScreenTestInput()
			input.Rows = []int{0, 2}
			input.Journal = journal
			result, _, err := s.batch.Invoke(context.Background(), input)
			if err == nil || result.Stage != "journal-failed" || statusOf(t, state, 1003) != "draft" {
				t.Fatalf("failure ignored: %+v %v", result, err)
			}
			expected := "draft"
			if kind == "result" {
				expected = "submitted"
				if len(result.Items) != 1 || result.Items[0].Result.Stage != "completed" {
					t.Fatal("completed effect hidden")
				}
			}
			if statusOf(t, state, 1001) != expected {
				t.Fatal("effect boundary changed")
			}
		})
	}
}

func TestPostgresScenarioBatchUncertainCommitIsReviewedWithoutRetry(t *testing.T) {
	state := postgresTestState(t)
	calls := 0
	state.commit = func(tx *sql.Tx) error {
		calls++
		if err := tx.Commit(); err != nil {
			return err
		}
		return errors.New("injected lost acknowledgement after actual commit")
	}
	s := postgresNonScreen(t, state, "csv", "", nil)
	journal, err := openBatchJournal(referenceWorkspacePath(t), filepath.Join(t.TempDir(), "uncertain.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.file.Close()
	input := nonScreenTestInput()
	input.Rows = []int{0, 2}
	input.Journal = journal
	result, _, err := s.batch.Invoke(context.Background(), input)
	if !errors.Is(err, errPostgresCommitUnknown) || calls != 1 || result.Stage != "execution-failed" || len(result.Items) != 1 || result.Items[0].Result.Command != nil ||
		statusOf(t, state, 1001) != "submitted" || statusOf(t, state, 1003) != "draft" {
		t.Fatalf("uncertain commit retried or hidden: %+v %v", result, err)
	}
	raw, err := os.ReadFile(journal.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	review, ids, err := decodeBatchReview(raw)
	if err != nil || len(ids) != 2 || review.Items[0].RecordedStage != "command" || review.Items[1].Recorded != "not-recorded-as-started" {
		t.Fatalf("uncertain review: %+v %v", review, err)
	}
	observer, err := wireBatchObservation(context.Background(), composition.NewRegistry(), state)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = observer.Withdraw(1); _ = observer.Reap(context.Background()) }()
	current, _, err := observer.Invoke(context.Background(), ids)
	if err != nil || current.Items[0].Observation.Result != "submitted" || current.Items[1].Observation.Result != "draft" || calls != 1 {
		t.Fatal("review retried command")
	}
	review.Current = &current
	b, _ := json.Marshal(review)
	if strings.Contains(string(b), "editor-secret") {
		t.Fatal("credential leaked")
	}
}
