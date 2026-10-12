// Command reference-purpose-preflight demonstrates read-only business guard
// preflight with synthetic fixtures. It cannot grant execution permission.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

type Fixture struct {
	Cases []Case `json:"cases"`
}

type Report struct {
	SchemaVersion string   `json:"schemaVersion"`
	Authority     string   `json:"authority"`
	Results       []Result `json:"results"`
}

func run(reader io.Reader, writer io.Writer) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var fixture Fixture
	if err := decoder.Decode(&fixture); err != nil {
		return fmt.Errorf("decode fixture: %w", err)
	}
	if len(fixture.Cases) == 0 {
		return fmt.Errorf("fixture must contain at least one case")
	}
	var remaining any
	if err := decoder.Decode(&remaining); err != io.EOF {
		return fmt.Errorf("fixture must contain exactly one JSON object")
	}
	report := Report{SchemaVersion: "tei.reference.purpose-preflight/v0", Authority: "fixture-only-no-execution", Results: make([]Result, 0, len(fixture.Cases))}
	for _, c := range fixture.Cases {
		report.Results = append(report.Results, Evaluate(c))
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func main() {
	input := flag.String("input", "", "synthetic JSON fixture file (read-only)")
	flag.Parse()
	if *input == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: reference-purpose-preflight -input <fixture.json>")
		os.Exit(2)
	}
	file, err := os.Open(*input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer file.Close()
	if err := run(file, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
