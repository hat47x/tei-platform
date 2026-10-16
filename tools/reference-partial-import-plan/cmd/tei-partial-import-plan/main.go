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

// The CLI reads two Workspaces and prints a plan. It writes nothing and
// cannot accept an import.
func main() {
	src := flag.String("source", "", "Workspace to import from")
	tgt := flag.String("target", "", "Workspace to import into")
	sel := flag.String("data", "", "comma-separated Data IDs to import")
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
	plan, err := partialimport.Analyze(s.World, t.World, strings.Split(*sel, ","))
	if err != nil {
		fail(err)
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
