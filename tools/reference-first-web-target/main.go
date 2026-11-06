//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/sourceacceptance"
	"tei/internal/workspace"
)

func main() {
	// The shipped CLI injects no Reader. A Source Reader is selected only by
	// -host-acceptance-store with -host-acceptance-trust (signed Decisions,
	// provisional policy); labels and Decision JSON are not a substitute. No
	// effect Grant Reader can be selected by default, so accepted Commands are
	// still refused before dispatch. Only the explicit, development-only D8 flag
	// triple (-provisional-effect-grant*) selects a signed provisional Reader.
	// Unaccepted fixtures require explicit diagnostics.
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
	// browse replaces the fixed-row browse executor of the order_entry read
	// runner, for both the read endpoint and the read the submit Command
	// validates. nil keeps the fixture.
	browse driver.BrowseQueryExecutor
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
		provGrant     = flags.String("provisional-effect-grant", "", "DEVELOPMENT ONLY (D8): signed provisional effect Grant file (requires the two flags below)")
		provTrust     = flags.String("provisional-effect-grant-trust", "", "DEVELOPMENT ONLY (D8): trust file for provisional effect Grant issuers")
		provAck       = flags.Bool("provisional-effect-grant-development-only", false, "acknowledge that the provisional effect Grant is development/verification only, not a production permission, and not G2 evidence")
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
	// Opt-in, development-only effect Grant (D8). Without its flags this is a
	// no-op and the Reader stays whatever the embedding Host injected (nil for
	// the shipped CLI, so accepted Commands are still refused before dispatch).
	provisionalReader, err := selectProvisionalEffectGrant(
		provisionalEffectGrantFlags{grantPath: *provGrant, trustPath: *provTrust, developmentOnly: *provAck},
		string(appID), effectReader, *diagnostic, time.Now)
	if err != nil {
		return err
	}
	if provisionalReader != nil {
		effectReader = provisionalReader
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
		orderRead := newReferenceReadScenarioRunner(loaded.World)
		if wiring.browse != nil {
			orderRead.browse = wiring.browse
		}
		readScenario = orderRead
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
			command.hostEffect = func(effectCtx context.Context, subjectID, resourceID string) error {
				if ctx.Err() != nil {
					return errHostEffectGrantUnavailable
				}
				err := requireCurrentHostEffectGrantFor(effectCtx, reader, effectReader,
					string(appID), subjectID, "order_entry.submit_order", resourceID,
					acceptedWorkspace.SourceIndex, acceptedWorkspace.Acceptance)
				if err != nil || ctx.Err() != nil {
					return errHostEffectGrantUnavailable
				}
				return nil
			}
		}
		command.submitStore = submitStore
		if wiring.browse != nil {
			command.read = orderRead
		}
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
	if !*diagnostic && acceptedWorkspace.Acceptance.AcceptanceKind != "" {
		// D9: a provisional (or unmarked) Acceptance is not a formal human
		// Acceptance. Say so on stderr (the log package's default).
		log.Printf("SOURCE_ACCEPTANCE_%s: Source accepted under the D9 provisional approval (acceptanceKind=%s), not a formal human Acceptance; not G evidence and not a production deployment basis",
			strings.ToUpper(acceptedWorkspace.Acceptance.AcceptanceKind), acceptedWorkspace.Acceptance.AcceptanceKind)
	}
	// Bind first: a port that is already in use must fail the start BEFORE the
	// development-only Grant is announced as active.
	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		return fmt.Errorf("WEB_TARGET_LISTEN_FAILED: %w", err)
	}
	// Source verification, every startup gate and the bind have passed. Only now
	// say that the development-only Grant is active (a failed start must not claim it).
	if announcer, ok := provisionalReader.(interface{ announce() }); ok {
		announcer.announce()
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
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("WEB_TARGET_HOST_CONTEXT_UNAVAILABLE: %w", err)
	}
	return nil
}
