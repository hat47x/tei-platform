package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	auxiliary "tei/tools/reference-authoring-auxiliary"
)

// The CLI only classifies auxiliary information and prints a report. It cannot
// rewrite a source, accept a definition, or store a derived fact.
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

	var in auxiliary.Input
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		fmt.Fprintln(os.Stderr, "decode input:", err)
		os.Exit(1)
	}

	report := auxiliary.Analyze(in)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, "encode report:", err)
		os.Exit(1)
	}
}
