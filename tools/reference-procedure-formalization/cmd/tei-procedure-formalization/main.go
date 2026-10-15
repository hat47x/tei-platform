package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	procedureform "tei/tools/reference-procedure-formalization"
)

// The CLI only derives questions, applies answers, and prints a report. It cannot
// write a Source, fill a missing value, or accept a candidate on its own.
func main() {
	input := flag.String("input", "-", "path to a JSON analysis request, or - for stdin")
	flag.Parse()

	var reader io.Reader = os.Stdin
	if *input != "-" {
		f, err := os.Open(*input)
		if err != nil {
			fmt.Fprintln(os.Stderr, "open input:", err)
			os.Exit(1)
		}
		defer f.Close()
		reader = f
	}

	var in procedureform.Input
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		fmt.Fprintln(os.Stderr, "decode input:", err)
		os.Exit(1)
	}

	report := procedureform.Analyze(in)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, "encode report:", err)
		os.Exit(1)
	}
}
