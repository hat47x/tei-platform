package inspectoradapter

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"tei/internal/composition"
	"tei/internal/operations"
	"tei/internal/sourceindex"
)

var testInspectorContract = composition.Contract{
	ID:      "tei.operations.inspect",
	Version: "0",
	Shape:   "bounded-reference-v0",
}

const inspectorArtifact = "sha256:" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func referenceInspectorRegistry(t *testing.T) (*composition.Registry, composition.Selection, composition.Binding) {
	t.Helper()
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, []composition.Provider{
		{ID: "inspector-A", Scope: "authoring-tool", Offers: testInspectorContract},
		{ID: "inspector-B", Scope: "authoring-tool", Offers: testInspectorContract},
		{ID: "runtime-only", Scope: "application-runtime", Offers: testInspectorContract},
	}, nil); err != nil {
		t.Fatal(err)
	}
	selection, err := registry.Select("inspector-B", "authoring-tool")
	if err != nil {
		t.Fatal(err)
	}
	binding := composition.Binding{
		ContractID:      testInspectorContract.ID,
		ContractVersion: testInspectorContract.Version,
		ProviderID:      selection.ProviderID,
		ArtifactDigest:  inspectorArtifact,
		Generation:      1,
	}
	return registry, selection, binding
}

func inspectorResponse(subject sourceindex.Subject) operations.SubjectInspection {
	return operations.SubjectInspection{
		SchemaVersion:          "tei.operations.inspect/v0",
		Effect:                 "read-only",
		Subject:                subject,
		Canonical:              json.RawMessage(`{"id":"sales.order"}`),
		WorldFingerprint:       "world-v1",
		SourceIndexFingerprint: "index-v1",
	}
}

func TestBindSelectedInvokesExactlyTheHostRegisteredProvider(t *testing.T) {
	registry, selection, binding := referenceInspectorRegistry(t)
	requested := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	calls := 0
	selected, err := Bind(registry, selection, binding, testInspectorContract,
		Registration{
			ProviderID:     "inspector-B",
			ArtifactDigest: inspectorArtifact,
			Inspect: func(got sourceindex.Subject) (operations.SubjectInspection, error) {
				calls++
				if got != requested {
					t.Fatalf("wrong subject: %+v", got)
				}
				return inspectorResponse(got), nil
			},
		}, func(composition.Binding) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	result, err := selected.Inspect(requested)
	if err != nil || result.Subject != requested || calls != 1 {
		t.Fatalf("selected-B invocation failed: calls=%d err=%v result=%+v", calls, err, result)
	}
}

func TestBindRejectsSelectedBRegisteredAWithIdenticalOutput(t *testing.T) {
	registry, selection, binding := referenceInspectorRegistry(t)
	calls := 0
	// Both functions would produce byte-identical responses. The trusted
	// registration ID rather than returned JSON must decide admission.
	wrongRegistration := Registration{
		ProviderID:     "inspector-A",
		ArtifactDigest: inspectorArtifact,
		Inspect: func(subject sourceindex.Subject) (operations.SubjectInspection, error) {
			calls++
			return inspectorResponse(subject), nil
		},
	}
	if _, err := Bind(registry, selection, binding, testInspectorContract, wrongRegistration,
		func(composition.Binding) error { return nil }); err == nil || calls != 0 {
		t.Fatalf("selected B incorrectly bound to A: err=%v calls=%d", err, calls)
	}
}

func TestBindRejectsWrongScopeAndContractAndArtifact(t *testing.T) {
	registry, selection, binding := referenceInspectorRegistry(t)
	registration := Registration{
		ProviderID:     "inspector-B",
		ArtifactDigest: inspectorArtifact,
		Inspect: func(subject sourceindex.Subject) (operations.SubjectInspection, error) {
			return inspectorResponse(subject), nil
		},
	}
	for _, test := range []struct {
		name      string
		selection composition.Selection
		binding   composition.Binding
		contract  composition.Contract
		reg       Registration
	}{
		{"different-scope", func() composition.Selection { x := selection; x.Scope = "application-runtime"; return x }(), binding, testInspectorContract, registration},
		{"contract-changed", selection, binding, composition.Contract{ID: "other", Version: "0", Shape: "other"}, registration},
		{"artifact-changed", selection, binding, testInspectorContract, func() Registration {
			x := registration
			x.ArtifactDigest = "sha256:" + strings.Repeat("b", 64)
			return x
		}()},
		{"implementation-missing", selection, binding, testInspectorContract, func() Registration { x := registration; x.Inspect = nil; return x }()},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Bind(registry, test.selection, test.binding, test.contract, test.reg,
				func(composition.Binding) error { return nil }); err == nil {
				t.Fatal("invalid Inspector binding accepted")
			}
		})
	}
}

