package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/workspace"
	plugincoverage "tei/tools/reference-plugin-capability-coverage"
)

// tei-plugin-capability-coverage prints which declared plugin capabilities an
// environment binds. It reads one Workspace and writes nothing.
func main() {
	root := flag.String("workspace", "", "Source Workspace root to read")
	flag.Parse()
	if *root == "" {
		fail(fmt.Errorf("-workspace is required"))
	}
	loaded, err := workspace.Load(*root)
	if err != nil {
		fail(fmt.Errorf("load Workspace: %w", err))
	}
	report := plugincoverage.Derive(loaded.World)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(2) }
