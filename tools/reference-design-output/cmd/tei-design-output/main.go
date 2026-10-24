package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/environment"
	designoutput "tei/tools/reference-design-output"
)

// tei-design-output writes one bundle of derived design artifacts into a new
// directory. It never edits the Workspace, never overwrites an existing file,
// and never grants acceptance or execution authority.
func main() {
	root := flag.String("workspace", "", "accepted Workspace root to read")
	out := flag.String("out", "", "new output directory; must not exist")
	provider := flag.String("provider", string(environment.ProviderPostgres), "database provider for DDL rendering")
	contract := flag.String("contract", "", "optional reference generator contract JSON")
	strategy := flag.String("strategy", "", "fixed generator strategy; requires -contract")
	version := flag.String("generator-version", "1", "declared generator version")
	flag.Parse()
	if *root == "" || *out == "" {
		fail(fmt.Errorf("-workspace and -out are required"))
	}
	var manifest designoutput.Manifest
	var err error
	if *contract == "" {
		if *strategy != "" || *version != "1" {
			fail(fmt.Errorf("-strategy and -generator-version require -contract"))
		}
		manifest, err = designoutput.Write(*root, *out, environment.DatabaseProvider(*provider))
	} else {
		contractJSON, readErr := os.ReadFile(*contract)
		if readErr != nil {
			fail(readErr)
		}
		manifest, err = designoutput.WriteWithGenerator(*root, *out, environment.DatabaseProvider(*provider), contractJSON, *strategy, *version)
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
