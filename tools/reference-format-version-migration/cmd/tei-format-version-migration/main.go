package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	formatversion "tei/tools/reference-format-version-migration"
)

// The CLI migrates definitions between format versions through a single conversion table
// and prints a report. It never writes a Source.
func main() {
	input := flag.String("input", "-", "path to a JSON request, or - for stdin")
	flag.Parse()

	var in formatversion.Input
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
	if err := enc.Encode(formatversion.Plan(in)); err != nil {
		fmt.Fprintln(os.Stderr, "encode output:", err)
		os.Exit(1)
	}
}
