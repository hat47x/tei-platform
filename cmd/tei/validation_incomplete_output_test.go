package main

import (
	"bytes"
	"strings"
	"testing"
)

// The JSON form has said `incomplete` since #4178; the text form printed only the
// first diagnostic, so a run that stopped at the first syntax error read as
// "there is one problem".
func TestTextOutputSaysTheRunIsIncomplete(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: incomplete_text\nformatVersion: 1\nsources:\n  - applications\n")
	writeWorkspaceFile(t, root, "applications/a/application.tei", `application a {
    frobnicate
}
`)
	// A second broken file that the run never reaches.
	writeWorkspaceFile(t, root, "applications/b/application.tei", `application b {
    frobnicate
}
`)

	var out bytes.Buffer
	code, err := validateWorkspace(&out, root)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 {
		t.Fatalf("exit code=%d output=%q", code, out.String())
	}
	if !strings.Contains(out.String(), "VALIDATION_INCOMPLETE") {
		t.Fatalf("text output does not say the run is incomplete: %q", out.String())
	}
	if !strings.Contains(out.String(), "were not examined") {
		t.Errorf("text output does not say other files were unexamined: %q", out.String())
	}
	if !strings.Contains(out.String(), "applications/a/application.tei") {
		t.Errorf("text output does not name the file it stopped at: %q", out.String())
	}
}

// A clean run must not claim truncation.
func TestTextOutputDoesNotClaimIncompletenessWhenClean(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: complete_text\nformatVersion: 1\nsources:\n  - applications\n")
	writeWorkspaceFile(t, root, "applications/a/application.tei", `application a {
    uses semantic sales
}
`)

	var out bytes.Buffer
	if _, err := validateWorkspace(&out, root); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "VALIDATION_INCOMPLETE") {
		t.Fatalf("a clean run claimed truncation: %q", out.String())
	}
}

// A semantic diagnostic is a complete answer: every file was parsed, so the read
// must not be labelled incomplete.
func TestTextOutputDoesNotClaimIncompletenessForSemanticFindings(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: semantic_text\nformatVersion: 1\nsources:\n  - applications\n")
	writeWorkspaceFile(t, root, "applications/order-entry/application.tei", `application order_entry {
    database sales {
        target sales_core
        access invalid
    }
}
`)

	var out bytes.Buffer
	code, err := validateWorkspace(&out, root)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 {
		t.Fatalf("exit code=%d output=%q", code, out.String())
	}
	if strings.Contains(out.String(), "VALIDATION_INCOMPLETE") {
		t.Fatalf("a semantic finding was labelled a truncated run: %q", out.String())
	}
}
