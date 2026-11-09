package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	semanticmappingcheck "tei/tools/reference-semantic-mapping-check"
)

// tei-semantic-mapping-check checks the mapping between meaning definitions
// and elements. It never creates, accepts, or suggests a definition.
func main() {
	inputPath := flag.String("input", "", "mapping input JSON")
	afterPath := flag.String("after", "", "input after adding definitions; compares the impact against -input")
	flag.Parse()
	if *inputPath == "" {
		fail(fmt.Errorf("-input is required"))
	}
	before, err := semanticmappingcheck.LoadInput(*inputPath)
	if err != nil {
		fail(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if *afterPath != "" {
		after, err := semanticmappingcheck.LoadInput(*afterPath)
		if err != nil {
			fail(err)
		}
		impact, err := semanticmappingcheck.CompareImpact(before, after)
		if err != nil {
			fail(err)
		}
		if err := encoder.Encode(impact); err != nil {
			fail(err)
		}
		if impact.Status != semanticmappingcheck.ImpactWithinScope {
			os.Exit(1)
		}
		return
	}
	report, err := semanticmappingcheck.Check(before)
	if err != nil {
		fail(err)
	}
	if err := encoder.Encode(report); err != nil {
		fail(err)
	}
	if report.Status != semanticmappingcheck.StatusComplete {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
