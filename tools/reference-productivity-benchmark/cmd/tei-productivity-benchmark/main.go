package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	benchmark "tei/tools/reference-productivity-benchmark"
)

// tei-productivity-benchmark records and aggregates #2018 run data. It does
// not execute comparisons or claim productivity results.
func main() {
	if len(os.Args) < 2 || (os.Args[1] != "record" && os.Args[1] != "aggregate") {
		fmt.Fprintln(os.Stderr, "usage: tei-productivity-benchmark <record|aggregate> [options]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	input := fs.String("input", "", "run JSON for record")
	out := fs.String("out", "", "record output directory")
	dir := fs.String("dir", "", "run directory for aggregate")
	_ = fs.Parse(os.Args[2:])
	switch command {
	case "record":
		if *input == "" || *out == "" {
			fail(fmt.Errorf("-input and -out are required for record"))
		}
		run, err := benchmark.Record(*input, *out)
		if err != nil {
			fail(err)
		}
		emit(run)
	case "aggregate":
		if *dir == "" {
			fail(fmt.Errorf("-dir is required for aggregate"))
		}
		summary, err := benchmark.Aggregate(*dir)
		if err != nil {
			fail(err)
		}
		emit(summary)
	}
}

func emit(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
