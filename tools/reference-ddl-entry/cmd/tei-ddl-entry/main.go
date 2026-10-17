package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/environment"
	"tei/internal/workspace"
	ddlentry "tei/tools/reference-ddl-entry"
)

// The CLI prints what the Provider can render. It writes no file and does not
// touch any database.
func main() {
	root := flag.String("workspace", "", "source Workspace root to read")
	database := flag.String("database", "", "database ID whose mappings to render")
	provider := flag.String("provider", "postgres", "database provider")
	flag.Parse()
	if *root == "" || *database == "" {
		fmt.Fprintln(os.Stderr, "-workspace and -database are required")
		os.Exit(1)
	}
	loaded, err := workspace.Load(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load Workspace:", err)
		os.Exit(1)
	}
	res, err := ddlentry.Render(loaded.World, *database, environment.DatabaseProvider(*provider))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(res); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
