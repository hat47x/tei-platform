package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/operations"
	driftcheck "tei/tools/reference-operation-drift-check"
)

// tei-operation-drift-check compares a stored operation surface to the meaning
// it was derived from. It reads one Workspace and prints a JSON report; it
// writes nothing.
func main() {
	root := flag.String("workspace", "", "Source Workspace root to read")
	application := flag.String("application", "", "Application canonical ID")
	expectedWorld := flag.String("expected-world-fingerprint", "", "WorldFingerprint the caller stored")
	expectedRevision := flag.String("expected-model-revision", "", "ModelRevision the caller stored")
	flag.Parse()
	if *root == "" || *application == "" {
		fail(fmt.Errorf("-workspace and -application are required"))
	}
	surface, err := operations.SurfaceWorkspace(*root, *application)
	if err != nil {
		fail(fmt.Errorf("derive operation surface: %w", err))
	}
	report := driftcheck.Check(surface, driftcheck.Expectation{
		WorldFingerprint: *expectedWorld,
		ModelRevision:    *expectedRevision,
	})
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fail(err)
	}
	if report.Status == driftcheck.StatusStale {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
