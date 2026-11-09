package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	hostdraftrecovery "tei/tools/reference-host-draft-recovery"
)

// The CLI decides one Host draft situation across an external edit, a restart
// and an interrupted side effect. It never reads a file path as a workspace,
// writes a Source, or resends a side effect.
func main() {
	input := flag.String("input", "-", "path to a JSON request, or - for stdin")
	flag.Parse()

	var in hostdraftrecovery.Input
	var reader io.Reader = os.Stdin
	if *input != "-" {
		f, err := os.Open(*input)
		if err != nil {
			fmt.Fprintln(os.Stderr, "open input:", err)
			os.Exit(1)
		}
		defer f.Close()
		reader = f
	}
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		fmt.Fprintln(os.Stderr, "decode input:", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(hostdraftrecovery.Decide(in)); err != nil {
		fmt.Fprintln(os.Stderr, "encode output:", err)
		os.Exit(1)
	}
}
