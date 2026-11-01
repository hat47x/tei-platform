package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	accountingperiod "tei/tools/reference-accounting-period"
)

// tei-accounting-period decides one action over accounting periods, or replays
// business events in their declared order. It writes nothing.
func main() {
	input := flag.String("input", "-", "path to a JSON request, or - for stdin")
	mode := flag.String("mode", "evaluate", "evaluate (one action) or replay (ordered events)")
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
		var in accountingperiod.Input
		if err := decode(reader, &in); err != nil {
			fail("decode input", err)
		}
		if err := enc.Encode(accountingperiod.Evaluate(in)); err != nil {
			fail("encode output", err)
		}
	case "replay":
		var in accountingperiod.ReplayInput
		if err := decode(reader, &in); err != nil {
			fail("decode input", err)
		}
		if err := enc.Encode(accountingperiod.Replay(in)); err != nil {
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
