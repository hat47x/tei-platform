package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	rulebinding "tei/tools/reference-rule-binding"
)

// The CLI resolves one application target from synthetic rules and bindings, or
// lists the bindings affected by a rule change. It never writes a Source.
func main() {
	input := flag.String("input", "-", "path to a JSON request, or - for stdin")
	mode := flag.String("mode", "resolve", "resolve (one target) or impact (rule-change impact)")
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
	case "resolve":
		var in rulebinding.Input
		if err := decode(reader, &in); err != nil {
			fail("decode input", err)
		}
		if err := enc.Encode(rulebinding.Resolve(in)); err != nil {
			fail("encode output", err)
		}
	case "impact":
		var in rulebinding.ImpactInput
		if err := decode(reader, &in); err != nil {
			fail("decode input", err)
		}
		if err := enc.Encode(rulebinding.PlanImpact(in)); err != nil {
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
