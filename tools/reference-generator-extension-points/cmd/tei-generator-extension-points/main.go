package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	generatorextensionpoints "tei/tools/reference-generator-extension-points"
)

// tei-generator-extension-points checks declared extension points, keeps the
// case extension outside the artifact, and reports direct edits before
// regeneration overwrites them. It never writes or regenerates anything.
func main() {
	inputPath := flag.String("input", "", "generator extension points input JSON")
	flag.Parse()
	if *inputPath == "" {
		fail(fmt.Errorf("-input is required"))
	}
	input, err := generatorextensionpoints.LoadInput(*inputPath)
	if err != nil {
		fail(err)
	}
	report := generatorextensionpoints.Check(input)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fail(err)
	}
	if report.Status != generatorextensionpoints.StatusPassed {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
