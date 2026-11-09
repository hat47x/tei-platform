package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	cognitionport "tei/tools/reference-cognition-port"
)

// tei-cognition-propose runs one bounded cognition provider and prints a
// proposal receipt. It never accepts or applies the proposal.
func main() {
	requestPath := flag.String("request", "", "cognition request JSON")
	providerPath := flag.String("provider", "", "cognition provider JSON")
	command := flag.String("command", "", "trusted command for an external provider; write JSON to $TEI_COGNITION_OUTPUT")
	flag.Parse()
	if *requestPath == "" || *providerPath == "" {
		fail(fmt.Errorf("-request and -provider are required"))
	}
	receipt, err := cognitionport.Propose(context.Background(), *requestPath, *providerPath, *command)
	if err != nil {
		fail(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(receipt); err != nil {
		fail(err)
	}
	if receipt.State != cognitionport.StateProposed {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
