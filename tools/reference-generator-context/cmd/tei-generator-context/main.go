package main

import (
	"flag"
	"fmt"
	"os"

	generatorcontext "tei/tools/reference-generator-context"
)

// tei-generator-context builds a bounded context for an external generator
// model. It does not call the model or accept its output.
func main() {
	workspaceDir := flag.String("workspace", "reference/workspace", "accepted Workspace root")
	contractPath := flag.String("contract", "", "generator contract JSON")
	fixturePath := flag.String("fixtures", "", "fixed fixture JSON")
	maxBytes := flag.Int("max-bytes", 32768, "maximum context size")
	out := flag.String("out", "", "output file; stdout when empty")
	flag.Parse()
	if *contractPath == "" || *fixturePath == "" {
		fail(fmt.Errorf("-contract and -fixtures are required"))
	}
	_, raw, err := generatorcontext.Build(*workspaceDir, *contractPath, *fixturePath, *maxBytes)
	if err != nil {
		fail(err)
	}
	if *out == "" {
		if _, err := os.Stdout.Write(append(raw, '\n')); err != nil {
			fail(err)
		}
		return
	}
	if err := os.WriteFile(*out, append(raw, '\n'), 0o644); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
