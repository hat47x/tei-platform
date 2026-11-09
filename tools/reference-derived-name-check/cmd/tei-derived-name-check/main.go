package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	derivednamecheck "tei/tools/reference-derived-name-check"
)

// tei-derived-name-check compares element names and the names derived from
// them under declared rules. It never changes an identifier or a name.
func main() {
	rulesPath := flag.String("rules", "", "derived-name rules JSON")
	inputPath := flag.String("input", "", "elements JSON")
	targets := flag.String("target", "", "comma-separated realization targets; empty leaves the check unevaluated")
	flag.Parse()
	if *rulesPath == "" || *inputPath == "" {
		fail(fmt.Errorf("-rules and -input are required"))
	}
	rules, err := derivednamecheck.LoadRules(*rulesPath)
	if err != nil {
		fail(err)
	}
	input, err := derivednamecheck.LoadInput(*inputPath)
	if err != nil {
		fail(err)
	}
	var selected []string
	if *targets != "" {
		selected = strings.Split(*targets, ",")
	}
	report, err := derivednamecheck.Check(rules, input, selected)
	if err != nil {
		fail(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fail(err)
	}
	if report.Status != derivednamecheck.StatusPassed {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
