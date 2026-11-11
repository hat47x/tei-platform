package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	facetprojectionmatrix "tei/tools/reference-facet-projection-matrix"
)

// tei-facet-projection-matrix answers every (structure, representation) pair and
// orders the answers. It does not merge them into one "possible" answer.
func main() {
	inputPath := flag.String("input", "", "matrix input JSON")
	expectedPath := flag.String("expected", "", "hand-written independent expected JSON")
	contractOnly := flag.Bool("contract", false, "print the contract self-description")
	cellList := flag.Bool("cells", false, "print one line per answered cell")
	pretty := flag.Bool("pretty", true, "indent the JSON report")
	flag.Parse()

	encoder := json.NewEncoder(os.Stdout)
	if *pretty {
		encoder.SetIndent("", "  ")
	}
	if *contractOnly {
		if err := encoder.Encode(facetprojectionmatrix.ContractDescription()); err != nil {
			fail(err)
		}
		return
	}
	if *inputPath == "" {
		fail(fmt.Errorf("-input is required"))
	}
	cases, err := facetprojectionmatrix.LoadCases(*inputPath)
	if err != nil {
		fail(err)
	}
	report := facetprojectionmatrix.Check(cases)
	if *expectedPath != "" {
		expected, err := facetprojectionmatrix.LoadExpected(*expectedPath)
		if err != nil {
			fail(err)
		}
		cmp := facetprojectionmatrix.Compare(report, expected)
		report.Comparison = &cmp
	}
	if *cellList {
		for _, row := range report.Matrix.Structures {
			for _, cell := range row.Cells {
				fmt.Fprintf(os.Stdout, "%s -> %s: %s\n", cell.Structure, cell.Representation, cell.Verdict)
			}
		}
		return
	}
	if err := encoder.Encode(report); err != nil {
		fail(err)
	}
	if report.Comparison != nil && report.Comparison.Status != "ok" {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "tei-facet-projection-matrix: %v\n", err)
	os.Exit(2)
}
