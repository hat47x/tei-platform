package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	prototyping "tei/tools/reference-exploratory-prototyping"
)

type effortRequest struct {
	Rigorous    prototyping.Effort `json:"rigorous"`
	Exploratory prototyping.Effort `json:"exploratory"`
}

type output struct {
	Plan       prototyping.Report            `json:"plan"`
	Comparison *prototyping.EffortComparison `json:"effortComparison,omitempty"`
}

// The CLI checks an exploratory prototyping workflow and prints a report. It cannot
// promote a hypothesis to an accepted specification.
func main() {
	input := flag.String("input", "-", "path to a JSON request, or - for stdin")
	effort := flag.String("effort", "", "path to a JSON {rigorous, exploratory} effort request")
	flag.Parse()

	var in prototyping.Input
	if err := decode(*input, &in); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	out := output{Plan: prototyping.Plan(in)}
	if *effort != "" {
		var req effortRequest
		if err := decode(*effort, &req); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		c := prototyping.CompareEffort(req.Rigorous, req.Exploratory)
		out.Comparison = &c
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, "encode output:", err)
		os.Exit(1)
	}
}

func decode(path string, v any) error {
	var reader io.Reader = os.Stdin
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open input: %w", err)
		}
		defer f.Close()
		reader = f
	}
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode input: %w", err)
	}
	return nil
}
