//go:build tei_modelref_candidate

// E15 remains opt-in because it invokes a Node.js child process.
// The modelref and Source Index snapshot packages are already in main.
// E16 exercises the Go-only path in the default package tests.
package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/subject"
	"tei/internal/workspace"
)

type e15Review struct {
	Status           string `json:"status"`
	Reason           string `json:"reason"`
	Authority        string `json:"authority"`
	FullClosure      bool   `json:"fullClosure"`
	Adoption         string `json:"adoption"`
	ReviewCandidates []struct {
		Consumer modelref.Key `json:"consumer"`
	} `json:"reviewCandidates"`
}

func e15WriteJSON(t *testing.T, name string, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal %s: %v", name, err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write temporary %s: %v", name, err)
	}
	return path
}

func e15RunCLI(t *testing.T, before, after e5EvidenceEnvelope, typed any) e15Review {
	t.Helper()
	base := e15WriteJSON(t, "base.json", before)
	curr := e15WriteJSON(t, "current.json", after)
	model := e15WriteJSON(t, "modelref.json", typed)
	cmd := exec.Command("node", "e12-bound-review.mjs", base, curr, model,
		"order_entry", "order-entry-reference")
	// Only the read-only E12 output is parsed. Avoid logging full evidence or
	// raw stderr; Source Index paths can contain sensitive information.
	out, err := cmd.Output()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 2 {
		t.Fatalf("E12 must exit 2 (read-only, not qualified); error kind=%T", err)
	}
	var reviewed e15Review
	if err := json.Unmarshal(out, &reviewed); err != nil {
		t.Fatalf("E12 response is not machine-readable: %v", err)
	}
	if reviewed.Status != "unresolved" || reviewed.FullClosure ||
		reviewed.Authority != "none" || reviewed.Adoption != "not-decided" {
		t.Fatalf("E12 improperly acquired authority (status=%q, reason=%q)", reviewed.Status, reviewed.Reason)
	}
	return reviewed
}

func TestE15ActualGoImpactToE12ReadOnlyReview(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load workspace: %v", err)
	}
	src, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("build Source Index: %v", err)
	}
	report := inspectE5(loaded.World, src, "order_entry")
	if report.Status != "observed" {
		t.Fatalf("E5 direct definitions unresolved, issue count=%d", len(report.Issues))
	}

	const scope = "order-entry-reference"
	full := index.Build(loaded.World)
	projection := derivedbridge.FromIndex(full, scope)
	diagnostics := modelref.Validate(projection.Graph, projection.Rules)
	relation := modelref.Key{Scope: scope, Subject: subject.Subject{Kind: "semantic-relation", ID: "sales.order.customer"}}
	pattern := modelref.Key{Scope: scope, Subject: subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"}}
	application := modelref.Key{Scope: scope, Subject: subject.Subject{Kind: "application", ID: "order_entry"}}
	affected := modelref.Impact(projection.Graph, []modelref.Change{{
		Key: relation, Kind: "version-changed",
	}}, projection.Rules)
	foundPattern, foundApplication := false, false
	for _, item := range affected {
		if item.Consumer == pattern {
			foundPattern = true
		}
		if item.Consumer == application {
			foundApplication = true
		}
	}
	if !foundPattern || !foundApplication {
		t.Fatalf("actual Go Impact lost selected Pattern/Application: pattern=%v app=%v", foundPattern, foundApplication)
	}

	// Normalize Go's nil/omitempty slices at the *trusted Go serialization
	// boundary*. E11 intentionally rejects omitted or null arrays in arbitrary
	// external JSON; no generic JS fallback is introduced.
	goCarrier := map[string]any{
		"projection": map[string]any{
			"graph":           projection.Graph,
			"rules":           append([]modelref.Rule{}, projection.Rules...),
			"unresolved":      append([]string{}, projection.Unresolved...),
			"authorityLimits": append([]string{}, projection.AuthorityLimits...),
		},
		"diagnostics": append([]modelref.Diagnostic{}, diagnostics...),
		"affected":    append([]modelref.Affected{}, affected...),
	}
	// E5 and modelref originate from one loaded.World in this Go invocation.
	// The merged #1017 shares parsed source bytes with Source Index, but live
	// filesystem races, unparsed assets and pinned Git revision still need
	// their own qualification before asserting a durable common snapshot.
	envelope := e13PairedEvidence(loaded.World, report, src, true)
	if envelope.Derived == nil {
		t.Fatal("same-World E13 direct Derived Index is missing")
	}
	review := e15RunCLI(t, envelope, envelope, goCarrier)
	if len(projection.Unresolved) != 0 || len(diagnostics) != 0 {
		// The existing Derived projection intentionally does not claim closure
		// for every reference in the Workspace. Verify that the real Node
		// review refuses to promote even the known Go Impact candidates.
		if review.Reason != "unresolved_typed_references" {
			t.Fatalf("typed projection was incomplete but review reason=%q", review.Reason)
		}
		if len(review.ReviewCandidates) != 0 {
			t.Fatalf("unresolved typed projection leaked %d review candidates", len(review.ReviewCandidates))
		}
		t.Logf("E15 correctly rejected incomplete projection: unresolved=%d diagnostics=%d; no acceptance or authority",
			len(projection.Unresolved), len(diagnostics))
		return
	}
	if review.Reason != "unverified_shared_snapshot" {
		t.Fatalf("E12 prematurely accepted Go evidence, reason=%q", review.Reason)
	}
	seen := map[modelref.Key]bool{}
	for _, candidate := range review.ReviewCandidates {
		seen[candidate.Consumer] = true
	}
	if !seen[pattern] || !seen[application] {
		t.Fatalf("E12 dropped real Go Impact candidates: pattern=%v app=%v", seen[pattern], seen[application])
	}
}
