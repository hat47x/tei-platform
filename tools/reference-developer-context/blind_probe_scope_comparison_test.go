// A fixed-budget and blind-world control for choosing where to probe next.
// This is a read-only test fixture, not the real SEI Method or Product Tool.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type scopeReadReceipt struct {
	Scope     string
	Path      string
	SHA256    string
	ByteCount int
	Attempts  int
	Match     bool
	State     string
}

// A test-only read-only tool with a deliberately restricted scope roster.
// enabled is an artificial capability switch, NOT a Host permission grant.
// Neither hidden world identity nor the sealed expected location is an input.
func readOneFixedScope(root, target, scope string, enabled bool) (scopeReadReceipt, error) {
	result := scopeReadReceipt{Scope: scope, State: "unsupported"}
	if !enabled {
		return result, nil
	}
	if scope != "docs" && scope != "plan/adr" {
		return result, fmt.Errorf("scope is not in the fixture tool roster: %q", scope)
	}
	dir := filepath.Join(root, filepath.FromSlash(scope))
	info, err := os.Lstat(dir)
	if err != nil {
		return result, err
	}
	if !info.IsDir() {
		return result, fmt.Errorf("scope is not a regular directory: %q", scope)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return result, err
	}
	result.State = "no-matching-file-in-scanned-scope"
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		rel := filepath.ToSlash(filepath.Join(scope, entry.Name()))
		result.Attempts++
		body, err := readInside(root, rel)
		if err != nil {
			// Name the cause: "read-failed" alone cannot be acted on when the
			// host's path handling differs (#3783).
			result.State = "read-failed: " + err.Error()
			return result, nil
		}
		result.Path = rel
		result.SHA256 = digest(body)
		result.ByteCount = len(body)
		result.Match = strings.Contains(string(body), target)
		result.State = "read-observed"
		return result, nil // exactly one file attempt per scope
	}
	return result, nil
}

func blindScopeWorld(t *testing.T, target, targetScope string) string {
	t.Helper()
	root := t.TempDir()
	writeCoverageFixture(t, root, target, "package validation\n")
	for i := 0; i < maxCandidates; i++ {
		writeCoverageFixture(t, root,
			fmt.Sprintf("internal/validation/a%02d_test.go", i),
			"package validation\n")
	}
	// Equal file lists, equal lengths and equal read budgets in both worlds.
	// Only the byte CONTENT assigning the exact target citation is swapped.
	other := "internal/validation/assessment.go"
	if len(other) != len(target) || other == target {
		t.Fatal("balanced unrelated source path must have the same length")
	}
	for _, scope := range []string{"docs", "plan/adr"} {
		reference := other
		if scope == targetScope {
			reference = target
		}
		writeCoverageFixture(t, root,
			filepath.ToSlash(filepath.Join(scope, "candidate.md")),
			"# Citation\n"+reference+"\n")
	}
	return root
}

func TestBlindProbeSameReportCompetingScopesOneReadBound(t *testing.T) {
	target := "internal/validation/validation.go"
	wDocs := blindScopeWorld(t, target, "docs")
	wADR := blindScopeWorld(t, target, "plan/adr")
	beforeDocs, err := Discover(wDocs, target)
	if err != nil {
		t.Fatal(err)
	}
	beforeADR, err := Discover(wADR, target)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforeDocs, beforeADR) ||
		len(beforeDocs.Candidates) != maxCandidates {
		t.Fatal("blind worlds must have indistinguishable initial Reports")
	}
	// Fixed, equal-budget scope choices; neither sees the hidden world's
	// target citation. Each choice wins exactly one of two balanced worlds.
	worlds := []struct {
		root       string
		truthScope string // test-side oracle ONLY
	}{
		{wDocs, "docs"},
		{wADR, "plan/adr"},
	}
	for _, selected := range []string{"docs", "plan/adr"} {
		matches := 0
		for _, world := range worlds {
			out, err := readOneFixedScope(world.root, target, selected, true)
			if err != nil {
				t.Fatal(err)
			}
			if out.State != "read-observed" || out.Attempts != 1 ||
				out.Path != filepath.ToSlash(filepath.Join(selected, "candidate.md")) ||
				out.SHA256 == "" || out.ByteCount == 0 ||
				out.Match != (selected == world.truthScope) {
				t.Fatalf("read observation does not match fixed independent oracle: %+v", out)
			}
			if out.Match {
				matches++
			}
		}
		if matches != 1 {
			t.Fatalf("%s: one blind fixed read should match one of two worlds, got %d", selected, matches)
		}
	}
	// Under the stipulated balanced two-world toy distribution, each
	// fixed one-scope policy reaches 1/2. No real model was run.
}

func TestBlindProbeAdditionalReadResolvesMissAtHigherCost(t *testing.T) {
	target := "internal/validation/validation.go"
	worlds := []struct {
		root       string
		needSecond bool
	}{
		{blindScopeWorld(t, target, "docs"), false},
		{blindScopeWorld(t, target, "plan/adr"), true},
	}
	totalReads := 0
	for _, world := range worlds {
		one, err := readOneFixedScope(world.root, target, "docs", true)
		if err != nil || one.Attempts != 1 || one.Match == world.needSecond {
			t.Fatalf("unexpected fixed first read: %+v err=%v", one, err)
		}
		totalReads += one.Attempts
		if !one.Match {
			two, err := readOneFixedScope(world.root, target, "plan/adr", true)
			if err != nil || two.Attempts != 1 || !two.Match {
				t.Fatalf("follow-up read failed to reach target: %+v err=%v", two, err)
			}
			totalReads += two.Attempts
		}
	}
	if totalReads != 3 {
		t.Fatalf("two-world adaptive approach needs three actual file reads, got %d", totalReads)
	}
	// An extra read is a cost, not a model's superior reasoning. The
	// test-only tool cannot establish any external Host or user grant.
	root := blindScopeWorld(t, target, "docs")
	unsupported, err := readOneFixedScope(root, target, "docs", false)
	if err != nil || unsupported.State != "unsupported" ||
		unsupported.Attempts != 0 || unsupported.SHA256 != "" || unsupported.Match {
		t.Fatalf("unavailable tool must not claim read evidence: %+v err=%v", unsupported, err)
	}
}
