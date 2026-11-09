package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/workspace"
	crosswalk "tei/tools/reference-operation-pattern-crosswalk"
)

// tei-operation-pattern-crosswalk prints the crosswalk as JSON.
// It only reads the Workspace and never writes an identifier back.
func main() {
	root := flag.String("workspace", "", "source Workspace root to read")
	flag.Parse()
	if *root == "" {
		fail(fmt.Errorf("-workspace is required"))
	}
	loaded, err := workspace.Load(*root)
	if err != nil {
		fail(fmt.Errorf("load Workspace: %w", err))
	}
	report := crosswalk.Derive(loaded.World)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
