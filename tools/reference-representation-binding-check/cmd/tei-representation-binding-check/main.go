package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	representationbindingcheck "tei/tools/reference-representation-binding-check"
)

// tei-representation-binding-check checks that a meaning-bearing representation
// instance is owned by one Function Data Context and carries the binding
// columns the mapping depends on. It renders nothing.
func main() {
	inputPath := flag.String("input", "", "cases JSON")
	expectedPath := flag.String("expected", "", "hand-written independent expected JSON")
	contractOnly := flag.Bool("contract", false, "print the contract self-description")
	pretty := flag.Bool("pretty", true, "indent the JSON report")
	flag.Parse()

	encoder := json.NewEncoder(os.Stdout)
	if *pretty {
		encoder.SetIndent("", "  ")
	}
	if *contractOnly {
		if err := encoder.Encode(representationbindingcheck.ContractDescription()); err != nil {
			fail(err)
		}
		return
	}
	if *inputPath == "" {
		fail(fmt.Errorf("-input is required"))
	}
	cases, err := representationbindingcheck.LoadCases(*inputPath)
	if err != nil {
		fail(err)
	}
	report := representationbindingcheck.Check(cases, representationbindingcheck.Options{})
	if *expectedPath != "" {
		expected, err := representationbindingcheck.LoadExpected(*expectedPath)
		if err != nil {
			fail(err)
		}
		cmp := representationbindingcheck.Compare(report, expected)
		report.Comparison = &cmp
	}
	if err := encoder.Encode(report); err != nil {
		fail(err)
	}
	if report.Comparison != nil && report.Comparison.Status != "ok" {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "tei-representation-binding-check: %v\n", err)
	os.Exit(2)
}
