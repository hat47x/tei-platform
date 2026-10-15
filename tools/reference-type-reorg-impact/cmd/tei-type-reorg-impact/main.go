package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"tei/internal/workspace"
	typereorg "tei/tools/reference-type-reorg-impact"
)

// The CLI only prints the impact of a proposed type merge. It cannot rewrite
// sources or accept the merge.
func main() {
	root := flag.String("workspace", "", "source Workspace root to read")
	from := flag.String("from", "", "comma-separated type IDs to merge away")
	into := flag.String("into", "", "type ID to merge into")
	flag.Parse()
	if *root == "" || *from == "" || *into == "" {
		fmt.Fprintln(os.Stderr, "-workspace, -from and -into are required")
		os.Exit(1)
	}
	loaded, err := workspace.Load(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load Workspace:", err)
		os.Exit(1)
	}
	impact, err := typereorg.Analyze(loaded.World, typereorg.Merge{From: strings.Split(*from, ","), Into: *into})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(impact); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
