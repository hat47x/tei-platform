package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/workspace"
	accessmatrix "tei/tools/reference-access-matrix"
)

// The CLI prints the matrix as JSON, or writes the Markdown to a new file.
// It never overwrites a file and never changes the Workspace.
func main() {
	root := flag.String("workspace", "", "source Workspace root to read")
	out := flag.String("out", "", "new Markdown file; omit to print JSON")
	flag.Parse()
	if *root == "" {
		fail(fmt.Errorf("-workspace is required"))
	}
	loaded, err := workspace.Load(*root)
	if err != nil {
		fail(fmt.Errorf("load Workspace: %w", err))
	}
	m, err := accessmatrix.Derive(loaded.World)
	if err != nil {
		fail(err)
	}
	if *out != "" {
		f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			fail(fmt.Errorf("create new document: %w", err))
		}
		if _, err := f.WriteString(m.Markdown); err != nil {
			f.Close()
			os.Remove(*out)
			fail(err)
		}
		if err := f.Close(); err != nil {
			os.Remove(*out)
			fail(err)
		}
		return
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
