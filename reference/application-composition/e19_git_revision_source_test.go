// E19 fixes two actual Git commit snapshots from a copied reference Workspace.
// These fixture commits are not published or accepted business revisions.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/sourceindex"
)

func e19Git(t *testing.T, root string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return out
}

func e19Commit(t *testing.T, root, message string) string {
	t.Helper()
	e19Git(t, root, "add", "--all")
	e19Git(t, root, "-c", "user.name=TEI Test",
		"-c", "user.email=tei-test@example.invalid",
		"commit", "--quiet", "-m", message)
	sha := strings.TrimSpace(string(e19Git(t, root, "rev-parse", "HEAD")))
	if len(sha) != 40 {
		t.Fatalf("unexpected Git revision %q", sha)
	}
	return sha
}

// A clean worktree and an exact Git object-to-Source Index byte match are
// required before presenting a source observation as pinned to a revision.
func e19PinnedEvidence(t *testing.T, root, revision string, sources sourceindex.Index) error {
	t.Helper()
	head := strings.TrimSpace(string(e19Git(t, root, "rev-parse", "HEAD")))
	if head != revision {
		return fmt.Errorf("worktree revision mismatch: %s != %s", head, revision)
	}
	if dirty := e19Git(t, root, "status", "--porcelain=v1", "--untracked-files=all"); len(dirty) > 0 {
		return fmt.Errorf("worktree is dirty; pinned Source cannot be trusted")
	}
	for _, path := range []string{e18SemanticFile, e18PatternFile} {
		gitBytes := e19Git(t, root, "show", revision+":"+path)
		actualBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return err
		}
		if !bytes.Equal(actualBytes, gitBytes) {
			return fmt.Errorf("%s differs from pinned Git blob", path)
		}
		evidence, unique := e17ArtifactFingerprint(sources, path)
		if !unique || evidence != sourceindex.Fingerprint(gitBytes) {
			return fmt.Errorf("%s Source Index fingerprint disagrees with pinned Git bytes", path)
		}
	}
	return nil
}

func TestE19GitPinnedSemanticChangeProducesScopedReviewCandidates(t *testing.T) {
	// Commits are created in a disposable repository using the current real
	// Workspace as the baseline. No existing repository history is rewritten.
	root := e17CopyReferenceWorkspace(t)
	e19Git(t, root, "init", "--quiet")
	oldCommit := e19Commit(t, root, "reference Workspace baseline")
	e18ReplaceOnce(t, root, e18SemanticFile,
		"relation customer to customer.customer one",
		"relation account_contact to customer.customer one")
	e18ReplaceOnce(t, root, e18PatternFile,
		"relation sales.order.customer",
		"relation sales.order.account_contact")
	newCommit := e19Commit(t, root, "candidate semantic Relation rename and Pattern update")
	if oldCommit == newCommit {
		t.Fatal("candidate revision did not advance")
	}
	changed := strings.Fields(string(e19Git(t, root,
		"diff", "--name-only", oldCommit, newCommit)))
	if len(changed) != 2 ||
		changed[0] != e18PatternFile || changed[1] != e18SemanticFile {
		t.Fatalf("revision includes unexpected changes: %q", changed)
	}

	// Load each named commit from a real Git worktree, not from two caller
	// supplied JSON objects or the same mutable filesystem root.
	previousRoot := filepath.Join(t.TempDir(), "previous")
	e19Git(t, root, "worktree", "add", "--quiet", "--detach", previousRoot, oldCommit)
	previous, previousSources, oldReport := e18Load(t, previousRoot)
	current, currentSources, newReport := e18Load(t, root)
	if err := e19PinnedEvidence(t, previousRoot, oldCommit, previousSources); err != nil {
		t.Fatalf("original revision is not pinned: %v", err)
	}
	if err := e19PinnedEvidence(t, root, newCommit, currentSources); err != nil {
		t.Fatalf("candidate revision is not pinned: %v", err)
	}
	for _, path := range changed {
		e18ChangedArtifact(t, previousSources, currentSources, path)
	}
	if oldReport.Status != "observed" || newReport.Status != "observed" {
		t.Fatal("Application declarations were not observed in both revisions")
	}
	oldRelation := e16Key("semantic-relation", "sales.order.customer")
	newRelation := e16Key("semantic-relation", "sales.order.account_contact")
	pattern := e16Key("application-pattern", "order_entry.customer_reference")
	application := e16Key("application", "order_entry")
	if !e14UniqueDefinition(previousSources, "semantic-relation", oldRelation.Subject.ID, e18SemanticFile) ||
		!e14UniqueDefinition(currentSources, "semantic-relation", newRelation.Subject.ID, e18SemanticFile) {
		t.Fatal("Relation definitions are not bound to their revision-specific Source")
	}
	const scope = "order-entry-reference"
	before := derivedbridge.FromIndex(index.Build(previous.World), scope)
	after := derivedbridge.FromIndex(index.Build(current.World), scope)
	if !e16HasDirectEdge(before.Graph, pattern, "relation", oldRelation) ||
		!e16HasDirectEdge(after.Graph, pattern, "relation", newRelation) ||
		!e16HasDirectEdge(before.Graph, application, "owns-pattern", pattern) ||
		!e16HasDirectEdge(after.Graph, application, "owns-pattern", pattern) {
		t.Fatal("pinned Relation change lacks Application ownership path")
	}
	review := modelref.ReviewImpactDiff(before.Graph, after.Graph, before.Rules)
	if !review.GraphValid || len(review.Diagnostics) != 0 {
		t.Fatalf("typed review contains unresolved graph diagnostics: %+v", review.Diagnostics)
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
		t.Fatalf("pinned Source changes not reflected in typed references: %+v", review.Changes)
	}
	seen := map[modelref.Key]bool{}
	for _, impact := range review.Affected {
		seen[impact.Consumer] = true
	}
	if !seen[pattern] || !seen[application] || len(review.AuthorityLimits) == 0 {
		t.Fatalf("missing affected application or authority limits: pattern=%v application=%v limits=%v",
			seen[pattern], seen[application], review.AuthorityLimits)
	}
	if len(before.Unresolved) > 0 || len(after.Unresolved) > 0 {
		t.Logf("unresolved typed references remain: before=%d after=%d; no full closure",
			len(before.Unresolved), len(after.Unresolved))
	}

	// Tampering after qualification must invalidate the Git provenance; a
	// stable revision string alone must never authorize a dirty Workspace.
	e18ReplaceOnce(t, root, e18SemanticFile,
		"relation account_contact to customer.customer one",
		"relation alternate_contact to customer.customer one")
	if err := e19PinnedEvidence(t, root, newCommit, currentSources); err == nil {
		t.Fatal("uncommitted Source tampering was accepted as a pinned Git revision")
	}
}
