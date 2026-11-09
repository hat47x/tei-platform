package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	quantityarith "tei/tools/reference-quantity-arith"
)

// The CLI checks one synthetic arithmetic step against declared annotations, or
// checks whether the declared result can be delegated to a target. It never
// writes a Source and never turns a missing input into zero.
func main() {
	input := flag.String("input", "-", "path to a JSON request, or - for stdin")
	mode := flag.String("mode", "evaluate", "evaluate (one arithmetic step) or delegation (target capability)")
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

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	switch *mode {
	case "evaluate":
		var in quantityarith.Input
		if err := decode(reader, &in); err != nil {
			fail("decode input", err)
		}
		if err := enc.Encode(quantityarith.Evaluate(in)); err != nil {
			fail("encode output", err)
		}
	case "delegation":
		var in quantityarith.DelegationInput
		if err := decode(reader, &in); err != nil {
			fail("decode input", err)
		}
		if err := enc.Encode(quantityarith.CheckDelegation(in)); err != nil {
			fail("encode output", err)
		}
	default:
		fail("mode", fmt.Errorf("unknown mode %q", *mode))
	}
}

func decode(reader io.Reader, value interface{}) error {
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	return dec.Decode(value)
}

func fail(what string, err error) {
	fmt.Fprintln(os.Stderr, what+":", err)
	os.Exit(1)
}
