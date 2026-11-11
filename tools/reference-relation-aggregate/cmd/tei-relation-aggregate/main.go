package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	relationaggregate "tei/tools/reference-relation-aggregate"
)

// The CLI checks a synthetic relation collection binding and prints the distinct
// total or a distinct refusal. It never writes a Source.
func main() {
	input := flag.String("input", "-", "path to a JSON request, or - for stdin")
	mode := flag.String("mode", "sum", "sum (collection binding) or reaggregate (reaggregation eligibility, #2593)")
	flag.Parse()

	var in relationaggregate.Input
	var claim relationaggregate.ReaggregationInput
	var request any = &in
	evaluate := func() any { return relationaggregate.Evaluate(in) }
	switch *mode {
	case "sum":
	case "reaggregate":
		request = &claim
		evaluate = func() any { return relationaggregate.EvaluateReaggregation(claim) }
	default:
		fmt.Fprintln(os.Stderr, "unknown mode:", *mode)
		os.Exit(1)
	}
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
	if err := dec.Decode(request); err != nil {
		fmt.Fprintln(os.Stderr, "decode input:", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(evaluate()); err != nil {
		fmt.Fprintln(os.Stderr, "encode output:", err)
		os.Exit(1)
	}
}
