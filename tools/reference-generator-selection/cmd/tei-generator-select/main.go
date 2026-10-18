package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	generatorselection "tei/tools/reference-generator-selection"
)

// tei-generator-select compares candidate generators for one reference
// contract and prints a receipt. It builds and runs candidates; it does not
// accept meaning, grant authority, or prove a real Provider.
func main() {
	workspaceDir := flag.String("workspace", "reference/workspace", "accepted Workspace root")
	contractPath := flag.String("contract", "", "generator contract JSON")
	fixturesPath := flag.String("fixtures", "", "fixed fixture JSON")
	candidatesDir := flag.String("candidates", "", "directory containing candidate directories")
	flag.Parse()
	if *contractPath == "" || *fixturesPath == "" || *candidatesDir == "" {
		fail(fmt.Errorf("-contract, -fixtures, and -candidates are required"))
	}
	receipt, err := generatorselection.Select(context.Background(), *workspaceDir, *contractPath, *fixturesPath, *candidatesDir)
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
