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
	target, err := newWebTargetServer(projection)
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
