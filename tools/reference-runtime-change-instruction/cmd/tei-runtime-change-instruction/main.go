package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	runtimechange "tei/tools/reference-runtime-change-instruction"
)

type output struct {
	Plan   runtimechange.Report  `json:"plan"`
	Accept *runtimechange.Report `json:"accept,omitempty"`
}

// The CLI classifies runtime screen change instructions and prints a report. It cannot
// write a Source, and it applies a change only for the IDs passed to -accept.
func main() {
	input := flag.String("input", "-", "path to a JSON request, or - for stdin")
	accept := flag.String("accept", "", "comma-separated instruction IDs a human accepted")
	flag.Parse()

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

	var in runtimechange.Input
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		fmt.Fprintln(os.Stderr, "decode input:", err)
		os.Exit(1)
	}

	out := output{Plan: runtimechange.Plan(in)}
	if strings.TrimSpace(*accept) != "" {
		ids := []string{}
		for _, p := range strings.Split(*accept, ",") {
			if s := strings.TrimSpace(p); s != "" {
				ids = append(ids, s)
			}
		}
		in.Accepted = ids
		r := runtimechange.Accept(in)
		out.Accept = &r
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, "encode output:", err)
		os.Exit(1)
	}
}
