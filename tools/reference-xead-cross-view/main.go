package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
	verifyProjectionFlag := flag.String("verify-projection", "", "verify saved output from -local-projection against exact source and contract bytes")
	oracleFlag := flag.String("expect-accounting", "", "compare the freshly derived public aggregate with a saved public accounting baseline")
	flag.Parse()

	if *inputFlag == "" || *contractFlag == "" {
		fmt.Fprintln(os.Stderr, "usage: reference-xead-cross-view -input <model.xead> -contract <transcription-contract.json> [-format xead] [-local-projection] [-verify-projection <saved-local-projection.json>] [-expect-accounting <public-baseline.json>]")
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
	var output commandOutput
	if *verifyProjectionFlag != "" {
		reported, readErr := os.ReadFile(*verifyProjectionFlag)
		if readErr != nil {
			fmt.Fprintln(os.Stderr, readErr)
			os.Exit(2)
		}
		output, err = buildVerifiedCommandOutput(xeadsource.Format(*formatFlag), source, contract, reported, *localProjectionFlag)
	} else {
		output, err = buildCommandOutput(xeadsource.Format(*formatFlag), source, contract, *localProjectionFlag)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if *oracleFlag != "" {
		baselineBytes, readErr := os.ReadFile(*oracleFlag)
		if readErr != nil {
			fmt.Fprintln(os.Stderr, readErr)
			os.Exit(2)
		}
		if err := verifyPublicAccountingBaseline(output.Accounting, baselineBytes); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
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
	accounting, err := xeadsource.CrossViewPublicAccounting(projection)
	if err != nil {
		return commandOutput{}, err
	}
	return outputFromProjection(accounting, projection, includeLocalProjection), nil
}

// buildVerifiedCommandOutput does not trust a previously saved projection just
// because its own fingerprint matches. The stored object must agree with a
// fresh derivation from the exact supplied source and transcription contract.
// The ordinary response remains aggregate-only unless explicitly requested.
func buildVerifiedCommandOutput(
	format xeadsource.Format,
	source []byte,
	contract []byte,
	reportedJSON []byte,
	includeLocalProjection bool,
) (commandOutput, error) {
	decoder := json.NewDecoder(bytes.NewReader(reportedJSON))
	decoder.DisallowUnknownFields()
	var saved commandOutput
	if err := decoder.Decode(&saved); err != nil {
		return commandOutput{}, fmt.Errorf("decode saved cross-view projection: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return commandOutput{}, fmt.Errorf("saved cross-view projection has a trailing JSON value")
		}
		return commandOutput{}, fmt.Errorf("saved cross-view projection has invalid trailing data: %w", err)
	}
	if saved.LocalProjection == nil {
		return commandOutput{}, fmt.Errorf("saved output has no localProjection; regenerate with -local-projection")
	}
	accounting, err := xeadsource.CrossViewSourceReconciledAccounting(format, source, contract, *saved.LocalProjection)
	if err != nil {
		return commandOutput{}, fmt.Errorf("verify saved cross-view projection: %w", err)
	}
	// Never trust accounting retained in the saved command output. Recompute
	// from the exact verified source and publish only the fresh aggregate.
	return outputFromProjection(accounting, *saved.LocalProjection, includeLocalProjection), nil
}

func outputFromProjection(
	accounting xeadsource.CrossViewAccounting,
	projection xeadsource.CrossViewProjection,
	includeLocalProjection bool,
) commandOutput {
	output := commandOutput{Accounting: accounting}
	if includeLocalProjection {
		output.LocalOutputNotice = "local/private evidence: contains individual source tokens and cross-view relations; do not commit as public Reference Model evidence"
		output.LocalProjection = &projection
	}
	return output
}
