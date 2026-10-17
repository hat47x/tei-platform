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
// a change proposal. With -candidate-out it also writes a new change candidate
// file, which must still pass the Shared Acceptance Boundary. It cannot write
// sources or accept a proposal.
func main() {
	root := flag.String("workspace", "", "source Workspace root to read")
	in := flag.String("propose", "", "edited list (CSV) to turn into a proposal; omit to export")
	candOut := flag.String("candidate-out", "", "new file for a set-symbol change candidate (needs -propose and -base-revision)")
	base := flag.String("base-revision", "", "target model revision from the Shared Acceptance revision command")
	flag.Parse()
	if *root == "" {
		fail(fmt.Errorf("-workspace is required"))
	}
	loaded, err := workspace.Load(*root)
	if err != nil {
		fail(fmt.Errorf("load Workspace: %w", err))
	}
	if *in == "" {
		if *candOut != "" {
			fail(fmt.Errorf("-candidate-out needs -propose"))
		}
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
	if *candOut != "" {
		raw, err := listroundtrip.Candidate(p, *base)
		if err != nil {
			fail(err)
		}
		out, err := os.OpenFile(*candOut, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			fail(fmt.Errorf("create candidate file: %w", err))
		}
		if _, err := out.Write(append(raw, '\n')); err != nil {
			out.Close()
			os.Remove(*candOut)
			fail(err)
		}
		if err := out.Close(); err != nil {
			os.Remove(*candOut)
			fail(err)
		}
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
