package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	semanticdomaincoverage "tei/tools/reference-semantic-domain-coverage"
)

// tei-semantic-domain-coverage reports untyped and ambiguously typed elements
// and whether each element separates What from Why. It never writes back.
func main() {
	inputPath := flag.String("input", "", "semantic domain coverage input JSON")
	flag.Parse()
	if *inputPath == "" {
		fail(fmt.Errorf("-input is required"))
	}
	input, err := semanticdomaincoverage.LoadInput(*inputPath)
	if err != nil {
		fail(err)
	}
	report := semanticdomaincoverage.Check(input)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fail(err)
	}
	if report.Status == semanticdomaincoverage.StatusRejected {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
