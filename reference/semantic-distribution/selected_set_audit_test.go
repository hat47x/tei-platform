package semanticdistribution

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestSelectedNamespaceSetAuditDoesNotExpandRealWorkspaceImports(t *testing.T) {
	consumerRoot, providerRoot := preparePair(t)
	writeFixture(t, filepath.Join(providerRoot, "semantic", "common.tei"),
		"namespace common\n\ntype MachineId from std.identifier\n")
	writeFixture(t, filepath.Join(providerRoot, "semantic", "os", "process.tei"),
		"namespace os\n\ntype ProcessId from common.machine_id\n\ndata Process {\n    attribute processId type os.process_id\n}\n")

	consumer, err := workspace.Load(consumerRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(consumer.Imports) != 1 {
		t.Fatalf("expected only explicitly imported namespace: %+v", consumer.Imports)
	}
	if !containsDiagnostic(validation.Validate(consumer.World), "UNKNOWN_SEMANTIC_TYPE_UNDERLYING") {
		t.Fatal("without common the original world must still fail validation")
	}
	os := findNamespace(t, consumer.World.Semantics, "os")
	incomplete, err := AuditSelectedNamespaceSet([]semantic.Namespace{os})
	if err != nil {
		t.Fatal(err)
	}
	if len(incomplete.Dependencies) != 0 || len(incomplete.CycleCandidates) != 0 ||
		len(incomplete.Unmatched) != 1 || incomplete.Unmatched[0].From != "os" ||
		incomplete.Unmatched[0].Reference.TargetID != "common.machine_id" {
		t.Fatalf("unselected dependency must stay unmatched: %+v", incomplete)
	}

	provider, err := workspace.Load(providerRoot)
	if err != nil {
		t.Fatal(err)
	}
	common := findNamespace(t, provider.World.Semantics, "common")
	complete, err := AuditSelectedNamespaceSet([]semantic.Namespace{os, common})
	if err != nil {
		t.Fatal(err)
	}
	if len(complete.Unmatched) != 0 || len(complete.CycleCandidates) != 0 ||
		len(complete.Dependencies) != 1 {
		t.Fatalf("expected exactly one explicit semantic dependency: %+v", complete)
	}
	edge := complete.Dependencies[0]
	if edge.From != "os" || edge.To != "common" || len(edge.References) != 1 ||
		edge.References[0].SourceKind != "semantic-type" ||
		edge.References[0].TargetID != "common.machine_id" {
		t.Fatalf("unexpected edge: %+v", edge)
	}
	reversed, err := AuditSelectedNamespaceSet([]semantic.Namespace{common, os})
	if err != nil || !reflect.DeepEqual(complete, reversed) {
		t.Fatalf("selection order changed report: %+v vs %+v; err %v", complete, reversed, err)
	}
	// The report did not mutate the current consumer import set or its
	// canonical world, and never loaded common on the consumer's behalf.
	if len(consumer.Imports) != 1 ||
		!containsDiagnostic(validation.Validate(consumer.World), "UNKNOWN_SEMANTIC_TYPE_UNDERLYING") {
		t.Fatal("dependency report silently changed the consumer's meaning")
	}
}

func TestSelectedNamespaceSetAuditFindsStronglyConnectedGroupsNotUnrelatedDependencies(t *testing.T) {
	makeRelation := func(ns, target string) semantic.Namespace {
		return semantic.Namespace{
			ID: semanticID(ns),
			Data: []semantic.Data{{
				ID: semanticID(ns + ".item"),
				Relations: []semantic.Relation{{
					ID: semanticID(ns + ".ref"), Target: semanticID(target + ".item"),
				}},
			}},
		}
	}
	chosen := []semantic.Namespace{
		makeRelation("a", "b"),
		makeRelation("b", "a"),
		makeRelation("c", "d"),
		makeRelation("d", "e"),
		makeRelation("e", "c"),
		makeRelation("f", "g"),
		{ID: "g"},
		{ID: "unused"},
	}
	observed, err := AuditSelectedNamespaceSet(chosen)
	if err != nil {
		t.Fatal(err)
	}
	want := []NamespaceCycleCandidate{
		{Namespaces: []string{"a", "b"}},
		{Namespaces: []string{"c", "d", "e"}},
	}
	if !reflect.DeepEqual(observed.CycleCandidates, want) {
		t.Fatalf("incorrect strongly connected groups: got %+v want %+v", observed.CycleCandidates, want)
	}
	if len(observed.Dependencies) != 6 || len(observed.Unmatched) != 0 {
		t.Fatalf("unexpected dependency set: %+v", observed)
	}
	// A cycle of relations is a review candidate, not an unconditional
	// failure to load, validate or accept business semantics.
	reversed := append([]semantic.Namespace(nil), chosen...)
	for l, r := 0, len(reversed)-1; l < r; l, r = l+1, r-1 {
		reversed[l], reversed[r] = reversed[r], reversed[l]
	}
	again, err := AuditSelectedNamespaceSet(reversed)
	if err != nil || !reflect.DeepEqual(observed, again) {
		t.Fatalf("report depended on input order: %+v, %+v, %v", observed, again, err)
	}
}

func TestSelectedNamespaceSetAuditRejectsDuplicateOrMalformedScope(t *testing.T) {
	tests := []struct {
		name string
		set  []semantic.Namespace
		want string
	}{
		{"duplicate-identical", []semantic.Namespace{{ID: "os"}, {ID: "os"}}, "conflicting"},
		{"duplicate-different", []semantic.Namespace{
			{ID: "os", Types: []semantic.Type{{ID: "os.id", Underlying: "std.identifier"}}},
			{ID: "os", Types: []semantic.Type{{ID: "os.id", Underlying: "std.text"}}},
		}, "conflicting"},
		{"invalid-id", []semantic.Namespace{{ID: "Invalid-ID"}}, "invalid"},
		{"whitespace-id", []semantic.Namespace{{ID: " os "}}, "invalid"},
		{"std-is-prelude", []semantic.Namespace{{ID: "std"}}, "std prelude"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := AuditSelectedNamespaceSet(tc.set); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("must fail closed on ambiguous selection: %v", err)
			}
		})
	}
}

func TestSelectedNamespaceSetAuditKeepsNestedScopesAndUnknownReferences(t *testing.T) {
	os := semantic.Namespace{
		ID: "os",
		Types: []semantic.Type{
			{ID: "os.x", Underlying: "common.sub.identifier"},
			{ID: "os.y", Underlying: "unknown.maybe_type"},
			{ID: "os.z", Underlying: "std.text"},
		},
	}
	result, err := AuditSelectedNamespaceSet([]semantic.Namespace{
		{ID: "common"}, os, {ID: "common.sub"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Dependencies) != 1 ||
		result.Dependencies[0].To != "common.sub" ||
		result.Dependencies[0].From != "os" ||
		len(result.Unmatched) != 1 ||
		result.Unmatched[0].Reference.TargetID != "unknown.maybe_type" {
		t.Fatalf("nested scope or unknown reference was incorrectly assigned: %+v", result)
	}
	if len(result.Selected) != 3 || len(result.CycleCandidates) != 0 {
		t.Fatalf("unexpected extra selection or inferred cycle: %+v", result)
	}
}

func semanticID(s string) core.CanonicalID {
	return core.CanonicalID(s)
}
