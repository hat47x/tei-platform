package exampleslide

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

func ruleSourceFixture(t *testing.T) (string, Input, ModelHypotheses, ModelRuleDraftIntent, string, []byte) {
	t.Helper()
	root, input, hypotheses, intent := modelRuleDraftFixture(t)
	path := "proposals/model-rules.tei"
	raw := []byte(strings.Join([]string{
		"# Reference-only, unaccepted source",
		"tei-model-rules v0",
		"candidate-key sales.child.business-parent on sales.child fields sales.child.parent_id",
		"functional-dependency sales.child.parent-depends-on-itself on sales.child determinants sales.child.parent_id dependents sales.child.parent_id",
		"",
	}, "\n"))
	return root, input, hypotheses, intent, path, raw
}

func TestModelRuleSourceAnchorsAuthoredRuleIdentitiesExactly(t *testing.T) {
	root, input, hypotheses, intent, path, content := ruleSourceFixture(t)
	review, err := ReviewLoadedWorkspaceRuleSource(root, input, hypotheses, intent, path, content)
	if err != nil {
		t.Fatal(err)
	}
	if review.Schema != "tei.reference.model-rule-source-review/v0" ||
		len(review.Entries) != 2 || len(review.SourceIndex.References) != 2 ||
		len(review.SourceIndex.Artifacts) != 1 || len(review.Unresolved) == 0 ||
		!strings.HasPrefix(review.Fingerprint, "sha256:") {
		t.Fatalf("incomplete source review: %+v", review)
	}
	if review.SourceIndex.Artifacts[0].Fingerprint != sourceindex.Fingerprint(content) {
		t.Fatalf("source bytes not bound to indexed fingerprint: %+v", review.SourceIndex)
	}
	for i, ref := range review.SourceIndex.References {
		if ref.Role != sourceindex.RoleAuthoring || ref.Subject.Kind != "reference-"+review.Entries[i].Kind ||
			ref.Subject.ID != review.Entries[i].RuleID || ref.ArtifactPath != path ||
			ref.Range.Start.Offset >= ref.Range.End.Offset ||
			string(content[ref.Range.Start.Offset:ref.Range.End.Offset]) != ref.Subject.ID ||
			review.Entries[i].DraftStatus != "unverified-business-rule" {
			t.Fatalf("wrong location, source scope or authority: %+v", ref)
		}
	}
	reversed := bytes.ReplaceAll(content,
		[]byte("candidate-key sales.child.business-parent on sales.child fields sales.child.parent_id\n"),
		[]byte(""))
	reversed = append(reversed, []byte("candidate-key sales.child.business-parent on sales.child fields sales.child.parent_id\n")...)
	again, err := ReviewLoadedWorkspaceRuleSource(root, input, hypotheses, intent, path, reversed)
	if err != nil {
		t.Fatal(err)
	}
	if review.Fingerprint == again.Fingerprint || reflect.DeepEqual(review.SourceIndex, again.SourceIndex) {
		t.Fatal("moving rule source location did not change source evidence")
	}
	if _, err := ReviewLoadedWorkspaceRuleSource(root, input, hypotheses, intent, path, content); err != nil {
		t.Fatal(err)
	}
}

