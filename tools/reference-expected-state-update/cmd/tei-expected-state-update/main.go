package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	expectedstate "tei/tools/reference-expected-state-update"
)

type fileInput struct {
	State    expectedstate.State           `json:"state"`
	Requests []expectedstate.UpdateRequest `json:"requests"`
}

// tei-expected-state-update applies synthetic updates in order and prints the
// business outcomes, the resulting state and the observation. It never writes.
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
	var in fileInput
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		fmt.Fprintln(os.Stderr, "decode input:", err)
		os.Exit(1)
	}
	run, err := expectedstate.Run(in.State, in.Requests)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(run); err != nil {
		fmt.Fprintln(os.Stderr, "encode output:", err)
		os.Exit(1)
	}
}
