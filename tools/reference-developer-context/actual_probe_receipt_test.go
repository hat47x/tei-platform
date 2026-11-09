// Evaluation-only deterministic next-read witness. This is not a Product
// Tool, CLI or SEI Method implementation; authorization here is a synthetic
// fixture switch, NOT a real Host grant. Expected relevance stays test-side.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type localReadWitness struct {
	Path      string
	SHA256    string
	ByteCount int
	ExactPath bool
}

type localProbeObservation struct {
	State          string // unsupported, scope-missing, observed
	Attempts       int
	ReadSucceeded  int
	ReadFailed     int
	BudgetExceeded bool
	Reads          []localReadWitness
	// FirstReadError records why a read failed. Without it a failing read on a
	// host whose path handling differs (the Windows CI runner) reports only
	// "ReadFailed: N" and the cause has to be guessed (#3783).
	FirstReadError string
}

// The evaluator proposes "inspect docs/*.md" with an explicit fixed budget.
// Only after actual directory enumeration does it learn document filenames.
// readInside applies the current PoC's workspace boundary and size check.
// There is NO independent oracle, model result or accepted business meaning
// in this procedure. Neither the tool roster nor Host grant is real.
func localDocsProbe(root, target string, allowed bool, budget int) (localProbeObservation, error) {
	out := localProbeObservation{State: "unsupported"}
	if !allowed {
		return out, nil
	}
	if budget < 1 {
		return out, fmt.Errorf("positive read budget required")
	}
	docs := filepath.Join(root, "docs")
	info, err := os.Lstat(docs)
	if os.IsNotExist(err) {
		out.State = "scope-missing"
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if !info.IsDir() {
		return out, fmt.Errorf("docs scope must be a directory, not a symlink")
	}
	entries, err := os.ReadDir(docs)
	if err != nil {
		return out, err
	}
	out.State = "observed"
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		if out.Attempts >= budget {
			out.BudgetExceeded = true
			break
		}
		rel := filepath.ToSlash(filepath.Join("docs", entry.Name()))
		out.Attempts++
		body, readErr := readInside(root, rel)
		if readErr != nil {
			out.ReadFailed++
			if out.FirstReadError == "" {
				out.FirstReadError = rel + ": " + readErr.Error()
			}
			continue
		}
		out.ReadSucceeded++
		out.Reads = append(out.Reads, localReadWitness{
			Path: rel, SHA256: digest(body), ByteCount: len(body),
			ExactPath: strings.Contains(string(body), target),
		})
	}
	return out, nil
}

func localProbeHasMatchingWitness(p localProbeObservation) bool {
	for _, r := range p.Reads {
		if r.ExactPath {
			return true
		}
	}
	return false
}

