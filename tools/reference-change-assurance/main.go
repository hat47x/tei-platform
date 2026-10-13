package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"tei/internal/modelref"
	"tei/internal/subject"
)

func run(reader io.Reader, writer io.Writer) error {
	d := json.NewDecoder(reader)
	d.DisallowUnknownFields()
	var fixture Fixture
	if err := d.Decode(&fixture); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	var rest any
	if err := d.Decode(&rest); err != io.EOF {
		return fmt.Errorf("fixture must be exactly one JSON value")
	}
	report, err := evaluate(fixture)
	if err != nil {
		return err
	}
	e := json.NewEncoder(writer)
	e.SetIndent("", "  ")
	return e.Encode(report)
}

func main() {
	input := flag.String("input", "", "path to synthetic, read-only fixture")
	workspaceRoot := flag.String("workspace", "", "path to existing read-only Workspace")
	beforeRoot := flag.String("before-workspace", "", "path to read-only Workspace before the change")
	afterRoot := flag.String("after-workspace", "", "path to read-only Workspace after the change")
	scope := flag.String("scope", "", "existing modelref scope for Workspace review")
	changeSubjectKind := flag.String("change-subject-kind", "", "existing changed subject kind")
	changeSubjectID := flag.String("change-subject-id", "", "existing changed logical ID")
	changeKind := flag.String("change-kind", "", "proposed modelref change kind (not an observed diff)")
	consumerKind := flag.String("consumer-kind", "", "consumer subject kind")
	consumerID := flag.String("consumer-id", "", "consumer logical ID")
	flag.Parse()

	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	if *beforeRoot != "" || *afterRoot != "" {
		if *beforeRoot == "" || *afterRoot == "" || *workspaceRoot != "" || *input != "" ||
			*scope == "" || *consumerKind == "" || *consumerID == "" ||
			*changeSubjectKind != "" || *changeSubjectID != "" || *changeKind != "" {
			fmt.Fprintln(os.Stderr, "usage: reference-change-assurance -before-workspace <dir> -after-workspace <dir> -scope <scope> -consumer-kind <kind> -consumer-id <id>")
			os.Exit(2)
		}
		consumer := modelref.Key{Scope: *scope, Subject: subject.Subject{Kind: *consumerKind, ID: *consumerID}}
		report, err := reviewWorkspaceDiff(*beforeRoot, *afterRoot, *scope, consumer)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *workspaceRoot != "" {
		if *input != "" || *scope == "" || *changeSubjectKind == "" || *changeSubjectID == "" ||
			*changeKind == "" || *consumerKind == "" || *consumerID == "" {
			fmt.Fprintln(os.Stderr, "usage: reference-change-assurance -workspace <dir> -scope <scope> -change-subject-kind <kind> -change-subject-id <id> -change-kind <kind> -consumer-kind <kind> -consumer-id <id>")
			os.Exit(2)
		}
		changed := modelref.Key{Scope: *scope, Subject: subject.Subject{Kind: *changeSubjectKind, ID: *changeSubjectID}}
		consumer := modelref.Key{Scope: *scope, Subject: subject.Subject{Kind: *consumerKind, ID: *consumerID}}
		report, err := reviewWorkspaceChange(*workspaceRoot, *scope, changed, *changeKind, consumer)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *input == "" || *scope != "" || *changeSubjectKind != "" || *changeSubjectID != "" ||
		*changeKind != "" || *consumerKind != "" || *consumerID != "" {
		fmt.Fprintln(os.Stderr, "usage: reference-change-assurance -input <fixture.json>")
		os.Exit(2)
	}
	file, err := os.Open(*input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer file.Close()
	if err := run(file, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
