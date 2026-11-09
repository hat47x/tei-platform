package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	aiwritewindow "tei/tools/reference-ai-write-window"
)

// tei-ai-write-window checks an AI edit against a plugin-declared write window.
// It never writes, accepts, or deploys anything.
func main() {
	inputPath := flag.String("input", "", "AI write window input JSON")
	flag.Parse()
	if *inputPath == "" {
		fail(fmt.Errorf("-input is required"))
	}
	input, err := aiwritewindow.LoadInput(*inputPath)
	if err != nil {
		fail(err)
	}
	report := aiwritewindow.Check(input)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fail(err)
	}
	switch report.Status {
	case aiwritewindow.StatusCandidateAccepted:
		// 0: the edit is an acceptance candidate.
	case aiwritewindow.StatusHumanDecisionNeeded:
		os.Exit(2)
	default:
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
