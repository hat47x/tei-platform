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

// compareLiveEntries observes two real Go entry points from the same clean
// checkout and target profile. Source drift between observations is rejected
// by the existing sourceRevision and build-environment comparison.
func compareLiveEntries(beforeEntry, afterEntry, tags string) (ClosureComparison, error) {
	before, err := produceEvidence(beforeEntry, tags)
	if err != nil {
		return ClosureComparison{}, fmt.Errorf("observe before entry: %w", err)
	}
	after, err := produceEvidence(afterEntry, tags)
	if err != nil {
		return ClosureComparison{}, fmt.Errorf("observe after entry: %w", err)
	}
	return compareReports(before, after)
}

func main() {
	entry := flag.String("entry", "./tools/reference-first-web-target", "literal Go entry package")
	tags := flag.String("tags", "", "optional Go build tags")
	compareEntry := flag.String("compare-entry", "", "second literal Go entry to observe and compare against -entry in this checkout")
	beforePath := flag.String("compare-before", "", "existing E0 JSON report for source-closure comparison")
	afterPath := flag.String("compare-after", "", "existing E0 JSON report for source-closure comparison")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	if *compareEntry != "" && (*beforePath != "" || *afterPath != "") {
		fmt.Fprintln(os.Stderr, "-compare-entry cannot be combined with -compare-before/-compare-after")
		os.Exit(2)
	}

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
	} else if *compareEntry != "" {
		comparison, err := compareLiveEntries(*entry, *compareEntry, *tags)
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
