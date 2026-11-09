// E18 uses two actual parsed file states to join Source evidence and
// typed impact candidates. Neither layer accepts business meaning or grants
// runtime authority.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

const e18SemanticFile = "semantic/sales/order.tei"
const e18PatternFile = "applications/order-entry/patterns/customer-reference.tei"

func e18ReplaceOnce(t *testing.T, root, path, before, after string) {
	t.Helper()
	name := filepath.Join(root, filepath.FromSlash(path))
	contents, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read isolated source: %v", err)
	}
	if strings.Count(string(contents), before) != 1 {
		t.Fatalf("expected one source declaration to change in %s", path)
	}
	replaced := strings.Replace(string(contents), before, after, 1)
	if err := os.WriteFile(name, []byte(replaced), 0600); err != nil {
		t.Fatalf("write isolated source: %v", err)
	}
}

func e18Load(t *testing.T, root string) (workspace.Result, sourceindex.Index, e5Report) {
	t.Helper()
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load copied Workspace: %v", err)
	}
	sources, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("build Source Index from parsed bytes: %v", err)
	}
	report := inspectE5(loaded.World, sources, "order_entry")
	if report.Status != "observed" || len(report.Issues) != 0 {
		t.Fatalf("order entry direct definitions unresolved: %+v", report.Issues)
	}
	return loaded, sources, report
}

func e18ChangedArtifact(t *testing.T, before, after sourceindex.Index, path string) {
	t.Helper()
	oldFingerprint, oldUnique := e17ArtifactFingerprint(before, path)
	newFingerprint, newUnique := e17ArtifactFingerprint(after, path)
	if !oldUnique || !newUnique || oldFingerprint == newFingerprint {
		t.Fatalf("Source change not uniquely observed for %s", path)
	}
}