func TestActualProbeD0VersusBoundedD1AndUnsupported(t *testing.T) {
	target := "internal/validation/validation.go"
	setup := func(docName, body string) string {
		t.Helper()
		root := t.TempDir()
		writeCoverageFixture(t, root, target, "package validation\n")
		for i := 0; i < maxCandidates; i++ {
			writeCoverageFixture(t, root,
				fmt.Sprintf("internal/validation/a%02d_test.go", i),
				"package validation\n")
		}
		if docName != "" {
			writeCoverageFixture(t, root, docName, body)
		}
		return root
	}
	w0 := setup("", "")
	w1 := setup("docs/reference.md", "# Target file\n"+target+"\n")
	w2 := setup("docs/reference.md", "# Different file\ninternal/validation/other.go\n")
	w3 := setup("docs/01-target.md", "# Target file\n"+target+"\n")
	writeCoverageFixture(t, w3, "docs/00-unrelated.md", "# Other file\n")

	// D0's complete Report JSON is indistinguishable in W0..W3: all
	// test slots are filled before docs are enumerated.
	worlds := []string{w0, w1, w2, w3}
	var d0 Report
	for i, root := range worlds {
		report, err := Discover(root, target)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			d0 = report
		} else if !reflect.DeepEqual(d0, report) {
			t.Fatalf("same initial D0, different result at world %d", i)
		}
	}

	// The synthetic unsupported state must not enumerate docs or read
	// files; it is not the same as a successful negative search.
	denied, err := localDocsProbe(w1, target, false, 1)
	if err != nil || denied.State != "unsupported" ||
		denied.Attempts != 0 || len(denied.Reads) != 0 {
		t.Fatalf("synthetic unavailability misrepresented: %+v %v", denied, err)
	}

	// D1's actual read receipts are observed by a test-only filesystem
	// operation. No hidden W truth or expected doc name is passed to D1.
	for i, root := range worlds {
		out, err := localDocsProbe(root, target, true, 1)
		if err != nil {
			t.Fatal(err)
		}
		switch i {
		case 0: // The docs scope is missing, not proof of global absence.
			if out.State != "scope-missing" || out.Attempts != 0 {
				t.Fatalf("W0 unexpected probe: %+v", out)
			}
		case 1: // One real read yields a SHA-bound matching witness.
			if out.State != "observed" || out.Attempts != 1 ||
				out.ReadSucceeded != 1 || out.BudgetExceeded ||
				!localProbeHasMatchingWitness(out) ||
				out.Reads[0].Path != "docs/reference.md" {
				t.Fatalf("W1 did not acquire matching bytes: %+v", out)
			}
			content, readErr := os.ReadFile(filepath.Join(root, "docs", "reference.md"))
			if readErr != nil || out.Reads[0].SHA256 != digest(content) ||
				out.Reads[0].ByteCount != len(content) {
				t.Fatalf("W1 read receipt not bound to observed bytes: %+v %v", out, readErr)
			}
		case 2: // Read succeeded but is not an exact target citation.
			if out.ReadSucceeded != 1 || localProbeHasMatchingWitness(out) ||
				out.ReadFailed != 0 {
				t.Fatalf("W2 irrelevant doc overclaimed: %+v", out)
			}
		case 3: // A later target doc remains unread under budget=1.
			if out.Attempts != 1 || out.ReadSucceeded != 1 ||
				!out.BudgetExceeded || localProbeHasMatchingWitness(out) ||
				out.Reads[0].Path != "docs/00-unrelated.md" {
				t.Fatalf("W3 truncated read misrepresented: %+v", out)
			}
			// Only the test's independent fixture oracle knows the
			// hidden second path, not D0 or the D1 read policy.
			if b, readErr := os.ReadFile(filepath.Join(root, "docs", "01-target.md")); readErr != nil || !strings.Contains(string(b), target) {
				t.Fatalf("W3 hidden fixture expected: %v", readErr)
			}
		}
	}
}

func TestActualProbeWitnessMustBeReacquiredAfterSourceRevision(t *testing.T) {
	target := "internal/validation/validation.go"
	root := t.TempDir()
	path := "docs/reference.md"
	writeCoverageFixture(t, root, target, "package validation\n")
	writeCoverageFixture(t, root, path, "# Target\n"+target+"\n")
	initial, err := localDocsProbe(root, target, true, 1)
	if err != nil || initial.ReadSucceeded != 1 || !localProbeHasMatchingWitness(initial) {
		t.Fatalf("initial read absent: %+v %v", initial, err)
	}
	writeCoverageFixture(t, root, path, "# Changed\ninternal/validation/other.go\n")
	changed, err := localDocsProbe(root, target, true, 1)
	if err != nil || changed.ReadSucceeded != 1 || localProbeHasMatchingWitness(changed) ||
		initial.Reads[0].SHA256 == changed.Reads[0].SHA256 {
		t.Fatalf("stale witness incorrectly reused: first=%+v second=%+v err=%v", initial, changed, err)
	}
	// Restoring original bytes makes the content digest equal again (ABA).
	// The same hash is not an independent witness of uninterrupted
	// history or of current consumer disclosure authority.
	writeCoverageFixture(t, root, path, "# Target\n"+target+"\n")
	restored, err := localDocsProbe(root, target, true, 1)
	if err != nil || !localProbeHasMatchingWitness(restored) ||
		restored.Reads[0].SHA256 != initial.Reads[0].SHA256 {
		t.Fatalf("ABA restoration unexpected: %+v err=%v", restored, err)
	}
}
