package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	inventory "tei/tools/reference-studio-plugin-inventory"
)

// tei-studio-plugin-inventory reads one synthetic observation and prints the
// read-only inventory and diagnosis. It never starts a plugin and never reads
// raw data.
func main() {
	input := flag.String("input", "-", "path to a JSON observation, or - for stdin")
	mode := flag.String("mode", "diagnose", "diagnose (rows and report) or observe (which plugins may be invoked)")
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
	var in inventory.Input
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		fmt.Fprintln(os.Stderr, "decode input:", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	switch *mode {
	case "diagnose":
		report, err := inventory.Diagnose(in, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, "encode output:", err)
			os.Exit(1)
		}
	case "observe":
		observation, err := inventory.Observe(in, func(string) error { return nil })
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := enc.Encode(observation); err != nil {
			fmt.Fprintln(os.Stderr, "encode output:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q\n", *mode)
		os.Exit(1)
	}
}
