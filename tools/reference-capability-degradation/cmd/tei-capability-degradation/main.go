package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	capabilitydegradation "tei/tools/reference-capability-degradation"
)

// tei-capability-degradation compares required guarantees with observed
// capabilities and reports withheld or explicitly degraded operations. It does
// not generate code or change a database.
func main() {
	inputPath := flag.String("input", "", "capability degradation input JSON")
	flag.Parse()
	if *inputPath == "" {
		fail(fmt.Errorf("-input is required"))
	}
	input, err := capabilitydegradation.LoadInput(*inputPath)
	if err != nil {
		fail(err)
	}
	report := capabilitydegradation.Check(input)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fail(err)
	}
	if report.Status == capabilitydegradation.StatusBlocked || report.Status == capabilitydegradation.StatusUnresolved {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
