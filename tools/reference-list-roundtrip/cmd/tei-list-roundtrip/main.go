package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/workspace"
	listroundtrip "tei/tools/reference-list-roundtrip"
)

// The CLI exports a list to standard output or reads an edited list and prints
// a change proposal. It cannot write sources or accept a proposal.
func main() {
	root := flag.String("workspace", "", "source Workspace root to read")
	in := flag.String("propose", "", "edited list (CSV) to turn into a proposal; omit to export")
	flag.Parse()
	if *root == "" {
		fail(fmt.Errorf("-workspace is required"))
	}
	loaded, err := workspace.Load(*root)
	if err != nil {
		fail(fmt.Errorf("load Workspace: %w", err))
	}
	if *in == "" {
		if err := listroundtrip.Export(os.Stdout, loaded.World); err != nil {
			fail(err)
		}
		return
	}
	f, err := os.Open(*in)
	if err != nil {
		fail(err)
	}
	defer f.Close()
	p, err := listroundtrip.Propose(loaded.World, f)
	if err != nil {
		fail(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(p); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
