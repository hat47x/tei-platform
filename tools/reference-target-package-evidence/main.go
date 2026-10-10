package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

func readEvidenceFile(file string) (Report, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return Report{}, fmt.Errorf("read package report: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var report Report
	if err := decoder.Decode(&report); err != nil {
		return Report{}, fmt.Errorf("decode package report: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Report{}, fmt.Errorf("package report has trailing JSON or invalid content")
	}
	return report, nil
}

func main() {
	entry := flag.String("entry", "./tools/reference-first-web-target", "literal Go entry package")
	tags := flag.String("tags", "", "optional Go build tags")
	beforePath := flag.String("compare-before", "", "existing E0 JSON report for source-closure comparison")
	afterPath := flag.String("compare-after", "", "existing E0 JSON report for source-closure comparison")
	flag.Parse()

	var output any
	if *beforePath != "" || *afterPath != "" {
		if *beforePath == "" || *afterPath == "" {
			fmt.Fprintln(os.Stderr, "-compare-before and -compare-after must both be provided")
			os.Exit(2)
		}
		before, err := readEvidenceFile(*beforePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		after, err := readEvidenceFile(*afterPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		comparison, err := compareReports(before, after)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		output = comparison
	} else {
		report, err := produceEvidence(*entry, *tags)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		output = report
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