func TestSelectedRechecksCurrentHostGrantAndRegistry(t *testing.T) {
	registry, selection, binding := referenceInspectorRegistry(t)
	requested := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	allowed := true
	calls := 0
	selected, err := Bind(registry, selection, binding, testInspectorContract,
		Registration{
			ProviderID:     selection.ProviderID,
			ArtifactDigest: inspectorArtifact,
			Inspect: func(s sourceindex.Subject) (operations.SubjectInspection, error) {
				calls++
				return inspectorResponse(s), nil
			},
		}, func(composition.Binding) error {
			if !allowed {
				return errors.New("host revoked")
			}
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	allowed = false
	if _, err := selected.Inspect(requested); err == nil || calls != 0 {
		t.Fatalf("revoked Host allowed Inspector: calls=%d err=%v", calls, err)
	}
	allowed = true
	if _, err := selected.Inspect(requested); err != nil || calls != 1 {
		t.Fatalf("Host grant did not permit selected Inspector: calls=%d err=%v", calls, err)
	}
	if _, err := registry.Apply(selection.Revision, nil, []string{"inspector-B"}); err != nil {
		t.Fatal(err)
	}
	if _, err := selected.Inspect(requested); !errors.Is(err, composition.ErrStaleRegistry) || calls != 1 {
		t.Fatalf("stale selection invoked Inspector: calls=%d err=%v", calls, err)
	}
	// Re-adding the same Provider ID does not revive an old selected epoch.
	if _, err := registry.Apply(selection.Revision+1, []composition.Provider{{
		ID: "inspector-B", Scope: "authoring-tool", Offers: testInspectorContract,
	}}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := selected.Inspect(requested); !errors.Is(err, composition.ErrStaleRegistry) || calls != 1 {
		t.Fatalf("re-registered stale selection invoked Inspector: calls=%d err=%v", calls, err)
	}
}

func TestSelectedRejectsProviderInvalidProjection(t *testing.T) {
	registry, selection, binding := referenceInspectorRegistry(t)
	selected, err := Bind(registry, selection, binding, testInspectorContract,
		Registration{
			ProviderID: selection.ProviderID, ArtifactDigest: inspectorArtifact,
			Inspect: func(s sourceindex.Subject) (operations.SubjectInspection, error) {
				v := inspectorResponse(s)
				v.Effect = "write"
				return v, nil
			},
		}, func(composition.Binding) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := selected.Inspect(sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}); err == nil {
		t.Fatal("selected Provider with invalid Effect was accepted")
	}
}

func TestBindDoesNotTreatUnrelatedRegistryEditAsStale(t *testing.T) {
	registry, selection, binding := referenceInspectorRegistry(t)
	selected, err := Bind(registry, selection, binding, testInspectorContract,
		Registration{
			ProviderID: selection.ProviderID, ArtifactDigest: inspectorArtifact,
			Inspect: func(s sourceindex.Subject) (operations.SubjectInspection, error) {
				return inspectorResponse(s), nil
			},
		}, func(composition.Binding) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Apply(selection.Revision, []composition.Provider{{
		ID: "another-scope", Scope: "application-runtime", Offers: testInspectorContract,
	}}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := selected.Inspect(sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}); err != nil {
		t.Fatalf("unrelated Provider caused false invalidation: %v", err)
	}
}
