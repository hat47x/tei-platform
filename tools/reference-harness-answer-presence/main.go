// Command reference-harness-answer-presence reports which facts from captured
// TEI CLI output an external-harness answer mentions.
//
// It reads only the files the caller names and writes only to the given output
// path. It does not grade correctness: the Runbook's reviewer-held Expected
// remains the authority.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"tei/tools/internal/harnessanswer"
)

type output struct {
	Note     string                    `json:"note"`
	Answer   string                    `json:"answerFile"`
	Inspect  string                    `json:"inspectFile"`
	Validate string                    `json:"validateFile,omitempty"`
	Ground   harnessanswer.GroundTruth `json:"groundTruth"`
	Report   harnessanswer.Report      `json:"report"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	fs := flag.NewFlagSet("reference-harness-answer-presence", flag.ContinueOnError)
	answerPath := fs.String("answer", "", "answer markdown captured from the harness")
	inspectPath := fs.String("inspect", "", "captured `tei inspect -json` response for the subject")
	validatePath := fs.String("validate", "", "captured `tei validate -json` response (optional)")
	outPath := fs.String("out", "", "output JSON path; must not exist")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if *answerPath == "" || *inspectPath == "" || *outPath == "" {
		return fmt.Errorf("-answer, -inspect and -out are required")
	}
	if _, err := os.Stat(*outPath); err == nil {
		return fmt.Errorf("output %s already exists; removable output only", *outPath)
	}

	answer, err := os.ReadFile(*answerPath)
	if err != nil {
		return fmt.Errorf("read answer: %w", err)
	}
	inspectBody, err := os.ReadFile(*inspectPath)
	if err != nil {
		return fmt.Errorf("read inspect: %w", err)
	}

	gt, err := harnessanswer.FromInspect(inspectBody)
	if err != nil {
		return fmt.Errorf("inspect ground truth: %w", err)
	}
	if *validatePath != "" {
		validateBody, readErr := os.ReadFile(*validatePath)
		if readErr != nil {
			return fmt.Errorf("read validate: %w", readErr)
		}
		gt, err = gt.WithValidate(validateBody)
		if err != nil {
			return fmt.Errorf("validate ground truth: %w", err)
		}
	}

	doc := output{
		Note:     "Mechanical presence check only: which observed CLI facts the answer mentions. Correctness, authority and completeness are NOT graded here.",
		Answer:   filepath.ToSlash(*answerPath),
		Inspect:  filepath.ToSlash(*inspectPath),
		Validate: filepath.ToSlash(*validatePath),
		Ground:   gt,
		Report:   gt.Check(string(answer)),
	}
	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal output: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(*outPath, encoded, 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s (found %d, missing %d)\n", *outPath, doc.Report.Found, doc.Report.Missing)
	return nil
}
