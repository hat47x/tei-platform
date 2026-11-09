package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	disable "tei/tools/reference-studio-plugin-disable"
)

type fileInput struct {
	State   disable.State   `json:"state"`
	Request disable.Request `json:"request"`
}

// tei-studio-plugin-disable plans or applies one enable/disable/remove
// operation. It never writes a source and never starts a plugin by itself.
func main() {
	input := flag.String("input", "-", "path to a JSON state and request, or - for stdin")
	mode := flag.String("mode", "plan", "plan (proposal only) or apply (rechecked settings update)")
	confirm := flag.Bool("confirm", false, "accept the expanded dependent candidate set")
	failExtension := flag.Bool("fail-extension", false, "simulate an extension start failure")
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
	var in fileInput
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		fmt.Fprintln(os.Stderr, "decode input:", err)
		os.Exit(1)
	}

	plan, err := disable.PlanAction(in.State, in.Request)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	switch *mode {
	case "plan":
		if err := enc.Encode(plan); err != nil {
			fmt.Fprintln(os.Stderr, "encode output:", err)
			os.Exit(1)
		}
	case "apply":
		starter := func([]string) error { return nil }
		if *failExtension {
			starter = func([]string) error { return fmt.Errorf("extension start failed (synthetic)") }
		}
		result, err := disable.Apply(in.State, plan, disable.ApplyRequest{Confirm: *confirm}, starter)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := enc.Encode(result); err != nil {
			fmt.Fprintln(os.Stderr, "encode output:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q\n", *mode)
		os.Exit(1)
	}
}
