package semanticdistribution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/modelref"
	"tei/internal/workspace"
)

func TestImportedMeaningImpactSurvivesSourceMoveAndFindsActualConsumer(t *testing.T) {
	consumer, provider := preparePair(t)
	writeFixture(t, filepath.Join(consumer, "semantic/watch.tei"),
		"namespace monitoring\n\ndata Watch {\n    attribute processId type os.process_id\n}\n\ndata Local {\n    attribute name type std.text\n}\n")
	before, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := ReviewImportedTypeImpact(before, before, "os")
	if err != nil || !unchanged.GraphValid || len(unchanged.Diagnostics) > 0 ||
		len(unchanged.Changes) != 0 || len(unchanged.Affected) != 0 {
		t.Fatalf("same meaning should have no candidate impacts: %+v, %v", unchanged, err)
	}
	from := filepath.Join(provider, "semantic/os/process.tei")
	to := filepath.Join(provider, "semantic/os/relocated.tei")
	if err := os.Rename(from, to); err != nil {
		t.Fatal(err)
	}
	relocated, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	move, err := ReviewImportedTypeImpact(before, relocated, "os")
	if err != nil || !move.GraphValid || len(move.Changes) != 0 || len(move.Affected) != 0 {
		t.Fatalf("moving the provider file must not change business meaning: %+v, %v", move, err)
	}
	writeFixture(t, to,
		"namespace os\n\ntype ProcessId from std.text\n\ndata Process {\n    attribute processId type os.process_id\n}\n")
	after, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	review, err := ReviewImportedTypeImpact(relocated, after, "os")
	if err != nil {
		t.Fatal(err)
	}
	if !review.GraphValid || len(review.Diagnostics) != 0 {
		t.Fatalf("valid changed models should yield candidate impacts: %+v", review)
	}
	changed := false
	for _, change := range review.Changes {
		if change.Kind == "evidence-changed" &&
			change.Key.Subject.Kind == "semantic-namespace" &&
			change.Key.Subject.ID == "os" {
			changed = true
		}
	}
	if !changed {
		t.Fatalf("selected semantic digest changed but no typed change was seen: %+v", review.Changes)
	}
	watch, local := false, false
	for _, impact := range review.Affected {
		if impact.Consumer.Subject.Kind == "semantic-data" && impact.Consumer.Subject.ID == "monitoring.watch" {
			watch = true
		}
		if impact.Consumer.Subject.ID == "monitoring.local" {
			local = true
		}
	}
	if !watch || local {
		t.Fatalf("changed import must surface known consumer only; watch=%v local=%v review=%+v", watch, local, review.Affected)
	}
	if len(review.AuthorityLimits) == 0 ||
		!strings.Contains(strings.Join(review.AuthorityLimits, ","), "unscanned-consumers") {
		t.Fatalf("the result must acknowledge unknown/unsupported consumer kinds: %+v", review.AuthorityLimits)
	}
}

func TestImportedMeaningImpactRejectsMissingOrContradictoryOrigin(t *testing.T) {
	consumer, _ := preparePair(t)
	before, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("missing-selected-provider", func(t *testing.T) {
		writeFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
			"workspace: consumer\nsources:\n  - semantic\n")
		without, err := workspace.Load(consumer)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ReviewImportedTypeImpact(before, without, "os"); err == nil || !strings.Contains(err.Error(), "not uniquely resolved") {
			t.Fatalf("lost provider import must not look unaffected: %v", err)
		}
	})
	t.Run("ambiguous-import", func(t *testing.T) {
		duplicate := before
		duplicate.Imports = append(append([]workspace.ResolvedImport{}, before.Imports...), before.Imports[0])
		if _, err := ReviewImportedTypeImpact(duplicate, before, "os"); err == nil || !strings.Contains(err.Error(), "not uniquely resolved") {
			t.Fatalf("duplicated origin must be rejected: %v", err)
		}
	})
	t.Run("forged-digest", func(t *testing.T) {
		stale := before
		stale.Imports = append([]workspace.ResolvedImport{}, before.Imports...)
		stale.Imports[0].SemanticDigest = "sha256:" + strings.Repeat("f", 64)
		if _, err := ReviewImportedTypeImpact(stale, before, "os"); err == nil || !strings.Contains(err.Error(), "stale or ambiguous semantic evidence") {
			t.Fatalf("out-of-sync import must not produce a valid graph: %v", err)
		}
	})
}

func TestImportedMeaningProjectionPreservesModelrefTypes(t *testing.T) {
	consumer, _ := preparePair(t)
	view, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ProjectImportTypeReferences(view, "os")
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Nodes) != 2 || len(graph.Edges) != 1 {
		t.Fatalf("expected provider Namespace and actual consumer Data: %+v", graph)
	}
	if diagnostics := modelref.Validate(graph, []modelref.Rule{ImportedTypeImpactRule}); len(diagnostics) != 0 {
		t.Fatalf("the projection must reuse the existing typed modelref graph: %+v", diagnostics)
	}
}
