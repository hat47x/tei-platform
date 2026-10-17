package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/workspace"
	funcdoc "tei/tools/reference-function-definition-document"
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
	doc, err := funcdoc.Render(loaded.World)
	if err != nil {
		fail(err)
	}
	f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		fail(fmt.Errorf("create new document: %w", err))
	}
	if _, err := f.WriteString(doc.Markdown); err != nil {
		f.Close()
		os.Remove(*out)
		fail(err)
	}
	if err := f.Close(); err != nil {
		os.Remove(*out)
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
