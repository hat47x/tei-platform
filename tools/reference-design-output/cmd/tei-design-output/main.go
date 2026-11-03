package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"tei/internal/environment"
	designoutput "tei/tools/reference-design-output"
)

// tei-design-output publishes or checks derived design bundles. It never
// edits the Workspace and never grants acceptance or execution authority.
func main() {
	root := flag.String("workspace", "", "accepted Workspace root to read")
	out := flag.String("out", "", "new output directory; must not exist")
	provider := flag.String("provider", string(environment.ProviderPostgres), "database provider for DDL rendering")
	contract := flag.String("contract", "", "optional reference generator contract JSON")
	strategy := flag.String("strategy", "", "fixed generator strategy; requires -contract")
	version := flag.String("generator-version", "1", "declared generator version")
	verify := flag.String("verify", "", "existing bundle to compare with current inputs; read-only, excludes -out")
	format := flag.String("format", "markdown", "document format plugin: markdown or text")
	effectsPath := flag.String("effects", "", "optional JSON file with an observed-effect report")
	flag.Parse()

	options := designoutput.Options{Format: *format}
	if *effectsPath != "" {
		raw, err := os.ReadFile(*effectsPath)
		if err != nil {
			fail(err)
		}
		if err := json.Unmarshal(raw, &options); err != nil {
			fail(fmt.Errorf("decode effects report: %w", err))
		}
		if strings.TrimSpace(options.Format) == "" {
			options.Format = *format
		}
	}

	destination := *out
	verifying := *verify != ""
	if verifying {
		if *out != "" {
			fail(fmt.Errorf("-verify and -out are mutually exclusive"))
		}
		destination = *verify
	}
	if *root == "" || destination == "" {
		fail(fmt.Errorf("-workspace and either -out or -verify are required"))
	}

	var manifest designoutput.Manifest
	var err error
	switch {
	case *contract == "" && !verifying:
		if *strategy != "" || *version != "1" {
			fail(fmt.Errorf("-strategy and -generator-version require -contract"))
		}
		manifest, err = designoutput.WriteWithOptions(*root, destination, environment.DatabaseProvider(*provider), options)
	case *contract == "" && verifying:
		if *strategy != "" || *version != "1" {
			fail(fmt.Errorf("-strategy and -generator-version require -contract"))
		}
		manifest, err = designoutput.VerifyWithOptions(*root, destination, environment.DatabaseProvider(*provider), options)
	case !verifying:
		contractJSON, readErr := os.ReadFile(*contract)
		if readErr != nil {
			fail(readErr)
		}
		manifest, err = designoutput.WriteWithGeneratorWithOptions(*root, destination, environment.DatabaseProvider(*provider), contractJSON, *strategy, *version, options)
	default:
		contractJSON, readErr := os.ReadFile(*contract)
		if readErr != nil {
			fail(readErr)
		}
		manifest, err = designoutput.VerifyWithGeneratorWithOptions(*root, destination, environment.DatabaseProvider(*provider), contractJSON, *strategy, *version, options)
	}
	if err != nil {
		fail(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(manifest); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
