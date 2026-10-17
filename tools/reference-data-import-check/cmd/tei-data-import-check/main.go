package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/workspace"
	dataimportcheck "tei/tools/reference-data-import-check"
)

// The CLI reads a Workspace and one TSV file and prints a shape report. It
// writes nothing and never prints a cell value.
func main() {
	root := flag.String("workspace", "", "source Workspace root to read")
	data := flag.String("data", "", "Data ID the rows belong to")
	in := flag.String("input", "", "TSV file with a header row")
	flag.Parse()
	if *root == "" || *data == "" || *in == "" {
		fail(fmt.Errorf("-workspace, -data and -input are required"))
	}
	loaded, err := workspace.Load(*root)
	if err != nil {
		fail(fmt.Errorf("load Workspace: %w", err))
	}
	f, err := os.Open(*in)
	if err != nil {
		fail(err)
	}
	defer f.Close()
	rep, err := dataimportcheck.Check(loaded.World, *data, f)
	if err != nil {
		fail(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rep); err != nil {
		fail(err)
	}
	if !rep.ShapeOK {
		os.Exit(3)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
