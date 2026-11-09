package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	generatorrevocationreview "tei/tools/reference-generator-revocation-review"
)

// tei-generator-revocation-review checks that revoked registry entries have a
// separate approval record. It does not verify approver authority or trust.
func main() {
	registryPath := flag.String("registry", "", "generator registry JSON")
	contractPath := flag.String("contract", "", "generator contract JSON")
	approvalsPath := flag.String("approvals", "", "revocation approvals JSON")
	flag.Parse()
	if *registryPath == "" || *contractPath == "" || *approvalsPath == "" {
		fail(fmt.Errorf("-registry, -contract, and -approvals are required"))
	}
	receipt, err := generatorrevocationreview.Review(*registryPath, *contractPath, *approvalsPath)
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
