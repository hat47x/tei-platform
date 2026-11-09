package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"tei/internal/workspace"
	partialimport "tei/tools/reference-partial-import-plan"
)

// The CLI reads two Workspaces and prints a plan. It writes no Workspace
// source; with -candidate-out it writes a new change candidate file that must
// still pass the Shared Acceptance Boundary.
func main() {
	src := flag.String("source", "", "Workspace to import from")
	tgt := flag.String("target", "", "Workspace to import into")
	sel := flag.String("data", "", "comma-separated Data IDs to import")
	refOnly := flag.String("reference-only-namespaces", "", "comma-separated namespaces the target uses read-only (#1570)")
	srcOwner := flag.String("source-owner", "", "owner of the source Workspace")
	tgtOwner := flag.String("target-owner", "", "owner of the target Workspace")
	candOut := flag.String("candidate-out", "", "new file for an add-data change candidate (needs -base-revision)")
	base := flag.String("base-revision", "", "target model revision from the Shared Acceptance revision command")
	flag.Parse()
	if *src == "" || *tgt == "" || *sel == "" {
		fail(fmt.Errorf("-source, -target and -data are required"))
	}
	s, err := workspace.Load(*src)
	if err != nil {
		fail(fmt.Errorf("load source: %w", err))
	}
	t, err := workspace.Load(*tgt)
	if err != nil {
		fail(fmt.Errorf("load target: %w", err))
	}
	opts := partialimport.Options{SourceOwner: *srcOwner, TargetOwner: *tgtOwner}
	if *refOnly != "" {
		opts.ReferenceOnlyNamespaces = strings.Split(*refOnly, ",")
	}
	plan, err := partialimport.AnalyzeWithOptions(s.World, t.World, strings.Split(*sel, ","), opts)
	if err != nil {
		fail(err)
	}
	if *candOut != "" {
		raw, err := partialimport.Candidate(plan, s.World, *base)
		if err != nil {
			fail(err)
		}
		f, err := os.OpenFile(*candOut, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			fail(fmt.Errorf("create candidate file: %w", err))
		}
		if _, err := f.Write(append(raw, '\n')); err != nil {
			f.Close()
			os.Remove(*candOut)
			fail(err)
		}
		if err := f.Close(); err != nil {
			os.Remove(*candOut)
			fail(err)
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(plan); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
