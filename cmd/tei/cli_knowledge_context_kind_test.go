package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestKnowledgeContextAcceptsTheCatalogNamespaceKind pins the kind-name
// mismatch between the read commands. `model catalog`, `inspect` and `trace`
// call a namespace "semantic"; `knowledge-context` used
// "semantic-namespace". A caller that copies `entries[].subject.kind` from the
// catalog into `-subject-kind` therefore got `ok:true` with no
// `result.semanticElement`, which reads as "this namespace has no definition"
// instead of "this kind name is not the one I use" (#3242 follow-up).
func TestKnowledgeContextAcceptsTheCatalogNamespaceKind(t *testing.T) {
	binary := buildCLIForLifecycle(t)
	reference := filepath.Join("..", "..", "reference", "workspace")

	run := func(args ...string) []byte {
		t.Helper()
		command := exec.Command(binary, args...)
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("%q: CLI execution: %v", args, err)
			}
			if exit.ExitCode() != 0 {
				t.Fatalf("%q: exit=%d stdout=%q stderr=%q", args, exit.ExitCode(), stdout.String(), stderr.String())
			}
		}
		if stderr.Len() > 0 {
			t.Fatalf("%q wrote stderr=%q", args, stderr.String())
		}
		return stdout.Bytes()
	}

	// The catalog advertises the namespace as "semantic"; read that value from
	// the catalog instead of hardcoding it, so a rename in discovery is caught.
	var catalog struct {
		Result struct {
			Entries []struct {
				Subject struct {
					Kind string `json:"kind"`
					ID   string `json:"id"`
				} `json:"subject"`
			} `json:"entries"`
		} `json:"result"`
	}
	if err := json.Unmarshal(run("model", "catalog", "-workspace", reference), &catalog); err != nil {
		t.Fatalf("model catalog: %v", err)
	}
	var namespaceKind, namespaceID string
	for _, entry := range catalog.Result.Entries {
		if entry.Subject.Kind == "semantic" && entry.Subject.ID == "sales" {
			namespaceKind, namespaceID = entry.Subject.Kind, entry.Subject.ID
			break
		}
	}
	if namespaceKind == "" {
		t.Fatal("model catalog no longer advertises the sales namespace")
	}

	type contextBody struct {
		OK     bool `json:"ok"`
		Result struct {
			Subject struct {
				Kind string `json:"kind"`
				ID   string `json:"id"`
			} `json:"subject"`
			SemanticElement *struct {
				Kind string `json:"kind"`
				ID   string `json:"id"`
			} `json:"semanticElement"`
			Unresolved []string `json:"unresolved"`
		} `json:"result"`
	}

	// Both names must resolve the namespace element. The documented name
	// remains semantic-namespace; the catalog's name is accepted as an alias.
	for _, kind := range []string{namespaceKind, "semantic-namespace"} {
		var body contextBody
		if err := json.Unmarshal(run("knowledge-context", "-workspace", reference,
			"-subject-kind", kind, "-subject-id", namespaceID), &body); err != nil {
			t.Fatalf("kind %q: %v", kind, err)
		}
		if !body.OK || body.Result.SemanticElement == nil {
			t.Fatalf("kind %q did not resolve the namespace: %+v", kind, body.Result.Unresolved)
		}
		if body.Result.SemanticElement.Kind != "namespace" || body.Result.SemanticElement.ID != namespaceID {
			t.Fatalf("kind %q resolved %+v", kind, body.Result.SemanticElement)
		}
		if body.Result.Subject.Kind != kind {
			t.Fatalf("kind %q was rewritten to %q in the response", kind, body.Result.Subject.Kind)
		}
	}

	// A kind the context does not model is not guessed: it stays unresolved and
	// the subject is echoed unchanged, so a caller can tell the difference.
	var unknown contextBody
	if err := json.Unmarshal(run("knowledge-context", "-workspace", reference,
		"-subject-kind", "not-a-kind", "-subject-id", namespaceID), &unknown); err != nil {
		t.Fatal(err)
	}
	if !unknown.OK || unknown.Result.SemanticElement != nil {
		t.Fatalf("unknown kind resolved an element: %+v", unknown.Result)
	}
	if unknown.Result.Subject.Kind != "not-a-kind" {
		t.Fatalf("unknown kind was rewritten: %+v", unknown.Result.Subject)
	}
}
