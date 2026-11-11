package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTempRepo builds a throwaway repository layout holding the P1-B sample
// and the minimum workspace material each check reads.
func writeTempRepo(t *testing.T, sample string) string {
	t.Helper()
	root := t.TempDir()

	samplePath := filepath.Join(root, filepath.FromSlash(repoRelativeSamplePath))
	if err := os.MkdirAll(filepath.Dir(samplePath), 0o755); err != nil {
		t.Fatalf("mkdir sample dir: %v", err)
	}
	if err := os.WriteFile(samplePath, []byte(sample), 0o644); err != nil {
		t.Fatalf("write sample: %v", err)
	}

	workspace := filepath.Join(root, filepath.FromSlash(repoRelativeWorkspace))
	for _, dir := range []string{"semantic/sales", "semantic/customer"} {
		if err := os.MkdirAll(filepath.Join(workspace, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatalf("mkdir workspace dir: %v", err)
		}
	}
	manifest := "workspace: temp\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n  - databases\n"
	if err := os.WriteFile(filepath.Join(workspace, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	order := "namespace sales\n\ndata Order {\n}\n\ndata OrderLine {\n}\n"
	if err := os.WriteFile(filepath.Join(workspace, "semantic", "sales", "order.tei"), []byte(order), 0o644); err != nil {
		t.Fatalf("write sales source: %v", err)
	}
	customer := "namespace customer\n\ndata Customer {\n}\n"
	if err := os.WriteFile(filepath.Join(workspace, "semantic", "customer", "customer.tei"), []byte(customer), 0o644); err != nil {
		t.Fatalf("write customer source: %v", err)
	}

	return root
}

// loadRealSample returns the committed P1-B sample text.
func loadRealSample(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(repoRelativeSamplePath)))
	if err != nil {
		t.Fatalf("read committed sample: %v", err)
	}
	return string(b)
}

func indexByID(results []result) map[string]result {
	m := map[string]result{}
	for _, r := range results {
		m[r.ID] = r
	}
	return m
}

// TestP1BAcceptancePassesOnCommittedSample is the positive control: the
// committed sample must satisfy every P1-B condition.
func TestP1BAcceptancePassesOnCommittedSample(t *testing.T) {
	root := writeTempRepo(t, loadRealSample(t))
	results := checkP1B(root)
	for _, r := range results {
		if !r.OK {
			t.Errorf("%s failed on the committed sample: %s", r.ID, r.Detail)
		}
	}
}

// TestP1BNegativeControls proves each check can fail. A check that cannot fail
// would make the P1-B PASS meaningless.
func TestP1BNegativeControls(t *testing.T) {
	sample := loadRealSample(t)

	t.Run("B-1 rejects a sample inside a declared source root", func(t *testing.T) {
		root := writeTempRepo(t, sample)

		// Relocate the sample under a declared root (semantic/).
		insidePath := filepath.Join(root, filepath.FromSlash(repoRelativeWorkspace), "semantic", "order_entry.tei.md")
		if err := os.WriteFile(insidePath, []byte(sample), 0o644); err != nil {
			t.Fatalf("write relocated sample: %v", err)
		}

		r := sampleOutsideSourceRoots(root, insidePath)
		if r.OK {
			t.Fatalf("B-1 accepted a sample placed inside a declared source root: %s", r.Detail)
		}
	})

	t.Run("B-2 rejects CRLF and broken fences", func(t *testing.T) {
		// Drop one opening fence so the block count becomes odd.
		broken := strings.Replace(sample, "```mermaid", "", 1)
		root := writeTempRepo(t, broken)
		r := indexByID(checkP1B(root))["B-2"]
		if r.OK {
			t.Fatalf("B-2 accepted a sample with an unterminated fence: %s", r.Detail)
		}

		crlfSample := strings.ReplaceAll(sample, "\n", "\r\n")
		crlfRoot := writeTempRepo(t, crlfSample)
		r = indexByID(checkP1B(crlfRoot))["B-2"]
		if r.OK {
			t.Fatalf("B-2 accepted a sample with CRLF endings: %s", r.Detail)
		}
	})

	t.Run("B-3 rejects an undeclared subject", func(t *testing.T) {
		unknown := strings.Replace(sample, "`sales.order_line`", "`sales.unmodelled_thing`", 1)
		root := writeTempRepo(t, unknown)
		r := indexByID(checkP1B(root))["B-3"]
		if r.OK {
			t.Fatalf("B-3 accepted an undeclared subject: %s", r.Detail)
		}
	})

	t.Run("B-4 rejects a sample without the raw-draft boundary", func(t *testing.T) {
		stripped := strings.Replace(sample, "Raw Draft", "document", 1)
		stripped = strings.Replace(stripped, "実Source消費", "利用", 1)
		root := writeTempRepo(t, stripped)
		r := indexByID(checkP1B(root))["B-4"]
		if r.OK {
			t.Fatalf("B-4 accepted a sample without the raw-draft boundary: %s", r.Detail)
		}
	})

	t.Run("B-5 rejects a sample not declared a draft", func(t *testing.T) {
		stripped := strings.Replace(sample, "status: draft", "status: reviewed", 1)
		root := writeTempRepo(t, stripped)
		r := indexByID(checkP1B(root))["B-5"]
		if r.OK {
			t.Fatalf("B-5 accepted a non-draft sample: %s", r.Detail)
		}
	})
}
