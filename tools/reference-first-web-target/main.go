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
	// The shipped CLI injects no Reader. A Source Reader is selected only by
	// -host-acceptance-store with -host-acceptance-trust (signed Decisions,
	// provisional policy); labels and Decision JSON are not a substitute. No
	// effect Grant Reader can be selected here, so accepted Commands are still
	// refused before dispatch. Unaccepted fixtures require explicit diagnostics.
	if err := runCombinedWebTarget(context.Background(), nil, os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

// runCombinedWebTarget remains the existing embedding entry point. A Source
// Reader alone is never sufficient to authorize an accepted Command effect.
func runCombinedWebTarget(ctx context.Context, reader sourceacceptance.Reader, args []string) error {
	return runCombinedWebTargetWithEffectGrant(ctx, reader, nil, args)
}

// runCombinedWebTargetWithEffectGrant checks the Host's current Source before
// publishing HTTP, and independently requires current subject/action-specific
// Host effect admission immediately before a Command changes state. The Host
// must inject both authentic Readers; the CLI cannot manufacture either one.
func runCombinedWebTargetWithEffectGrant(ctx context.Context, reader sourceacceptance.Reader, effectReader HostEffectGrantReader, args []string) error {
	return runCombinedWebTargetWithHost(ctx, reader, effectReader, webHostWiring{}, args)
}

// webHostWiring is what an embedding Host (or a test acting as one) may add
// beyond the two Readers. The shipped CLI passes the zero value: it cannot
// select a real store, so it cannot write business state.
type webHostWiring struct {
	// submitStore makes the order_entry submit Command write real state.
	submitStore orderSubmitStore
	// wrapCommandRead wraps the read the submit Command validates. Tests use
	// it to present a changed synthetic fixture row; it is not a Provider.
	wrapCommandRead func(readScenarioRunner) readScenarioRunner
}

// runCombinedWebTargetWithHost is runCombinedWebTargetWithEffectGrant plus the
// Host's optional wiring.
func runCombinedWebTargetWithHost(ctx context.Context, reader sourceacceptance.Reader, effectReader HostEffectGrantReader, wiring webHostWiring, args []string) error {
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
		storeDir      = flags.String("host-acceptance-store", "", "existing Host Acceptance Store directory (requires -host-acceptance-trust)")
		trustPath     = flags.String("host-acceptance-trust", "", "Host trust file listing issuer public keys (requires -host-acceptance-store)")
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
	if (*storeDir == "") != (*trustPath == "") {
		return fmt.Errorf("HOST_ACCEPTANCE_STORE_FLAGS: -host-acceptance-store and -host-acceptance-trust are required together")
	}
	if *storeDir != "" {
		if reader != nil {
			return fmt.Errorf("HOST_ACCEPTANCE_STORE_CONFLICT: the embedding Host already supplied a Source Reader")
		}
		if *diagnostic {
			return fmt.Errorf("SOURCE_ACCEPTANCE_DIAGNOSTIC_CONFLICT: Host Readers cannot be bypassed")
		}
		store, err := openHostAcceptanceStore(*storeDir, *trustPath)
		if err != nil {
			return err
		}
		reader = store
	}
	var loaded workspace.Result
	var acceptedWorkspace sourceacceptance.VerifiedWorkspace
	if *diagnostic {
		if reader != nil || effectReader != nil {
			return fmt.Errorf("SOURCE_ACCEPTANCE_DIAGNOSTIC_CONFLICT: Host Readers cannot be bypassed")
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
		// A real store must match the accepted declaration before HTTP is
		// published, and again before each write (S02, physical side).
		submitStore := wiring.submitStore
		if submitStore != nil {
			if submitStore, err = requireOrderSubmitSchema(ctx, loaded.World, submitStore); err != nil {
				return err
			}
		}
		command, err := newReferenceCommandScenarioRunner(
			loaded.World,
			*legacyUsers,
			*legacyAuthz,
		)
		if err != nil {
			return err
		}
		if !*diagnostic {
			command.hostEffect = func(effectCtx context.Context, subjectID string) error {
				if ctx.Err() != nil {
					return errHostEffectGrantUnavailable
				}
				err := requireCurrentHostEffectGrant(effectCtx, reader, effectReader,
					string(appID), subjectID, "order_entry.submit_order",
					acceptedWorkspace.SourceIndex, acceptedWorkspace.Acceptance)
				if err != nil || ctx.Err() != nil {
					return errHostEffectGrantUnavailable
				}
				return nil
			}
		}
		command.submitStore = submitStore
		if wiring.wrapCommandRead != nil {
			command.read = wiring.wrapCommandRead(command.read)
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
		handler = guardCurrentAcceptedSource(ctx, handler, reader, string(appID),
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
	// The Host owns the lifetime of this HTTP endpoint. Do not leave a
	// previously verified Target serving requests when that Host exits.
	// Close promptly rather than waiting for new HTTP traffic to notice.
	stopped := make(chan struct{})
	defer close(stopped)
	go func() {
		select {
		case <-ctx.Done():
			_ = server.Close()
		case <-stopped:
		}
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("WEB_TARGET_HOST_CONTEXT_UNAVAILABLE: %w", err)
	}
	return nil
}
