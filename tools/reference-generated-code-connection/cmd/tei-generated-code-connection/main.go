package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	generatedcode "tei/tools/reference-generated-code-connection"
)

type output struct {
	Plan       generatedcode.Report      `json:"plan"`
	Comparison *generatedcode.Comparison `json:"comparison,omitempty"`
}

// The CLI decides how generated-code artifacts connect and prints a report. It never
// runs the code and never accepts the code's execution meaning.
func main() {
	input := flag.String("input", "-", "path to a JSON request, or - for stdin")
	compare := flag.String("compare", "", "path to a second request to compare against the first")
	flag.Parse()

	in, err := readInput(*input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	out := output{Plan: generatedcode.Plan(in)}
	if *compare != "" {
		after, err := readInput(*compare)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		cmp := generatedcode.Compare(in, after)
		out.Comparison = &cmp
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, "encode output:", err)
		os.Exit(1)
	}
}

func readInput(path string) (generatedcode.Input, error) {
	var reader io.Reader = os.Stdin
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			return generatedcode.Input{}, fmt.Errorf("open input: %w", err)
		}
		defer f.Close()
		reader = f
	}
	var in generatedcode.Input
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return generatedcode.Input{}, fmt.Errorf("decode input: %w", err)
	}
	return in, nil
}
