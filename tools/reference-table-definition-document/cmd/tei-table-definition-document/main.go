package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/workspace"
	tabledoc "tei/tools/reference-table-definition-document"
)

// The CLI writes a derived document to a new file and prints its receipt.
// It never overwrites an existing file and never changes the Workspace.
func main() {
	root := flag.String("workspace", "", "source Workspace root to read")
	out := flag.String("out", "", "new Markdown file; will not overwrite")
	flag.Parse()
	if *root == "" || *out == "" {
		fail(fmt.Errorf("-workspace and -out are required"))
	}
	loaded, err := workspace.Load(*root)
	if err != nil {
		fail(fmt.Errorf("load Workspace: %w", err))
	}
	doc, err := tabledoc.Render(loaded.World)
	if err != nil {
		fail(err)
	}
	path, err := filepath.Abs(*out)
	if err != nil {
		fail(err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		fail(fmt.Errorf("create new document: %w", err))
	}
	if _, err := f.WriteString(doc.Markdown); err != nil {
		f.Close()
		os.Remove(path)
		fail(err)
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		fail(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
