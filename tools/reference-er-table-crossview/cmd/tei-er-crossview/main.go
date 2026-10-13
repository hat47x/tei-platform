package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	ercrossview "tei/tools/reference-er-table-crossview"
	"tei/internal/workspace"
)

// CLI deliberately exports observations of authored model structure only.
// It cannot accept proposals, change sources, emit DDL, or authorize a DB.
func main() {
	root := flag.String("workspace", "", "source Workspace root to read")
	output := flag.String("out", "", "new JSON output file; will not overwrite")
	flag.Parse()
	if err := run(*root, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root, output string) error {
	if root == "" || output == "" {
		return fmt.Errorf("-workspace and -out are required")
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return fmt.Errorf("load Workspace: %w", err)
	}
	view, err := ercrossview.Project(loaded.World)
	if err != nil {
		return fmt.Errorf("project logical ER and physical table views: %w", err)
	}
	raw, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	path, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create new projection output: %w", err)
	}
	_, writeErr := file.Write(raw)
	closeErr := file.Close()
	if writeErr != nil {
		os.Remove(path)
		return writeErr
	}
	if closeErr != nil {
		os.Remove(path)
		return closeErr
	}
	return nil
}
