package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	generatorselection "tei/tools/reference-generator-selection"
)

// tei-generator-reconcile reports whether candidate directories still match the
// active generator versions in a registry. It does not build or execute them.
func main() {
	registryPath := flag.String("registry", "", "generator registry JSON")
	contractPath := flag.String("contract", "", "generator contract JSON")
	candidatesDir := flag.String("candidates", "", "directory containing candidate directories")
	flag.Parse()
	if *registryPath == "" || *contractPath == "" || *candidatesDir == "" {
		fail(fmt.Errorf("-registry, -contract, and -candidates are required"))
	}
	receipt, err := generatorselection.Reconcile(*registryPath, *contractPath, *candidatesDir)
	if err != nil {
		fail(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(receipt); err != nil {
		fail(err)
	}
	if !receipt.Passed {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
