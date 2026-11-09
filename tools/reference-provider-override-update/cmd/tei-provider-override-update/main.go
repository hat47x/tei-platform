package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	provideroverrideupdate "tei/tools/reference-provider-override-update"
)

// tei-provider-override-update compares a provider definition update with the
// case-side overrides and additions. It never writes or applies a definition.
func main() {
	inputPath := flag.String("input", "", "provider override update input JSON")
	flag.Parse()
	if *inputPath == "" {
		fail(fmt.Errorf("-input is required"))
	}
	input, err := provideroverrideupdate.LoadInput(*inputPath)
	if err != nil {
		fail(err)
	}
	result := provideroverrideupdate.Plan(input)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fail(err)
	}
	if result.Report.Blocked {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
