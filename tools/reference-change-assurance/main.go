package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

func run(reader io.Reader, writer io.Writer) error {
	d := json.NewDecoder(reader)
	d.DisallowUnknownFields()
	var fixture Fixture
	if err := d.Decode(&fixture); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	var rest any
	if err := d.Decode(&rest); err != io.EOF {
		return fmt.Errorf("fixture must be exactly one JSON value")
	}
	report, err := evaluate(fixture)
	if err != nil {
		return err
	}
	e := json.NewEncoder(writer)
	e.SetIndent("", "  ")
	return e.Encode(report)
}

func main() {
	input := flag.String("input", "", "path to synthetic, read-only fixture")
	flag.Parse()
	if *input == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: reference-change-assurance -input <fixture.json>")
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
