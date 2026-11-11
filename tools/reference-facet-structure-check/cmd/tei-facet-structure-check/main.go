package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	facetstructurecheck "tei/tools/reference-facet-structure-check"
)

// tei-facet-structure-check judges, for each standard data structure, whether
// the orthogonal Facet description carries it. It writes no Source and adopts
// no Core type.
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
		if err := encoder.Encode(facetstructurecheck.ContractDescription()); err != nil {
			fail(err)
		}
		return
	}
	if *inputPath == "" {
		fail(fmt.Errorf("-input is required"))
	}
	cases, err := facetstructurecheck.LoadCases(*inputPath)
	if err != nil {
		fail(err)
	}
	report := facetstructurecheck.Check(cases, facetstructurecheck.Options{})
	if *expectedPath != "" {
		expected, err := facetstructurecheck.LoadExpected(*expectedPath)
		if err != nil {
			fail(err)
		}
		cmp := facetstructurecheck.Compare(report, expected)
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
	fmt.Fprintf(os.Stderr, "tei-facet-structure-check: %v\n", err)
	os.Exit(2)
}
