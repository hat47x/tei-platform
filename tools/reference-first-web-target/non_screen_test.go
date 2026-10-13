//go:build !tei_repair_only && !tei_frozen_world

package main

import (
	"context"
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

func postgresNonScreen(t *testing.T, state *postgresScenarioState, format, path string, progress func(batchItem)) *nonScreenServices {
	t.Helper()
	command, err := newReferenceCommandScenarioRunner(state.world, legacyFixturePath(t, "users.json"), legacyFixturePath(t, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	command.persistence = state
	command.model = state.model
	registry := composition.NewRegistry()
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