func TestE18ActualRelationRenameAndPatternRebindingReachesApplication(t *testing.T) {
	root := e17CopyReferenceWorkspace(t)
	beforeWorld, beforeSource, beforeReport := e18Load(t, root)
	oldRelation := e16Key("semantic-relation", "sales.order.customer")
	newRelation := e16Key("semantic-relation", "sales.order.account_contact")
	pattern := e16Key("application-pattern", "order_entry.customer_reference")
	application := e16Key("application", "order_entry")
	if !e14UniqueDefinition(beforeSource, "semantic-relation", oldRelation.Subject.ID, e18SemanticFile) {
		t.Fatal("original Relation Source definition is not unique")
	}
	if source, ok := e5UniqueSource(beforeSource, sourceindex.Subject{
		Kind: "semantic-relation", ID: oldRelation.Subject.ID,
	}); !ok || source.Path != e18SemanticFile {
		t.Fatal("E5 cannot observe the original Relation Source")
	}
	if beforeReport.Status != "observed" {
		t.Fatal("original Application was not observed")
	}

	// A named Relation and its Pattern reference change together. This is
	// an isolated candidate edit, not a validated/accepted business revision.
	e18ReplaceOnce(t, root, e18SemanticFile,
		"relation customer to customer.customer one",
		"relation account_contact to customer.customer one")
	e18ReplaceOnce(t, root, e18PatternFile,
		"relation sales.order.customer",
		"relation sales.order.account_contact")

	// A later disk edit must not rewrite evidence from the original Load.
	pinned, err := workspace.BuildSourceIndex(root, beforeWorld)
	if err != nil {
		t.Fatalf("re-index original parsed snapshot: %v", err)
	}
	for _, path := range []string{e18SemanticFile, e18PatternFile} {
		original, okOld := e17ArtifactFingerprint(beforeSource, path)
		retained, okPinned := e17ArtifactFingerprint(pinned, path)
		if !okOld || !okPinned || original != retained {
			t.Fatalf("parsed Source snapshot changed after write: %s", path)
		}
	}

	afterWorld, afterSource, afterReport := e18Load(t, root)
	if afterReport.Status != "observed" {
		t.Fatal("updated Application was not observed")
	}
	for _, path := range []string{e18SemanticFile, e18PatternFile} {
		e18ChangedArtifact(t, beforeSource, afterSource, path)
	}
	if !e14UniqueDefinition(afterSource, "semantic-relation", newRelation.Subject.ID, e18SemanticFile) ||
		!e14UniqueDefinition(afterSource, "application-pattern", pattern.Subject.ID, e18PatternFile) {
		t.Fatal("changed Relation and Pattern lack unique Source definitions")
	}
	if !e14UniqueDefinition(beforeSource, "application-pattern", pattern.Subject.ID, e18PatternFile) {
		t.Fatal("original Pattern lacks a unique Source definition")
	}

	const scope = "order-entry-reference"
	oldProjection := derivedbridge.FromIndex(index.Build(beforeWorld.World), scope)
	newProjection := derivedbridge.FromIndex(index.Build(afterWorld.World), scope)
	if !e16HasDirectEdge(oldProjection.Graph, pattern, "relation", oldRelation) ||
		!e16HasDirectEdge(newProjection.Graph, pattern, "relation", newRelation) ||
		!e16HasDirectEdge(oldProjection.Graph, application, "owns-pattern", pattern) ||
		!e16HasDirectEdge(newProjection.Graph, application, "owns-pattern", pattern) {
		t.Fatal("changed Relation is not attached to the declared Application path")
	}
	// The projection also has unrelated unresolved references. Do not infer
	// that the whole Workspace graph has a closed dependency set.
	if len(oldProjection.Unresolved) > 0 || len(newProjection.Unresolved) > 0 {
		t.Logf("typed projection incomplete: before=%d after=%d; no full-closure claim",
			len(oldProjection.Unresolved), len(newProjection.Unresolved))
	}
	review := modelref.ReviewImpactDiff(oldProjection.Graph, newProjection.Graph, oldProjection.Rules)
	if !review.GraphValid || len(review.Diagnostics) != 0 {
		t.Fatalf("bounded typed comparison has diagnostics: %+v", review.Diagnostics)
	}
	removed, added := false, false
	for _, change := range review.Changes {
		if change.Key == oldRelation && change.Kind == "removed" {
			removed = true
		}
		if change.Key == newRelation && change.Kind == "added" {
			added = true
		}
	}
	if !removed || !added {
		t.Fatalf("real declaration edit did not change typed Relation identities: %+v", review.Changes)
	}
	foundPattern, foundApplication := false, false
	for _, candidate := range review.Affected {
		if candidate.Consumer == pattern {
			foundPattern = true
		}
		if candidate.Consumer == application {
			foundApplication = true
		}
	}
	if !foundPattern || !foundApplication {
		t.Fatalf("changed Relation did not reach Pattern/Application: pattern=%v application=%v",
			foundPattern, foundApplication)
	}
	if len(review.AuthorityLimits) == 0 {
		t.Fatal("candidate impact cannot lose its authority limitations")
	}
}

func TestE18CardinalityChangeCannotBecomeFalseTypedUnchangedAcceptance(t *testing.T) {
	root := e17CopyReferenceWorkspace(t)
	beforeWorld, beforeSource, _ := e18Load(t, root)
	e18ReplaceOnce(t, root, e18SemanticFile,
		"relation customer to customer.customer one",
		"relation customer to customer.customer many")
	afterWorld, afterSource, _ := e18Load(t, root)
	e18ChangedArtifact(t, beforeSource, afterSource, e18SemanticFile)
	oldRelation := sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.customer"}
	beforeEvidence, beforeOK := e5UniqueSource(beforeSource, oldRelation)
	afterEvidence, afterOK := e5UniqueSource(afterSource, oldRelation)
	if !beforeOK || !afterOK || beforeEvidence.Fingerprint == afterEvidence.Fingerprint {
		t.Fatal("cardinality Source change did not require Relation requalification")
	}
	const scope = "order-entry-reference"
	oldProjection := derivedbridge.FromIndex(index.Build(beforeWorld.World), scope)
	newProjection := derivedbridge.FromIndex(index.Build(afterWorld.World), scope)
	// Existing Derived Index omits Relation cardinality. A typed empty Diff
	// therefore cannot cancel the independently observed Source change.
	if changes := modelref.Diff(oldProjection.Graph, newProjection.Graph); len(changes) != 0 {
		t.Fatalf("unexpected typed identity change for cardinality-only edit: %+v", changes)
	}
	t.Log("Source requalification required despite empty typed Diff; no automatic acceptance")
}
