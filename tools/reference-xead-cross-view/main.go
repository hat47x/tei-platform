package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/xeadsource"
)

type commandOutput struct {
	Accounting        xeadsource.CrossViewAccounting  `json:"accounting"`
	LocalOutputNotice string                          `json:"localOutputNotice,omitempty"`
	LocalProjection   *xeadsource.CrossViewProjection `json:"localProjection,omitempty"`
}

func main() {
	formatFlag := flag.String("format", "xead", "source format; RMT-4 currently supports xead")
	inputFlag := flag.String("input", "", "path to Xead source XML")
	contractFlag := flag.String("contract", "", "path to transcription contract JSON")
	localProjectionFlag := flag.Bool("local-projection", false, "include token-level cross-view projection; local/private output only")
	flag.Parse()

	if *inputFlag == "" || *contractFlag == "" {
		fmt.Fprintln(os.Stderr, "usage: reference-xead-cross-view -input <model.xead> -contract <transcription-contract.json> [-format xead] [-local-projection]")
		os.Exit(2)
	}
	source, err := os.ReadFile(*inputFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	contract, err := os.ReadFile(*contractFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	output, err := buildCommandOutput(xeadsource.Format(*formatFlag), source, contract, *localProjectionFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func buildCommandOutput(
	format xeadsource.Format,
	source []byte,
	contract []byte,
	includeLocalProjection bool,
) (commandOutput, error) {
	projection, err := xeadsource.BuildCrossViewProjection(format, source, contract)
	if err != nil {
		return commandOutput{}, err
	}
	accounting := xeadsource.CrossViewPublicAccounting(projection)
	output := commandOutput{Accounting: accounting}
	if includeLocalProjection {
		output.LocalOutputNotice = "local/private evidence: contains individual source tokens and cross-view relations; do not commit as public Reference Model evidence"
		output.LocalProjection = &projection
	}
	return output, nil
}
