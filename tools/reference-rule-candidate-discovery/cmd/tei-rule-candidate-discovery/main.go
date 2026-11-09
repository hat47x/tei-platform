package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	rulecandidate "tei/tools/reference-rule-candidate-discovery"
)

type executeResult struct {
	Input  string             `json:"input"`
	Kind   rulecandidate.Kind `json:"kind"`
	Output string             `json:"output"`
	OK     bool               `json:"ok"`
}

type output struct {
	Report          rulecandidate.Report     `json:"report"`
	Confirm         *rulecandidate.Confirmed `json:"confirm,omitempty"`
	ConfirmFindings []rulecandidate.Finding  `json:"confirmFindings,omitempty"`
	Execute         *executeResult           `json:"execute,omitempty"`
}

// The CLI proposes rule hypotheses from examples and prints a report. It never writes
// a Source, accepts a rule, or connects a hypothesis to execution on its own.
func main() {
	input := flag.String("input", "-", "path to a JSON request with examples, or - for stdin")
	selectKind := flag.String("select", "", "hypothesis kind a human selected (optional)")
	claim := flag.String("claim", "format", "meaning claim for the selection: format, business-identity, authorization")
	probe := flag.String("probe", "", "input to run the selected hypothesis on (requires -select)")
	flag.Parse()

	if *probe != "" && *selectKind == "" {
		fmt.Fprintln(os.Stderr, "-probe requires -select")
		os.Exit(2)
	}

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

	var req struct {
		Examples []rulecandidate.Example `json:"examples"`
	}
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		fmt.Fprintln(os.Stderr, "decode input:", err)
		os.Exit(1)
	}
	if len(req.Examples) == 0 {
		fmt.Fprintln(os.Stderr, "input needs at least one example")
		os.Exit(1)
	}

	out := output{Report: rulecandidate.Analyze(req.Examples)}
	if *selectKind != "" {
		ids := make([]string, 0, len(req.Examples))
		for _, e := range req.Examples {
			ids = append(ids, e.ID)
		}
		sel := rulecandidate.Selection{Kind: rulecandidate.Kind(*selectKind), ExampleIDs: ids}
		confirmed, findings := rulecandidate.Confirm(sel, req.Examples, *claim)
		out.ConfirmFindings = findings
		if len(findings) == 0 {
			out.Confirm = &confirmed
		}
		if *probe != "" {
			value, ok := rulecandidate.Execute(sel, req.Examples, *probe)
			out.Execute = &executeResult{Input: *probe, Kind: sel.Kind, Output: value, OK: ok}
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, "encode output:", err)
		os.Exit(1)
	}
}
