package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	generatorselection "tei/tools/reference-generator-selection"
)

// tei-generator-build renders candidate sources from one contract and fixed
// strategies, including a Python reference strategy. It writes only to a new
// directory and does not select, accept, or execute the generated code.
func main() {
	contractPath := flag.String("contract", "", "generator contract JSON")
	outDir := flag.String("out", "", "new output directory")
	all := flag.Bool("all", false, "generate all fixed strategies under -out at version 1")
	strategy := flag.String("strategy", "", "one of template, alternative, dangerous, python-template, javascript-template, java-template, wasm-template, long-running-template, python-long-running-template, javascript-long-running-template, java-long-running-template, wasm-long-running-template")
	version := flag.String("version", "1", "generator version recorded in the candidate manifest")
	flag.Parse()
	if *contractPath == "" || *outDir == "" {
		fail(fmt.Errorf("-contract and -out are required"))
	}
	if *all && *version != "1" {
		fail(fmt.Errorf("-version is only supported with one -strategy"))
	}
	var results []generatorselection.Generated
	var err error
	if *all {
		results, err = generatorselection.GenerateAll(*contractPath, *outDir)
	} else {
		if *strategy == "" {
			fail(fmt.Errorf("-strategy or -all is required"))
		}
		var generated generatorselection.Generated
		generated, err = generatorselection.GenerateVersion(*contractPath, *strategy, *version, *outDir)
		results = append(results, generated)
	}
	if err != nil {
		fail(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
