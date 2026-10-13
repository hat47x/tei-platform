//go:build (tei_order_only && !tei_repair_only) || (tei_repair_only && !tei_order_only)

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"tei/internal/core"
	"tei/internal/ir"
)

// singleApplicationConfig contains startup inputs, not a new declaration of
// Application meaning, permissions, or Provider selection.
type singleApplicationConfig struct {
	LegacyUsers         string
	LegacyAuthorization string
}

type singleApplicationRunnerFactory func(ir.World, singleApplicationConfig) (
	readScenarioRunner, commandScenarioRunner, []webTargetOption, error,
)

// runSingleApplicationTarget deliberately has no runtime switch over
// Application-specific business implementations. The injected factory is
// fixed by the compiled main and uses the unchanged Driver-facing Runners.
func runSingleApplicationTarget(application string, factory singleApplicationRunnerFactory) error {
	flags := flag.NewFlagSet(application+"-web-target", flag.ContinueOnError)
	workspacePath := flags.String("workspace", "reference/workspace", "TEI workspace directory")
	requestedApplication := flags.String("application", application, "must match the compiled Application")
	baselineID := flags.String("baseline-id", "", "accepted Handover Baseline B")
	baselineRev := flags.String("baseline-revision", "", "accepted baseline repository revision R")
	targetRev := flags.String("target-revision", "", "Target build revision T")
	listen := flags.String("listen", "127.0.0.1:8080", "HTTP listen address")
	legacyUsers := flags.String("legacy-users", "reference/legacy-auth/users.json", "Reference legacy authentication fixture")
	legacyAuthorization := flags.String("legacy-authorization", "reference/legacy-auth/authorization.json", "Reference legacy authorization fixture")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if *requestedApplication != application {
		return fmt.Errorf("compiled Application %s refuses requested Application %q", application, *requestedApplication)
	}
	if *baselineID == "" || *baselineRev == "" || *targetRev == "" {
		return fmt.Errorf("-baseline-id, -baseline-revision and -target-revision are required")
	}
	// Reject non-loopback listeners before loading even synthetic fixtures.
	// The E1 JSON API has not acquired production Identity/TLS/Host authority;
	// existing presentation-bearing reference Targets keep their old contract.
	if err := validateReferenceListenAddress(*listen); err != nil {
		return err
	}
	// The diagnostic API must not describe a caller-invented Target build
	// revision T as the revision of the actual executable. This comparison
	// precedes Workspace/frozen-World or synthetic credential loading.
	if err := validateReferenceBuildRevision(*targetRev); err != nil {
		return err
	}
	appID, err := core.ParseCanonicalID(application)
	if err != nil {
		return err
	}
	world, err := loadSingleApplicationWorld(*workspacePath, *baselineID, *baselineRev, *targetRev)
	if err != nil {
		return fmt.Errorf("load reference Application model: %w", err)
	}
	projection, err := buildWebTargetProjection(world, projectionConfig{
		HandoverBaselineID:         *baselineID,
		BaselineRepositoryRevision: *baselineRev,
		TargetBuildRevision:        *targetRev,
		ApplicationID:              appID,
	})
	if err != nil {
		return err
	}
	read, command, options, err := factory(world, singleApplicationConfig{
		LegacyUsers:         *legacyUsers,
		LegacyAuthorization: *legacyAuthorization,
	})
	if err != nil {
		return err
	}
	target, err := newWebTargetServer(projection, read, command, options...)
	if err != nil {
		return err
	}
	// Each tagged binary registers only its own route group. This source
	// split alone does not establish real Provider or authorization trust.
	server := &http.Server{Addr: *listen, Handler: target.Handler()}
	log.Printf("TEI compiled %s reference Web Target: http://%s/ baseline=%s R=%s T=%s",
		application, *listen, *baselineID, *baselineRev, *targetRev)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve compiled %s reference target: %w", application, err)
	}
	return nil
}
