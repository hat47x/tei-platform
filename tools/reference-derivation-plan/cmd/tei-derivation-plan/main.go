package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	derivationplan "tei/tools/reference-derivation-plan"
)

// The CLI executes one checked query plan over derived values, or checks whether
// the plan can be delegated to SQL or evaluated after a complete fetch. It never
// writes a Source.
func main() {
	input := flag.String("input", "-", "path to a JSON request, or - for stdin")
	mode := flag.String("mode", "run", "run (checked plan) or applicability (SQL vs post-fetch)")
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
	case "run":
		var in derivationplan.Input
		if err := decode(reader, &in); err != nil {
			fail("decode input", err)
		}
		if err := enc.Encode(derivationplan.Run(in)); err != nil {
			fail("encode output", err)
		}
	case "applicability":
		var in derivationplan.ApplicabilityInput
		if err := decode(reader, &in); err != nil {
			fail("decode input", err)
		}
		if err := enc.Encode(derivationplan.CheckApplicability(in)); err != nil {
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
