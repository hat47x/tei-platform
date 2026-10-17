package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	generationgraph "tei/tools/reference-generation-graph"
)

// The CLI checks one phase-annotated transform graph or compares the operator
// contracts shared between two phases. It never executes an operator, never
// generates output, and never promotes a generated candidate to accepted
// business meaning.
func main() {
	input := flag.String("input", "-", "path to a JSON request, or - for stdin")
	flag.Parse()

	var in generationgraph.Input
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
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		fmt.Fprintln(os.Stderr, "decode input:", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(generationgraph.Decide(in)); err != nil {
		fmt.Fprintln(os.Stderr, "encode output:", err)
		os.Exit(1)
	}
}
