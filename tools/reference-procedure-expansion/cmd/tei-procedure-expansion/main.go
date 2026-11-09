package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	procedureexpand "tei/tools/reference-procedure-expansion"
)

type output struct {
	Plan       procedureexpand.Report      `json:"plan"`
	Comparison *procedureexpand.Comparison `json:"comparison,omitempty"`
}

// The CLI expands an accepted procedure into realization options and prints a report.
// It never writes a Source and never enables execution.
func main() {
	input := flag.String("input", "-", "path to a JSON request, or - for stdin")
	compare := flag.String("compare", "", "path to a second request to compare against the first")
	flag.Parse()

	in, err := readInput(*input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	out := output{Plan: procedureexpand.Plan(in)}
	if *compare != "" {
		after, err := readInput(*compare)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		cmp := procedureexpand.Compare(in, after)
		out.Comparison = &cmp
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, "encode output:", err)
		os.Exit(1)
	}
}

func readInput(path string) (procedureexpand.Input, error) {
	var reader io.Reader = os.Stdin
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			return procedureexpand.Input{}, fmt.Errorf("open input: %w", err)
		}
		defer f.Close()
		reader = f
	}
	var in procedureexpand.Input
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return procedureexpand.Input{}, fmt.Errorf("decode input: %w", err)
	}
	return in, nil
}
