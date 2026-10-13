package semanticdistribution

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestAuditNamespaceDependenciesUsesRealWorkspaceButDoesNotSelectMissingProvider(t *testing.T) {
	consumer, providerPath := preparePair(t)
	writeFixture(t, filepath.Join(providerPath, "semantic", "common.tei"),
		"namespace common\n\ntype MachineId from std.identifier\n")
	writeFixture(t, filepath.Join(providerPath, "semantic", "os", "process.tei"),
		"namespace os\n\ntype ProcessId from common.machine_id\n\ndata Process {\n    attribute processId type os.process_id\n}\n")
	loaded, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Imports) != 1 {
		t.Fatalf("no automatic import closure should appear: %+v", loaded.Imports)
	}
	if !containsDiagnostic(validation.Validate(loaded.World), "UNKNOWN_SEMANTIC_TYPE_UNDERLYING") {
		t.Fatal("original consumer must still reject absent common namespace")
	}
	osNS := findNamespace(t, loaded.World.Semantics, "os")
	noCommon, err := AuditNamespaceDependencies(osNS, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := NamespaceDependencyUse{
		SourceKind: "semantic-type", SourceID: "os.process_id",
		Field: "underlying", TargetID: "common.machine_id",
	}
	if len(noCommon.External) != 0 || !reflect.DeepEqual(noCommon.Unmatched, []NamespaceDependencyUse{want}) {
		t.Fatalf("unselected namespace should remain unmatched: %+v", noCommon)
	}
	provider, err := workspace.Load(providerPath)
	if err != nil {
		t.Fatal(err)
	}
	common := findNamespace(t, provider.World.Semantics, "common")
	withCommon, err := AuditNamespaceDependencies(osNS, []semantic.Namespace{common})
	if err != nil {
		t.Fatal(err)
	}
	want.Namespace = "common"
	if len(withCommon.Unmatched) != 0 || !reflect.DeepEqual(withCommon.External, []NamespaceDependencyUse{want}) {
		t.Fatalf("selected namespace should appear as a declared dependency: %+v", withCommon)
	}
	// This audit is purely a report. It must not silently change the consumer
	// import list, Canonical World, or model validation result.
	if len(loaded.Imports) != 1 || !containsDiagnostic(validation.Validate(loaded.World), "UNKNOWN_SEMANTIC_TYPE_UNDERLYING") {
		t.Fatal("audit implicitly accepted or resolved a provider")
	}
}

func TestAuditNamespaceDependenciesCoversSemanticReferencesAndKeepsNestedScope(t *testing.T) {
	ns := semantic.Namespace{
		ID: "os",
		Types: []semantic.Type{
			{ID: "os.process_id", Underlying: "common.machine_id"},
			{ID: "os.local_id", Underlying: "std.identifier"},
		},
		Data: []semantic.Data{{
			ID: "os.process",
			Attributes: []semantic.Attribute{
				{ID: "os.process.id", TypeRef: "common.sub.identifier"},
				{ID: "os.process.name", TypeRef: "missing.unknown"},
			},
			Relations: []semantic.Relation{{
				ID: "os.process.machine", Target: "common.machine",
				Keys: []semantic.RelationKey{{
					SourceAttribute: "os.process.id", TargetAttribute: "common.machine.id",
				}},
			}},
			CandidateKeys: []semantic.CandidateKey{{
				ID: "os.process.key", Attributes: []core.CanonicalID{"os.process.id"},
			}},
			FunctionalDependencies: []semantic.FunctionalDependency{{
				ID: "os.process.fd", Determinants: []core.CanonicalID{"os.process.id"},
				Dependents: []core.CanonicalID{"os.process.name"},
			}},
		}},
		Capabilities: []semantic.Capability{{
			ID:       "os.start",
			Inputs:   []semantic.Parameter{{Name: "machine", Type: "common.machine_id"}},
			Output:   "std.text",
			Outcomes: []semantic.Outcome{{Name: "started", Payload: "common.sub.result"}},
			Effect:   &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "common.machine"},
		}},
		Validations: []semantic.Validation{{
			ID: "os.valid", Attribute: "common.machine.id", Relation: "os.process.machine",
		}},
	}
	common := semantic.Namespace{ID: "common"}
	nested := semantic.Namespace{ID: "common.sub"}
	a, err := AuditNamespaceDependencies(ns, []semantic.Namespace{common, nested})
	if err != nil {
		t.Fatal(err)
	}
	b, err := AuditNamespaceDependencies(ns, []semantic.Namespace{nested, common})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("available order changed inventory: %+v / %+v", a, b)
	}
	if len(a.Unmatched) != 1 || a.Unmatched[0].TargetID != "missing.unknown" {
		t.Fatalf("unmatched reference not isolated: %+v", a.Unmatched)
	}
	// Every external declaration kind must be inventoried. This is a prefix
	// inventory only; symbol/kind existence belongs to canonical validation.
	fields := map[string]string{}
	for _, use := range a.External {
		fields[use.SourceKind+"/"+use.Field] = use.Namespace
	}
	for field, want := range map[string]string{
		"semantic-type/underlying":                  "common",
		"semantic-attribute/type":                   "common.sub",
		"semantic-relation/target":                  "common",
		"semantic-relation/keys[0].targetAttribute": "common",
		"semantic-capability/input:machine":         "common",
		"semantic-capability/outcome:started":       "common.sub",
		"semantic-capability/effect.data":           "common",
		"semantic-validation/attribute":             "common",
	} {
		if fields[field] != want {
			t.Fatalf("missing cross-namespace reference %s expected %s: %+v", field, want, a.External)
		}
	}
	for _, use := range a.External {
		if use.TargetID == "std.identifier" || use.TargetID == "std.text" || use.Namespace == "os" {
			t.Fatalf("local or standard reference became a provider requirement: %+v", use)
		}
	}
}

func TestAuditNamespaceDependenciesRejectsAmbiguousAvailableNamespaces(t *testing.T) {
	ns := semantic.Namespace{ID: "os"}
	if _, err := AuditNamespaceDependencies(ns, []semantic.Namespace{{ID: "common"}, {ID: "common"}}); err == nil {
		t.Fatal("duplicate available namespace must not silently win")
	}
	if _, err := AuditNamespaceDependencies(ns, []semantic.Namespace{{ID: "Invalid-ID"}}); err == nil {
		t.Fatal("invalid candidate namespace must not be accepted")
	}
	if _, err := AuditNamespaceDependencies(semantic.Namespace{ID: "Invalid-ID"}, nil); err == nil {
		t.Fatal("invalid audited namespace must not be accepted")
	}
}
