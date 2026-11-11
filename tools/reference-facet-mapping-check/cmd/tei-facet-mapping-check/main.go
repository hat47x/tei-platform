package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	facetmappingcheck "tei/tools/reference-facet-mapping-check"
)

// tei-facet-mapping-check compares a typed mapping contract with the facets
// observed on the data and representation sides. It writes no Source, renders
// nothing, and never grants an edit inverse.
func main() {
	inputPath := flag.String("input", "", "cases JSON")
	expectedPath := flag.String("expected", "", "hand-written independent expected JSON")
	pretty := flag.Bool("pretty", true, "indent the JSON report")
	flag.Parse()
	if *inputPath == "" {
		fail(fmt.Errorf("-input is required"))
	}
	cases, err := facetmappingcheck.LoadCases(*inputPath)
	if err != nil {
		fail(err)
	}
	report := facetmappingcheck.Check(cases, facetmappingcheck.Options{})
	report.Comparison = nil
	if *expectedPath != "" {
		expected, err := facetmappingcheck.LoadExpected(*expectedPath)
		if err != nil {
			fail(err)
		}
		cmp := facetmappingcheck.Compare(report, expected)
		report.Comparison = &cmp
	}

	encoder := json.NewEncoder(os.Stdout)
	if *pretty {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(report); err != nil {
		fail(err)
	}
	if report.Comparison != nil && report.Comparison.Status != "ok" {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "tei-facet-mapping-check: %v\n", err)
	os.Exit(2)
}