func TestModelRuleSourceRejectsDriftUnexpectedAndMissingRules(t *testing.T) {
	tests := []struct {
		name    string
		rewrite func(string) string
	}{
		{"header", func(s string) string {
			return strings.Replace(s, "tei-model-rules v0", "tei-model-rules v1", 1)
		}},
		{"kind", func(s string) string {
			return strings.Replace(s, "candidate-key sales.child.business-parent", "functional-dependency sales.child.business-parent", 1)
		}},
		{"table", func(s string) string {
			return strings.Replace(s, "on sales.child fields", "on sales.other fields", 1)
		}},
		{"field", func(s string) string {
			return strings.Replace(s, "fields sales.child.parent_id", "fields sales.child.other", 1)
		}},
		{"unknown", func(s string) string {
			return strings.Replace(s, "business-parent", "unknown-rule", 1)
		}},
		{"duplication", func(s string) string {
			return s + "candidate-key sales.child.business-parent on sales.child fields sales.child.parent_id\n"
		}},
		{"missing", func(s string) string {
			return strings.Replace(s, "functional-dependency sales.child.parent-depends-on-itself on sales.child determinants sales.child.parent_id dependents sales.child.parent_id\n", "", 1)
		}},
		{"extra token", func(s string) string {
			return strings.Replace(s, "fields sales.child.parent_id", "fields sales.child.parent_id accepted", 1)
		}},
		{"wrong FD", func(s string) string {
			return strings.Replace(s, "dependents sales.child.parent_id", "dependents sales.child.other", 1)
		}},
		{"duplicate attribute", func(s string) string {
			return strings.Replace(s, "fields sales.child.parent_id", "fields sales.child.parent_id,sales.child.parent_id", 1)
		}},
		{"bad first", func(s string) string {
			return "candidate-key x on y fields z\n" + s
		}},
		{"trailing declaration", func(s string) string {
			return s + "not-a-rule sales.child.fake on sales.child fields sales.child.parent_id"
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, input, hypotheses, intent, path, content := ruleSourceFixture(t)
			if _, err := ReviewLoadedWorkspaceRuleSource(root, input, hypotheses, intent, path, []byte(tc.rewrite(string(content)))); err == nil {
				t.Fatal("unexpected, stale, or malformed source was indexed")
			}
		})
	}
}

func TestModelRuleSourceRejectsPathTraversalOrUnsafeBytes(t *testing.T) {
	root, input, hypotheses, intent, _, content := ruleSourceFixture(t)
	for _, path := range []string{"../escape.rules", "/tmp/rules", "a/../proposals/rules.tei", "C:/outside.rules", ""} {
		if _, err := ReviewLoadedWorkspaceRuleSource(root, input, hypotheses, intent, path, content); err == nil {
			t.Fatalf("unsafe source path %q was accepted", path)
		}
	}
	for _, invalid := range [][]byte{nil, bytes.Repeat([]byte("x"), maxRuleSourceBytes+1), append(bytes.Clone(content), 0)} {
		if _, err := ReviewLoadedWorkspaceRuleSource(root, input, hypotheses, intent, "proposals/rules.tei", invalid); err == nil {
			t.Fatal("unsafe source content was accepted")
		}
	}
}

func TestModelRuleSourceRejectsStaleBusinessModelAndForgedIntents(t *testing.T) {
	root, input, hypotheses, intent, path, content := ruleSourceFixture(t)
	intent.ModelReviewFingerprint = "sha256:" + strings.Repeat("0", 64)
	if _, err := ReviewLoadedWorkspaceRuleSource(root, input, hypotheses, intent, path, content); err == nil {
		t.Fatal("forged rule intent passed source binding")
	}
	root, input, hypotheses, intent, path, content = ruleSourceFixture(t)
	input.Example.ModelRevision = "stale"
	if _, err := ReviewLoadedWorkspaceRuleSource(root, input, hypotheses, intent, path, content); err == nil {
		t.Fatal("stale example was accepted")
	}
}

func TestModelRuleSourceDoesNotAcceptOrAmendDraft(t *testing.T) {
	root, input, hypotheses, intent, path, content := ruleSourceFixture(t)
	draft, err := DraftLoadedWorkspaceModelRules(root, input, hypotheses, intent)
	if err != nil {
		t.Fatal(err)
	}
	before := digestHypothesisJSON(draft)
	result, err := IndexModelRuleDraftSource(draft, path, content)
	if err != nil {
		t.Fatal(err)
	}
	if digestHypothesisJSON(draft) != before || len(result.SourceIndex.References) != 2 {
		t.Fatal("creating rule source altered the draft or discarded locators")
	}
	for _, ref := range result.SourceIndex.References {
		if ref.Role == sourceindex.RoleDefinition || strings.HasPrefix(ref.Subject.Kind, "semantic-") {
			t.Fatalf("unaccepted rule was indexed as canonical semantic definition: %+v", ref)
		}
	}
	if !strings.Contains(strings.Join(result.Unresolved, " "), "Shared Acceptance") {
		t.Fatal("source review has no explicit acceptance limitation")
	}
	draft.Fingerprint = "sha256:" + strings.Repeat("0", 64)
	if _, err := IndexModelRuleDraftSource(draft, path, content); err == nil {
		t.Fatal("forged draft self fingerprint accepted")
	}
}
