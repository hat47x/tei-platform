package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	temporalderivation "tei/tools/reference-temporal-derivation"
)

// tei-temporal-derivation labels one recalculation request and, only when the
// evidence allows it, computes a value. It never writes a Source and never
// updates a confirmed saved fact.
func main() {
	input := flag.String("input", "-", "path to a JSON request, or - for stdin")
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
	var in temporalderivation.Input
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		fmt.Fprintln(os.Stderr, "decode input:", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(temporalderivation.Evaluate(in)); err != nil {
		fmt.Fprintln(os.Stderr, "encode output:", err)
		os.Exit(1)
	}
}
