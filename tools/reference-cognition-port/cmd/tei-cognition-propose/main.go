package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"

	cognitionport "tei/tools/reference-cognition-port"
)

// tei-cognition-propose runs one bounded cognition provider and prints a
// proposal receipt. It never accepts or applies the proposal.
func main() {
	requestPath := flag.String("request", "", "cognition request JSON")
	providerPath := flag.String("provider", "", "cognition provider JSON")
	command := flag.String("command", "", "trusted command for an external provider; write JSON to $TEI_COGNITION_OUTPUT")
	plugin := flag.String("plugin", "", "statically composed provider: local-choice")
	flag.Parse()
	if *requestPath == "" || *providerPath == "" {
		fail(fmt.Errorf("-request and -provider are required"))
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	var receipt cognitionport.Receipt
	var err error
	switch *plugin {
	case "":
		receipt, err = cognitionport.Propose(ctx, *requestPath, *providerPath, *command)
	case "local-choice":
		if *command != "" {
			fail(fmt.Errorf("-plugin and -command cannot be combined"))
		}
		receipt, err = cognitionport.ProposeWithProvider(ctx, *requestPath, *providerPath, selectedChoice)
	default:
		fail(fmt.Errorf("unknown statically composed plugin"))
	}
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
