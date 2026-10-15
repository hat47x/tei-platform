//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

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
		os.Exit(2)
	}
}

// runCombinedWebTarget checks the Host witness before constructing any runner
// or HTTP handler. The Reader's provenance and current effect grants remain
// the embedding Host's responsibility; this checks startup Source identity.
func runCombinedWebTarget(ctx context.Context, reader sourceacceptance.Reader, args []string) error {
	flags := flag.NewFlagSet("first-web-target", flag.ContinueOnError)
	var (
		workspacePath = flags.String("workspace", "reference/workspace", "TEI workspace directory")
		applicationID = flags.String("application", "order_entry", "Canonical Application ID")
		baselineID    = flags.String("baseline-id", "", "Handover Baseline ID, for example B")
		baselineRev   = flags.String("baseline-revision", "", "Handover baseline repository revision R")
		targetRev     = flags.String("target-revision", "", "Target build repository revision T")
		legacyUsers   = flags.String("legacy-users", "reference/legacy-auth/users.json", "Legacy authentication fixture")
		legacyAuthz   = flags.String("legacy-authorization", "reference/legacy-auth/authorization.json", "Legacy authorization fixture")
		listen        = flags.String("listen", "127.0.0.1:8080", "HTTP listen address")
		scopedRoutes  = flags.Bool("scoped-routes", false, "diagnostic: expose only the selected Application's HTTP routes")
		diagnostic    = flags.Bool("diagnostic-unaccepted-source", false, "test-only: load unaccepted Reference Source without a Host witness")
	)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}

	if *baselineID == "" || *baselineRev == "" || *targetRev == "" {
		return fmt.Errorf("-baseline-id, -baseline-revision and -target-revision are required")
	}
	appID, err := core.ParseCanonicalID(*applicationID)
	if err != nil {
		return err
	}
	var loaded workspace.Result
	var acceptedWorkspace sourceacceptance.VerifiedWorkspace
	if *diagnostic {
		if reader != nil {
			return fmt.Errorf("SOURCE_ACCEPTANCE_DIAGNOSTIC_CONFLICT: Host Reader cannot be bypassed")
		}
		log.Print("SOURCE_ACCEPTANCE_NOT_VERIFIED: explicit diagnostic-unaccepted-source")
		loaded, err = workspace.Load(*workspacePath)
	} else {
		acceptedWorkspace, err = sourceacceptance.LoadVerifiedWorkspace(ctx, reader, string(appID), *workspacePath)
		loaded = acceptedWorkspace.Loaded
	}
	if err != nil {
		return err
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
	case core.MustCanonicalID("repair_intake"):
		options = append(options, withCustomerLookup(newReferenceCustomerLookupScenarioRunner(loaded.World)))
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
	if !*diagnostic {
		// The Host decision used at startup must still be current before
		// every HTTP dispatch. A replacement decision requires a restart;
		// the compiled World is never implicitly upgraded.
		handler = guardCurrentAcceptedSource(handler, reader, string(appID),
			acceptedWorkspace.SourceIndex, acceptedWorkspace.Acceptance)
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
