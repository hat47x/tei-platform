// Command reference-purpose-preflight demonstrates read-only business guard
// preflight with synthetic fixtures or source inspections. It cannot grant execution permission.
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

func runCLI(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("reference-purpose-preflight", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	input := flags.String("input", "", "synthetic JSON fixture file (read-only)")
	root := flags.String("workspace", "", "real TEI Workspace (read-only source inspection)")
	kind := flags.String("subject-kind", "", "typed subject kind for Workspace inspection")
	id := flags.String("subject-id", "", "typed subject ID for Workspace inspection")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("positional arguments are not supported")
	}
	if *input != "" && *root == "" && *kind == "" && *id == "" {
		file, err := os.Open(*input)
		if err != nil {
			return err
		}
		defer file.Close()
		return run(file, output)
	}
	if *input == "" && *root != "" && *kind != "" && *id != "" {
		assessment, err := InspectWorkspace(*root, *kind, *id)
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		return encoder.Encode(assessment)
	}
	return fmt.Errorf("usage: reference-purpose-preflight -input <fixture.json> OR -workspace <path> -subject-kind application-pattern -subject-id <id>")
}

func main() {
	if err := runCLI(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
