package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #4486 S1: the entry can look at the two state directories itself. These tests pin
// what each look can and cannot show, so the basis recorded in the report matches
// the evidence that actually backes it.

func TestListStateDirEntries(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := listStateDirEntries(dir)
	if err != nil {
		t.Fatalf("list = %v", err)
	}
	if strings.Join(got, ",") != "a.txt,b.txt" {
		t.Fatalf("names = %v, want sorted a.txt,b.txt", got)
	}

	// A child that ignores -state never creates the directory. That has to read as
	// "nothing was written", not as a reason the check could not run.
	missing, err := listStateDirEntries(filepath.Join(dir, "absent"))
	if err != nil {
		t.Fatalf("absent directory should not be an error: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("absent directory = %v, want empty", missing)
	}
}

func TestStateSeparationListingVerdict(t *testing.T) {
	cases := []struct {
		name          string
		before, after []string
		want          string
	}{
		{
			name:   "the second build left the first run's state alone",
			before: []string{"a", "b"},
			after:  []string{"a", "b"},
		},
		{
			name:   "the second build added a file to the first run's state",
			before: []string{"a"},
			after:  []string{"a", "b"},
			want:   "the second build changed the first build's state directory, so the two runs were not separate",
		},
		{
			// Same count, different names: a length check alone would miss this.
			name:   "the second build replaced a file in the first run's state",
			before: []string{"a", "b"},
			after:  []string{"a", "c"},
			want:   "the second build changed the first build's state directory, so the two runs were not separate",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := stateSeparationListingVerdict(tc.before, tc.after); got != tc.want {
				t.Fatalf("verdict = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStateSeparationSecondDirVerdict(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "state"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got := stateSeparationSecondDirVerdict(filepath.Join(dir, "state")); got != "" {
		t.Fatalf("existing directory verdict = %q, want none", got)
	}
	if got := stateSeparationSecondDirVerdict(filepath.Join(dir, "elsewhere")); !strings.Contains(got, "did not create the state directory") {
		t.Fatalf("missing directory verdict = %q", got)
	}
	if got := stateSeparationSecondDirVerdict("  "); !strings.Contains(got, "no state directory was passed") {
		t.Fatalf("empty path verdict = %q", got)
	}
}

// The basis must not overstate what was seen. Post-hoc inspection shows that two
// runs did not share the directory; it does not show that a write was prevented,
// and #4486 measured a child that satisfies it while still sharing state.
func TestStateSeparationPostHocScopeDoesNotClaimPrevention(t *testing.T) {
	for _, scope := range []string{
		stateSeparationChildStatementScope,
		stateSeparationPostHocScope,
	} {
		lower := strings.ToLower(scope)
		for _, forbidden := range []string{"prevent", "impossible", "prohibit", "denied"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("scope %q claims prevention; only the host boundary can show that", scope)
			}
		}
	}
}

// Three bases are named so the reserved one cannot be emitted by accident and then
// read as if it had been observed.
func TestStateSeparationBasesAreDistinct(t *testing.T) {
	bases := []string{
		stateSeparationChildStatement,
		stateSeparationEntryPostHoc,
		stateSeparationHostBoundary,
	}
	seen := map[string]bool{}
	for _, b := range bases {
		if seen[b] {
			t.Fatalf("duplicate basis %q", b)
		}
		seen[b] = true
	}
	if !seen["host-boundary"] {
		t.Fatal("host-boundary must stay named as the reserved basis")
	}
}
