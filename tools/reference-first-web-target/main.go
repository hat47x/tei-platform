//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/sourceacceptance"
	"tei/internal/workspace"
)

func main() {
	// The Reference CLI has no authoritative Host acceptance service yet.
	// A Host integration must inject its Reader here; labels and Decision JSON
	// are not a substitute. Unaccepted fixtures require explicit diagnostics.
	if err := runCombinedWebTarget(context.Background(), nil, os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		var execution *nonScreenCommandError
		if errors.As(err, &execution) {
			os.Exit(1)
		}
		os.Exit(2)
	}
}

type nonScreenCommandError struct{ error }

func (err *nonScreenCommandError) Unwrap() error { return err.error }

// runCombinedWebTarget checks the Host witness before constructing any runner
// or HTTP handler. The Reader's provenance and current effect grants remain
// the embedding Host's responsibility; this checks startup Source identity.
func runCombinedWebTarget(ctx context.Context, reader sourceacceptance.Reader, args []string) error {
	flags := flag.NewFlagSet("first-web-target", flag.ContinueOnError)
	var (
		workspacePath    = flags.String("workspace", "reference/workspace", "TEI workspace directory")
		applicationID    = flags.String("application", "order_entry", "Canonical Application ID")
		baselineID       = flags.String("baseline-id", "", "Handover Baseline ID, for example B")
		baselineRev      = flags.String("baseline-revision", "", "Handover baseline repository revision R")
		targetRev        = flags.String("target-revision", "", "Target build repository revision T")
		legacyUsers      = flags.String("legacy-users", "reference/legacy-auth/users.json", "Legacy authentication fixture")
		legacyAuthz      = flags.String("legacy-authorization", "reference/legacy-auth/authorization.json", "Legacy authorization fixture")
		listen           = flags.String("listen", "127.0.0.1:8080", "HTTP listen address")
		scopedRoutes     = flags.Bool("scoped-routes", false, "diagnostic: expose only the selected Application's HTTP routes")
		composeScenarios = flags.Bool("composition-scenarios", false, "reference: run the selected Application through statically linked service contracts")
		postgresDSNFile  = flags.String("postgres-dsn-file", "", "reference: host-owned PostgreSQL connection file; never creates fixture objects")
		postgresFence    = flags.Int64("postgres-fencing-token", 0, "reference: expected positive database execution-grant fencing token")
		modelRevision    = flags.String("model-revision", "", "reference: exact source revision explicitly selected by the host; does not accept proposals")
		diagnostic       = flags.Bool("diagnostic-unaccepted-source", false, "test-only: load unaccepted Reference Source without a Host witness")
	)
	nonScreen := registerNonScreenFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if *nonScreen.mode != "" && (!*composeScenarios || *postgresDSNFile == "") {
		return fmt.Errorf("-non-screen requires composed PostgreSQL execution")
	}
	if *modelRevision != "" && !*composeScenarios {
		return fmt.Errorf("-model-revision requires -composition-scenarios")
	}
	if (*postgresDSNFile != "" || *postgresFence != 0) && (!*composeScenarios || *postgresDSNFile == "" || *postgresFence <= 0) {
		return fmt.Errorf("PostgreSQL reference requires -composition-scenarios, -postgres-dsn-file and a positive -postgres-fencing-token")
	}
	if *postgresDSNFile != "" && *applicationID != "order_entry" {
		return fmt.Errorf("PostgreSQL reference currently requires -application order_entry")
	}

	if *baselineID == "" || *baselineRev == "" || *targetRev == "" {
		return fmt.Errorf("-baseline-id, -baseline-revision and -target-revision are required")
	}
	appID, err := core.ParseCanonicalID(*applicationID)
	if err != nil {
		return err
	}
	var loaded workspace.Result
	if *diagnostic {
		if reader != nil {
			return fmt.Errorf("SOURCE_ACCEPTANCE_DIAGNOSTIC_CONFLICT: Host Reader cannot be bypassed")
		}
		log.Print("SOURCE_ACCEPTANCE_NOT_VERIFIED: explicit diagnostic-unaccepted-source")
		loaded, err = workspace.Load(*workspacePath)
	} else {
		var verified sourceacceptance.VerifiedWorkspace
		verified, err = sourceacceptance.LoadVerifiedWorkspace(ctx, reader, string(appID), *workspacePath)
		loaded = verified.Loaded
	}
	if err != nil {
		return err
	}
	var model *scenarioModel
	if *modelRevision != "" {
		model, err = bindScenarioModel(*workspacePath, loaded, *modelRevision)
		if err != nil {
			return err
		}
	}
	projection, err := buildWebTargetProjection(loaded.World, projectionConfig{
		HandoverBaselineID:         *baselineID,
		BaselineRepositoryRevision: *baselineRev,
		TargetBuildRevision:        *targetRev,
		ApplicationID:              appID,
	})
	if err != nil {
		return err
	}
	if model != nil {
		projection.ModelRevision = model.revision
	}
	// Each Application gets only the scenario runners that exist for it. A runner for another
	// Application is never attached, and newWebTargetServer rejects one that is.
	var (
		readScenario    readScenarioRunner
		commandScenario commandScenarioRunner
		options         []webTargetOption
	)
	switch appID {
	case core.MustCanonicalID("order_entry"):
		readScenario = newReferenceReadScenarioRunner(loaded.World)
		command, err := newReferenceCommandScenarioRunner(
			loaded.World,
			*legacyUsers,
			*legacyAuthz,
		)
		if err != nil {
			return err
		}
		commandScenario = command
		command.model = model
		if *composeScenarios {
			var state *postgresScenarioState
			digest, err := runningExecutableDigest()
			if err != nil {
				return err
			}
			if *postgresDSNFile != "" {
				connectionCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				db, err := openScenarioPostgres(connectionCtx, *postgresDSNFile)
				cancel()
				if err != nil {
					return err
				}
				defer db.Close()
				state = &postgresScenarioState{db: db, world: loaded.World, digest: digest, fence: *postgresFence, model: model}
				readScenario = &postgresReadScenario{state: state}
				command.persistence = state
			}
			// This host permits only its two statically linked reference instances.
			// It does not grant production permissions or replace business checks.
			guard := func(b composition.Binding) error {
				if b.ArtifactDigest != digest || b.Generation != 1 ||
					!((b.ProviderID == "order-read" && b.ContractID == readServiceContract.ID) ||
						(b.ProviderID == "order-submit" && ((command.persistence == nil && b.ContractID == submitServiceContract.ID) || (command.persistence != nil && b.ContractID == postgresSubmitServiceContract.ID)))) {
					return fmt.Errorf("reference host rejects instance binding")
				}
				return nil
			}
			registry := composition.NewRegistry()
			readScenario, commandScenario, err = wireComposedScenarios(ctx, registry, readScenario, command, digest, guard)
			if err != nil {
				return err
			}
			if *nonScreen.mode != "" {
				if err := nonScreen.run(registry, readScenario, command, state, *workspacePath); err != nil {
					return &nonScreenCommandError{err}
				}
				return nil
			}
		}
	case core.MustCanonicalID("repair_intake"):
		var lookup customerLookupRunner = newReferenceCustomerLookupScenarioRunner(loaded.World)
		if *composeScenarios {
			digest, err := runningExecutableDigest()
			if err != nil {
				return err
			}
			lookup, err = wireComposedCustomerLookup(ctx, composition.NewRegistry(), lookup, model, digest,
				referenceInstanceGuard(digest, map[string]composition.Contract{"repair-customer": repairLookupContract}))
			if err != nil {
				return err
			}
		}
		options = append(options, withCustomerLookup(lookup))
	default:
		return fmt.Errorf("no scenario runner exists for Application %s (supported: order_entry, repair_intake)", appID)
	}
	target, err := newWebTargetServer(projection, readScenario, commandScenario, options...)
	if err != nil {
		return err
	}

	// Preserve the existing multi-Application Gate C diagnostic routes by
	// default. Scoped routing is explicitly opt-in and is NOT evidence that
	// the unused Go packages were removed from the binary.
	handler := target.Handler()
	if *scopedRoutes {
		handler, err = target.ScopedHandler()
		if err != nil {
			return err
		}
	}
	server := &http.Server{
		Addr:    *listen,
		Handler: handler,
	}
	log.Printf(
		"TEI first Web Target projection: http://%s/ application=%s baseline=%s R=%s T=%s",
		*listen,
		appID,
		*baselineID,
		*baselineRev,
		*targetRev,
	)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
