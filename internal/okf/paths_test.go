package okf

import (
	"testing"
)

// existsSet is a pure file-set predicate for resolveTarget tests.
func existsSet(paths ...string) func(string) bool {
	set := map[string]bool{}
	for _, p := range paths {
		set[p] = true
	}
	return func(candidate string) bool { return set[candidate] }
}

// TestResolveTargetForms covers OKF v0.2 section 6.2's three accepted forms and the
// reader's refusal to choose between two bases that both resolve.
func TestResolveTargetForms(t *testing.T) {
	existing := existsSet("data/order.md", "data/neighbor.md", "metrics/revenue.md")
	for _, tc := range []struct {
		name       string
		containing string
		raw        string
		want       Resolution
		resolved   []string
	}{
		{name: "absolute https", containing: "data/order.md", raw: "https://example.test/x", want: ResolutionAbsoluteURL},
		{name: "mailto", containing: "data/order.md", raw: "mailto:someone@example.test", want: ResolutionAbsoluteURL},
		{name: "bundle root form", containing: "data/order.md", raw: "/metrics/revenue.md", want: ResolutionBundleRoot, resolved: []string{"metrics/revenue.md"}},
		{name: "document relative", containing: "data/order.md", raw: "neighbor.md", want: ResolutionDocumentRelative, resolved: []string{"data/neighbor.md"}},
		{name: "parent relative stays in bundle", containing: "functions/submit.md", raw: "../data/order.md", want: ResolutionDocumentRelative, resolved: []string{"data/order.md"}},
		{name: "bundle root wins when only it resolves", containing: "functions/submit.md", raw: "metrics/revenue.md", want: ResolutionBundleRoot, resolved: []string{"metrics/revenue.md"}},
		{name: "scope descriptor", containing: "data/order.md", raw: "all rows in the sample", want: ResolutionScopeDescriptor},
		{name: "unresolved", containing: "data/order.md", raw: "missing/thing.md", want: ResolutionUnresolved},
		{name: "traversal", containing: "a/b/c.md", raw: "../../../../etc/passwd", want: ResolutionUnresolved},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ref := resolveTarget(tc.containing, tc.raw, "probe", 1, existing)
			if ref.Resolution != tc.want {
				t.Fatalf("resolution = %q, want %q", ref.Resolution, tc.want)
			}
			if tc.resolved != nil {
				if len(ref.Resolved) != len(tc.resolved) {
					t.Fatalf("resolved = %v, want %v", ref.Resolved, tc.resolved)
				}
				for i := range tc.resolved {
					if ref.Resolved[i] != tc.resolved[i] {
						t.Fatalf("resolved[%d] = %q, want %q", i, ref.Resolved[i], tc.resolved[i])
					}
				}
			}
		})
	}
}

// TestResolveTargetAmbiguityIsReported covers a relative value that resolves to two
// different files. The reader keeps both and labels the result ambiguous instead of
// picking one silently.
func TestResolveTargetAmbiguityIsReported(t *testing.T) {
	existing := existsSet("data/order.md", "metrics/data/order.md")
	ref := resolveTarget("metrics/revenue.md", "data/order.md", "sources[].resource", 4, existing)
	if ref.Resolution != ResolutionAmbiguous {
		t.Fatalf("resolution = %q, want %q", ref.Resolution, ResolutionAmbiguous)
	}
	if len(ref.Resolved) != 2 {
		t.Fatalf("resolved = %v, want both candidates", ref.Resolved)
	}
}

// TestEscapesBundleIsRelativeToTheContainingDocument covers the traversal guard.
// A normal relative link must not be reported as an escape.
func TestEscapesBundleIsRelativeToTheContainingDocument(t *testing.T) {
	for _, tc := range []struct {
		containing string
		raw        string
		want       bool
	}{
		{containing: "functions/submit.md", raw: "../data/order.md", want: false},
		{containing: "functions/submit.md", raw: "./submit.md", want: false},
		{containing: "index.md", raw: "data/order.md", want: false},
		{containing: "index.md", raw: "/data/order.md", want: false},
		{containing: "functions/submit.md", raw: "../../../etc/passwd", want: true},
		{containing: "a/b/c.md", raw: "../../../outside.md", want: true},
		{containing: "data/order.md", raw: "https://example.test/x", want: false},
		{containing: "data/order.md", raw: "C:/Windows/win.ini", want: true},
	} {
		if got := escapesBundle(tc.containing, tc.raw); got != tc.want {
			t.Errorf("escapesBundle(%q, %q) = %t, want %t", tc.containing, tc.raw, got, tc.want)
		}
	}
}

// TestCleanBundleCandidateRefusesEscapes covers the path normalizer.
func TestCleanBundleCandidateRefusesEscapes(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want string
	}{
		{raw: "data/order.md", want: "data/order.md"},
		{raw: "./data/order.md", want: "data/order.md"},
		{raw: "a//b.md", want: "a/b.md"},
		{raw: "../x.md", want: ""},
		{raw: "/abs.md", want: ""},
		{raw: "C:/x.md", want: ""},
		{raw: "", want: ""},
	} {
		if got := cleanBundleCandidate(tc.raw); got != tc.want {
			t.Errorf("cleanBundleCandidate(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}
