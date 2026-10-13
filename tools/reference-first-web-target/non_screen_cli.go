//go:build !tei_repair_only

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"tei/internal/composition"
)

type nonScreenFlags struct{ mode, from, rows, format, output, login, credential, orderIDs *string }

func (f nonScreenFlags) enabled() bool { return f.mode != nil && *f.mode != "" }

func registerNonScreenFlags(flags *flag.FlagSet) nonScreenFlags {
	return nonScreenFlags{
		flags.String("non-screen", "", "reference: report, batch or batch-observe; exits without serving HTTP"),
		flags.String("from", "2026-09-18T08:30:00Z", "reference: selected order search lower bound"),
		flags.String("rows", "0", "reference: comma-separated selected row indices, at most 100"),
		flags.String("report-format", "csv", "reference: csv or json renderer"),
		flags.String("artifact-out", "", "reference: new output file outside model workspace"),
		flags.String("login-id", "legacy-order-editor", "reference: batch login identity"),
		flags.String("credential-file", "", "reference: private one-line batch credential file"),
		flags.String("order-ids", "", "reference: explicit order identities for read-only batch observation"),
	}
}
func (f nonScreenFlags) run(registry *composition.Registry, read readScenarioRunner, command *referenceCommandScenarioRunner, state *postgresScenarioState, workspace string) error {
	if *f.mode == "batch-observe" {
		ids, err := parseObservationIDs(*f.orderIDs)
		if err != nil {
			return err
		}
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		service, err := wireBatchObservation(ctx, registry, state)
		if err != nil {
			return err
		}
		defer func() { _ = service.Withdraw(1); _ = service.Reap(context.Background()) }()
		result, binding, err := service.Invoke(ctx, ids)
		if err != nil {
			return err
		}
		result.Composition = projectCompositionBinding(binding)
		if state.model != nil {
			result.Composition.ModelRevision = state.model.revision
		}
		return json.NewEncoder(os.Stdout).Encode(result)
	}
	if *f.mode != "report" && *f.mode != "batch" {
		return fmt.Errorf("-non-screen must be report, batch or batch-observe")
	}
	input := nonScreenInput{From: *f.from, LoginID: *f.login}
	for _, part := range strings.Split(*f.rows, ",") {
		row, err := strconv.Atoi(part)
		if err != nil || row < 0 {
			return fmt.Errorf("invalid selected rows")
		}
		input.Rows = append(input.Rows, row)
	}
	if *f.mode == "report" && *f.output == "" {
		return fmt.Errorf("report requires -artifact-out")
	}
	if *f.mode == "batch" {
		b, err := os.ReadFile(*f.credential)
		if err != nil || len(b) > 4096 {
			return fmt.Errorf("batch requires a readable private credential file")
		}
		input.Credential = strings.TrimSuffix(strings.TrimSuffix(string(b), "\n"), "\r")
		if input.Credential == "" || strings.ContainsAny(input.Credential, "\r\n") {
			return fmt.Errorf("invalid credential file")
		}
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	s, err := wireNonScreen(ctx, registry, read, command, state, workspace, *f.output, *f.format, func(item batchItem) { _ = json.NewEncoder(os.Stderr).Encode(item) })
	if err != nil {
		return err
	}
	defer s.Close()
	if *f.mode == "report" {
		r, b, err := s.report.Invoke(ctx, input)
		r.Composition = projectCompositionBinding(b)
		if state.model != nil {
			r.Composition.ModelRevision = state.model.revision
		}
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(r)
	}
	r, b, err := s.batch.Invoke(ctx, input)
	r.Composition = projectCompositionBinding(b)
	if state.model != nil {
		r.Composition.ModelRevision = state.model.revision
	}
	if encodeErr := json.NewEncoder(os.Stdout).Encode(r); encodeErr != nil {
		return encodeErr
	}
	if err != nil {
		return fmt.Errorf("batch stopped; inspect receipt before retry")
	}
	if r.Stage != "completed" {
		return fmt.Errorf("batch stopped with business outcome; inspect receipt")
	}
	return nil
}
