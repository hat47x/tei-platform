package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	lifecycle "tei/tools/reference-studio-plugin-data-lifecycle"
)

type deleteInput struct {
	State   lifecycle.DataState     `json:"state"`
	Request lifecycle.DeleteRequest `json:"request"`
}

type convertInput struct {
	State   lifecycle.DataState      `json:"state"`
	Request lifecycle.ConvertRequest `json:"request"`
}

// tei-studio-data-lifecycle saves a raw source, plans an explicit data
// deletion, or plans a format conversion. It never writes a file.
func main() {
	input := flag.String("input", "-", "path to a JSON request, or - for stdin")
	mode := flag.String("mode", "save", "save, delete, or convert")
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
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	switch *mode {
	case "save":
		var req lifecycle.EditRequest
		if err := decode(reader, &req); err != nil {
			fail(err)
		}
		result, err := lifecycle.Save(req)
		if err != nil {
			fail(err)
		}
		if err := enc.Encode(result); err != nil {
			fail(err)
		}
	case "delete":
		var in deleteInput
		if err := decode(reader, &in); err != nil {
			fail(err)
		}
		plan, err := lifecycle.PlanDeletion(in.State, in.Request)
		if err != nil {
			fail(err)
		}
		if err := enc.Encode(plan); err != nil {
			fail(err)
		}
	case "convert":
		var in convertInput
		if err := decode(reader, &in); err != nil {
			fail(err)
		}
		plan, err := lifecycle.PlanConversion(in.State, in.Request)
		if err != nil {
			fail(err)
		}
		if err := enc.Encode(plan); err != nil {
			fail(err)
		}
	default:
		fail(fmt.Errorf("unknown mode %q", *mode))
	}
}

func decode(reader io.Reader, value any) error {
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	return dec.Decode(value)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
