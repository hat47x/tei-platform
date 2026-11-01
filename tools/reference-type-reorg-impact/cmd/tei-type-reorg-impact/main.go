package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"tei/internal/workspace"
	typereorg "tei/tools/reference-type-reorg-impact"
)

// The CLI reports the impact of a type merge or split. It may export a
// merge candidate, but never rewrites source or accepts a meaning change.
func validateCandidateMode(split bool, candidateOut string) error {
	if split && candidateOut != "" {
		return fmt.Errorf("-candidate-out is not supported with -split: split reassignment is not yet represented in candidates")
	}
	return nil
}

func main() {
	root := flag.String("workspace", "", "source Workspace root to read")
	from := flag.String("from", "", "comma-separated type IDs to merge away (or single source type for -split)")
	into := flag.String("into", "", "type ID to merge into (or comma-separated target types for -split)")
	split := flag.Bool("split", false, "compute the impact of splitting a type into several target types")
	candOut := flag.String("candidate-out", "", "new file for a set-attribute-type change candidate (needs -base-revision)")
	base := flag.String("base-revision", "", "target model revision from the Shared Acceptance revision command")
	flag.Parse()
	if *root == "" || *from == "" || *into == "" {
		fmt.Fprintln(os.Stderr, "-workspace, -from and -into are required")
		os.Exit(1)
	}
	if err := validateCandidateMode(*split, *candOut); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	loaded, err := workspace.Load(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load Workspace:", err)
		os.Exit(1)
	}
	var impact typereorg.Impact
	if *split {
		impact, err = typereorg.SplitImpact(loaded.World, typereorg.Split{From: *from, Into: strings.Split(*into, ",")})
	} else {
		impact, err = typereorg.Analyze(loaded.World, typereorg.Merge{From: strings.Split(*from, ","), Into: *into})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *candOut != "" {
		raw, err := typereorg.Candidate(impact, *base)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		f, err := os.OpenFile(*candOut, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			fmt.Fprintln(os.Stderr, "create candidate file:", err)
			os.Exit(1)
		}
		if _, err := f.Write(append(raw, '\n')); err != nil {
			f.Close()
			os.Remove(*candOut)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := f.Close(); err != nil {
			os.Remove(*candOut)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(impact); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
