package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	choice "tei/plugins/reference-cognition-choice"
	cognitionport "tei/tools/reference-cognition-port"
)

// tei-cognition-propose runs one bounded cognition provider and prints a
// proposal receipt. It never accepts or applies the proposal.
func main() {
	requestPath := flag.String("request", "", "cognition request JSON")
	providerPath := flag.String("provider", "", "cognition provider JSON")
	command := flag.String("command", "", "trusted command for an external provider; write JSON to $TEI_COGNITION_OUTPUT")
	localChoice := flag.Bool("local-choice", false, "bind the reference local choice plugin (requires matching provider settings)")
	flag.Parse()
	if *requestPath == "" || *providerPath == "" {
		fail(fmt.Errorf("-request and -provider are required"))
	}
	var invoker cognitionport.ExternalInvoker
	if *localChoice {
		invoker = func(ctx context.Context, provider cognitionport.Provider, raw []byte) ([]byte, error) {
			if provider.Settings["transport"] != choice.Transport || provider.Network != cognitionport.NetworkNone {
				return nil, fmt.Errorf("local choice requires loopback-chat transport and no external network")
			}
			endpoint, _ := provider.Settings["endpoint"].(string)
			return choice.Invoke(ctx, endpoint, provider.Model, provider.Settings, raw)
		}
	}
	receipt, err := cognitionport.ProposeWithInvoker(context.Background(), *requestPath, *providerPath, *command, invoker)
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
