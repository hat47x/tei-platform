package main

import (
	"flag"
	"fmt"
	"os"

	cognitionhandoff "tei/tools/reference-cognition-handoff"
)

// tei-cognition-handoff converts one cognition proposal into a Shared
// Acceptance change candidate. It does not preview, review, accept, or apply it.
func main() {
	proposalPath := flag.String("proposal", "", "cognition proposal JSON")
	workspaceDir := flag.String("workspace", "reference/workspace", "accepted Workspace root")
	providerID := flag.String("provider", "", "cognition provider id")
	model := flag.String("model", "", "cognition model")
	version := flag.String("version", "", "cognition model version")
	actor := flag.String("actor", "", "optional origin actor")
	candidateID := flag.String("candidate", "", "optional candidate id")
	out := flag.String("out", "", "output candidate JSON")
	flag.Parse()
	if *proposalPath == "" || *out == "" {
		fail(fmt.Errorf("-proposal and -out are required"))
	}
	candidate, err := cognitionhandoff.Handoff(*proposalPath, *workspaceDir, *providerID, *model, *version, *actor, *candidateID)
	if err != nil {
		fail(err)
	}
	if err := cognitionhandoff.Write(candidate, *out); err != nil {
		fail(err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", *out)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
