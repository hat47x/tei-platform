package okf

import (
	"path/filepath"
	"testing"
)

// writeTree writes files into dir. It fails if the tree cannot be created, so a
// fixture defect is never silently read as an empty bundle.
func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := osMkdirAll(filepath.Dir(path)); err != nil {
			t.Fatalf("create %s: %v", filepath.Dir(path), err)
		}
		writeFile(t, path, content)
	}
}

// TestStructuralBundleWithComputationAndLogs exercises the shapes the upstream
// sample bundles use: an Attested Computation whose sources, executor and attester
// are bundle-root paths, a deprecated concept, a usage_window, a nested index and a
// log. The fixture is synthetic and built in a temporary directory; no network read
// and no upstream content is involved.
func TestStructuralBundleWithComputationAndLogs(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"index.md": "# Subdirectories\n\n" +
			"* [tables](tables/index.md) - tables the bundle grounds against\n" +
			"* [computations](computations/index.md) - sanctioned SQL\n" +
			"* [metrics](metrics/index.md) - business definitions\n",
		"tables/index.md":           "# Table\n\n* [Orders](orders.md) - one row per order\n",
		"tables/orders.md":          "---\ntype: BigQuery Table\ntitle: Orders\nresource: https://example.test/tables/orders\ntags: [sales]\ngenerated: { by: agent/synthetic, at: 2026-06-30T14:00:00Z }\nverified:\n  - { by: human:reviewer, at: 2026-07-01T09:00:00Z }\nstatus: stable\nstale_after: 2026-12-31T00:00:00Z\nsources:\n  - id: schema\n    resource: tables/schema.md\n    title: Warehouse schema\n    author: team:data\n    usage_count: 1240\n    last_modified: 2026-06-15T00:00:00Z\nusage_window: { from: 2026-04-01T00:00:00Z, to: 2026-06-30T00:00:00Z }\n---\n\n# Schema\n\n| Column | Type |\n|---|---|\n| `order_id` | STRING |\n\nRevenue is recognised per policy.[^schema]\n\n[^schema]: Warehouse schema\n",
		"tables/schema.md":          "---\ntype: Reference\ntitle: Warehouse schema\n---\n\nschema notes\n",
		"computations/index.md":     "# Computation\n\n* [Revenue](revenue.md) - sanctioned SQL for revenue\n",
		"computations/revenue.md":   "---\ntype: Attested Computation\ntitle: Revenue for a fiscal year\nruntime: bigquery\nparameters:\n  - { name: year, type: integer, required: true }\nexecutor:\n  resource: skills/run-on-bq.md\n  receipt: [job_id, executed_sql, result]\nattester:\n  resource: attesters/sql_equality.py\ngenerated: { by: agent/synthetic, at: 2026-06-30T14:00:00Z }\nverified: { by: human:reviewer, at: 2026-07-01T09:00:00Z }\nstatus: stable\nsources:\n  - id: policy\n    resource: metrics/policy.md\n---\n\n# Computation\n\n```sql\nSELECT SUM(amount) AS revenue FROM orders WHERE year = @year\n```\n\nPer the policy.[^policy]\n\n[^policy]: policy\n",
		"skills/run-on-bq.md":       "---\ntype: Skill\ntitle: Run on BigQuery\n---\n\nrun instructions\n",
		"attesters/sql_equality.py": "attester placeholder text\n",
		"metrics/index.md":          "# Metric\n\n* [Revenue](revenue.md) - recognised revenue\n* [Legacy](legacy.md) - deprecated definition\n",
		"metrics/revenue.md":        "---\ntype: Metric\ntitle: Revenue\ntags: [finance, revenue]\ngenerated: { by: agent/synthetic, at: 2026-06-30T14:00:00Z }\nverified: { by: human:reviewer, at: 2026-07-01T09:00:00Z }\nstatus: stable\nsources:\n  - id: policy\n    resource: metrics/policy.md\n---\n\n# Definition\n\nRevenue links the computation at [revenue computation](/computations/revenue.md).\n\n[^policy]: policy\n",
		"metrics/policy.md":         "---\ntype: Policy\ntitle: Revenue recognition\nresource: https://example.test/policy\n---\n\npolicy text\n",
		"metrics/legacy.md":         "---\ntype: Metric\ntitle: Revenue (legacy)\ngenerated: { by: human:author, at: 2024-01-15T10:00:00Z }\nverified: { by: human:author, at: 2024-01-15T10:00:00Z }\nstatus: deprecated\n---\n\n# Deprecated\n\nRetired.\n\n# Why no attested computation\n\nThe SQL was deleted.\n",
		"log.md":                    "---\ntype: Log\ntitle: Bundle history\n---\n\n# Bundle history\n\n## 2026-07-01\n\n- **Verified** the bundle.\n\n## 2026-02-10\n\n- **Bootstrapped** the bundle.\n",
	})

	bundle, err := Read(dir, Options{})
	if err != nil {
		t.Fatalf("read the structural bundle: %v", err)
	}
	if !bundle.Format.Conformant {
		t.Fatalf("the structural bundle must be conformant; errors: %+v", bundle.Format.FormatErrors)
	}
	if len(bundle.Concepts) != 7 {
		t.Fatalf("concept count = %d, want 7: %v", len(bundle.Concepts), conceptIDs(bundle))
	}
	if len(bundle.Indexes) != 4 {
		t.Fatalf("index count = %d, want 4 (root, tables, computations, metrics)", len(bundle.Indexes))
	}
	if len(bundle.Logs) != 1 {
		t.Fatalf("log count = %d, want 1", len(bundle.Logs))
	}

	computation := findConcept(bundle, "computations/revenue")
	if computation == nil {
		t.Fatal("missing computations/revenue")
	}
	if computation.Runtime != "bigquery" {
		t.Fatalf("runtime = %q", computation.Runtime)
	}
	if len(computation.Parameters) != 1 || computation.Parameters[0].Name != "year" {
		t.Fatalf("parameters = %+v", computation.Parameters)
	}
	// Bundle-root form: "skills/run-on-bq.md" from "computations/" resolves only
	// against the root, which is how the upstream sample writes these fields.
	if computation.Executor == nil || computation.Executor.Resolution != ResolutionBundleRoot {
		t.Fatalf("executor = %+v, want a bundle-root resolution", computation.Executor)
	}
	if computation.Attester == nil || computation.Attester.Resolution != ResolutionBundleRoot {
		t.Fatalf("attester = %+v, want a bundle-root resolution", computation.Attester)
	}
	if !computation.HasComputationSection || computation.FencedLanguageCount != 1 {
		t.Fatalf("computation section=%t fences=%d", computation.HasComputationSection, computation.FencedLanguageCount)
	}

	orders := findConcept(bundle, "tables/orders")
	if orders == nil {
		t.Fatal("missing tables/orders")
	}
	if orders.ResourceResolution != ResolutionAbsoluteURL {
		t.Fatalf("resource resolution = %q, want absolute-url-not-fetched", orders.ResourceResolution)
	}
	if len(orders.Sources) != 1 || orders.Sources[0].ResourceKind != "bundle-path" {
		t.Fatalf("sources = %+v", orders.Sources)
	}
	if len(bundle.Concepts) > 0 && len(orders.Footnotes) != 1 {
		t.Fatalf("footnotes = %+v, want the one cited source", orders.Footnotes)
	}

	legacy := findConcept(bundle, "metrics/legacy")
	if legacy == nil {
		t.Fatal("missing metrics/legacy")
	}
	if legacy.Status != "deprecated" {
		t.Fatalf("legacy status = %q, want deprecated", legacy.Status)
	}
	if legacy.TrustTierLabel != "human-reviewed" {
		t.Fatalf("legacy tier = %q", legacy.TrustTierLabel)
	}
	if len(legacy.Sources) != 0 {
		t.Fatalf("a concept with no sources must report none; got %+v", legacy.Sources)
	}

	revenue := findConcept(bundle, "metrics/revenue")
	if revenue == nil {
		t.Fatal("missing metrics/revenue")
	}
	// The body link uses the bundle-root form, so it must resolve to the
	// computation concept rather than being reported broken.
	found := false
	for _, link := range revenue.Links {
		if link.Target == "/computations/revenue.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("links = %+v, want the bundle-root form recorded verbatim", revenue.Links)
	}
	if len(bundle.UnresolvedReferences) != 0 {
		t.Fatalf("every reference in this bundle resolves; unresolved = %+v", bundle.UnresolvedReferences)
	}

	log := bundle.Logs[0]
	if len(log.Entries) != 2 || log.Entries[0].Date != "2026-07-01" {
		t.Fatalf("log entries = %+v", log.Entries)
	}
	// A prose file that is not .md is not a concept; the attester placeholder is
	// on disk and was never read.
	for _, concept := range bundle.Concepts {
		if concept.Path == "attesters/sql_equality.py" {
			t.Fatal("a non-Markdown file must not be read as a concept")
		}
	}
}
