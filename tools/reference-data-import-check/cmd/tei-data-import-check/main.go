package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/ir"
	"tei/internal/workspace"
	dataimportcheck "tei/tools/reference-data-import-check"
)

// The CLI reads a Workspace and one TSV file and prints a shape report. With
// -batch it reads several TSV files and checks the references between them. It
// writes nothing and never prints a cell value.
func main() {
	root := flag.String("workspace", "", "source Workspace root to read")
	data := flag.String("data", "", "Data ID the rows belong to")
	in := flag.String("input", "", "TSV file with a header row")
	batch := flag.String("batch", "", "JSON file listing several tables and their roles")
	flag.Parse()
	if *root == "" {
		fail(fmt.Errorf("-workspace is required"))
	}
	loaded, err := workspace.Load(*root)
	if err != nil {
		fail(fmt.Errorf("load Workspace: %w", err))
	}

	if *batch != "" {
		if *data != "" || *in != "" {
			fail(fmt.Errorf("-batch cannot be combined with -data or -input"))
		}
		runBatch(loaded.World, *batch)
		return
	}
	if *data == "" || *in == "" {
		fail(fmt.Errorf("-data and -input are required without -batch"))
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
	encode(rep)
	if !rep.ShapeOK {
		os.Exit(3)
	}
}

type batchSpec struct {
	Tables []struct {
		Data string `json:"data"`
		Role string `json:"role"`
		Path string `json:"path"`
	} `json:"tables"`
}

func runBatch(world ir.World, path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		fail(err)
	}
	var spec batchSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		fail(fmt.Errorf("decode %s: %w", path, err))
	}
	if len(spec.Tables) == 0 {
		fail(fmt.Errorf("%s lists no tables", path))
	}
	tables := make([]dataimportcheck.TableInput, 0, len(spec.Tables))
	for _, table := range spec.Tables {
		if table.Data == "" || table.Path == "" {
			fail(fmt.Errorf("each batch table needs data and path"))
		}
		rows, err := os.ReadFile(table.Path)
		if err != nil {
			fail(err)
		}
		tables = append(tables, dataimportcheck.TableInput{Data: table.Data, Role: table.Role, TSV: string(rows)})
	}
	rep, err := dataimportcheck.CheckBatch(world, tables)
	if err != nil {
		fail(err)
	}
	encode(rep)
	if !rep.OK {
		os.Exit(3)
	}
}

func encode(value interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
