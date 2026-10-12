//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/workspace"
)

func main() {
	var (
		workspacePath    = flag.String("workspace", "reference/workspace", "TEI workspace directory")
		applicationID    = flag.String("application", "order_entry", "Canonical Application ID")
		baselineID       = flag.String("baseline-id", "", "Handover Baseline ID, for example B")
		baselineRev      = flag.String("baseline-revision", "", "Handover baseline repository revision R")
		targetRev        = flag.String("target-revision", "", "Target build repository revision T")
		legacyUsers      = flag.String("legacy-users", "reference/legacy-auth/users.json", "Legacy authentication fixture")
		legacyAuthz      = flag.String("legacy-authorization", "reference/legacy-auth/authorization.json", "Legacy authorization fixture")
		listen           = flag.String("listen", "127.0.0.1:8080", "HTTP listen address")
		scopedRoutes     = flag.Bool("scoped-routes", false, "diagnostic: expose only the selected Application's HTTP routes")
		composeScenarios = flag.Bool("composition-scenarios", false, "reference: run the selected Application through statically linked service contracts")
		postgresDSNFile  = flag.String("postgres-dsn-file", "", "reference: host-owned PostgreSQL connection file; never creates fixture objects")
		postgresFence    = flag.Int64("postgres-fencing-token", 0, "reference: expected positive database execution-grant fencing token")
		modelRevision    = flag.String("model-revision", "", "reference: exact source revision explicitly selected by the host; does not accept proposals")
	)
	nonScreen := registerNonScreenFlags()
	flag.Parse()
	if *nonScreen.mode != "" && (!*composeScenarios || *postgresDSNFile == "") {
		fmt.Fprintln(os.Stderr, "-non-screen requires composed PostgreSQL execution")
		os.Exit(2)
	}
	if *modelRevision != "" && !*composeScenarios {
		fmt.Fprintln(os.Stderr, "-model-revision requires -composition-scenarios")
		os.Exit(2)
	}
	if (*postgresDSNFile != "" || *postgresFence != 0) && (!*composeScenarios || *postgresDSNFile == "" || *postgresFence <= 0) {
		fmt.Fprintln(os.Stderr, "PostgreSQL reference requires -composition-scenarios, -postgres-dsn-file and a positive -postgres-fencing-token")
		os.Exit(2)
	}
	if *postgresDSNFile != "" && *applicationID != "order_entry" {
		fmt.Fprintln(os.Stderr, "PostgreSQL reference currently requires -application order_entry")
		os.Exit(2)
	}

	if *baselineID == "" || *baselineRev == "" || *targetRev == "" {
		fmt.Fprintln(os.Stderr, "-baseline-id, -baseline-revision and -target-revision are required")
		os.Exit(2)
	}
	appID, err := core.ParseCanonicalID(*applicationID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	loaded, err := workspace.Load(*workspacePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	var model *scenarioModel
	if *modelRevision != "" {
		model, err = bindScenarioModel(*workspacePath, loaded, *modelRevision)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	projection, err := buildWebTargetProjection(loaded.World, projectionConfig{
		HandoverBaselineID:         *baselineID,
		BaselineRepositoryRevision: *baselineRev,
		TargetBuildRevision:        *targetRev,
		ApplicationID:              appID,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
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
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		commandScenario = command
		command.model = model
		if *composeScenarios {
			var state *postgresScenarioState
			digest, err := runningExecutableDigest()
			if err != nil {
				log.Fatal(err)
			}
			if *postgresDSNFile != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				db, err := openScenarioPostgres(ctx, *postgresDSNFile)
				cancel()
				if err != nil {
					log.Fatal(err)
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
			readScenario, commandScenario, err = wireComposedScenarios(context.Background(), registry, readScenario, command, digest, guard)
			if err != nil {
				log.Fatal(err)
			}
			if *nonScreen.mode != "" {
				if err := nonScreen.run(registry, readScenario, command, state, *workspacePath); err != nil {
					fmt.Fprintln(os.Stderr, err)
					// Close the owned connection before exiting with a failure status.
					_ = state.db.Close()
					os.Exit(1)
				}
				return
			}
		}
	case core.MustCanonicalID("repair_intake"):
		var lookup customerLookupRunner = newReferenceCustomerLookupScenarioRunner(loaded.World)
		if *composeScenarios {
			digest, err := runningExecutableDigest()
			if err != nil {
				log.Fatal(err)
			}
			lookup, err = wireComposedCustomerLookup(context.Background(), composition.NewRegistry(), lookup, model, digest,
				referenceInstanceGuard(digest, map[string]composition.Contract{"repair-customer": repairLookupContract}))
			if err != nil {
				log.Fatal(err)
			}
		}
		options = append(options, withCustomerLookup(lookup))
	default:
		fmt.Fprintf(os.Stderr, "no scenario runner exists for Application %s (supported: order_entry, repair_intake)\n", appID)
		os.Exit(2)
	}
	target, err := newWebTargetServer(projection, readScenario, commandScenario, options...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// Preserve the existing multi-Application Gate C diagnostic routes by
	// default. Scoped routing is explicitly opt-in and is NOT evidence that
	// the unused Go packages were removed from the binary.
	handler := target.Handler()
	if *scopedRoutes {
		handler, err = target.ScopedHandler()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
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
		log.Fatal(err)
	}
}
