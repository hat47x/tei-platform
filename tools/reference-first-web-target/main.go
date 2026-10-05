package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"tei/internal/core"
	"tei/internal/workspace"
)

func main() {
	var (
		workspacePath = flag.String("workspace", "reference/workspace", "TEI workspace directory")
		applicationID = flag.String("application", "order_entry", "Canonical Application ID")
		baselineID    = flag.String("baseline-id", "", "Handover Baseline ID, for example B")
		baselineRev   = flag.String("baseline-revision", "", "Handover baseline repository revision R")
		targetRev     = flag.String("target-revision", "", "Target build repository revision T")
		legacyUsers   = flag.String("legacy-users", "reference/legacy-auth/users.json", "Legacy authentication fixture")
		legacyAuthz   = flag.String("legacy-authorization", "reference/legacy-auth/authorization.json", "Legacy authorization fixture")
		listen        = flag.String("listen", "127.0.0.1:8080", "HTTP listen address")
	)
	flag.Parse()

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
	case core.MustCanonicalID("repair_intake"):
		options = append(options, withCustomerLookup(newReferenceCustomerLookupScenarioRunner(loaded.World)))
	default:
		fmt.Fprintf(os.Stderr, "no scenario runner exists for Application %s (supported: order_entry, repair_intake)\n", appID)
		os.Exit(2)
	}
	target, err := newWebTargetServer(projection, readScenario, commandScenario, options...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	server := &http.Server{
		Addr:    *listen,
		Handler: target.Handler(),
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
