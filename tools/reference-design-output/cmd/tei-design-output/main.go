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
	flag.Parse()
	if *root == "" || *out == "" {
		fail(fmt.Errorf("-workspace and -out are required"))
	}
	manifest, err := designoutput.Write(*root, *out, environment.DatabaseProvider(*provider))
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
