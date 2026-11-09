package buildintent_test

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/buildintent"
	"tei/internal/core"
	"tei/internal/plugin"
)

// These tests pin the resolver where it now lives (SB-C0 §4.13, case I-1b). The
// reference tool's own tests still cover the end-to-end receipt against the
// fixture catalog; these prove the resolver itself works from internal/, which is
// what lets a caller derive a plan without starting a subprocess.
//
// A resolved receipt is a derived cache. Nothing here accepts anything, and the
// boundary string in the receipt says so.

const definitionSource = `plugin tei.plugin.resolve_probe v1.0.0 {
    runtime builtin
    provides deployment-executor runtime.resolve_probe {
        action application-start
    }
}
`

type fakeSources struct{}

func (fakeSources) Resolve(source buildintent.IntentSource) (buildintent.ResolvedSource, error) {
	if source.ID != "definition" {
		return buildintent.ResolvedSource{}, errNotFound
	}
	files := map[string][]byte{".": []byte(definitionSource)}
	return buildintent.ResolvedSource{
		Source: plugin.BuildSource{
			Kind:    source.Kind,
			ID:      source.ID,
			Locator: source.Ref,
			Digest:  "sha256:" + strings.Repeat("a", 64),
		},
		Files: files,
	}, nil
}

type fakeBuilders struct{}

func (fakeBuilders) Resolve(req buildintent.BuilderRequest) (buildintent.BuilderResolution, error) {
	return buildintent.BuilderResolution{
		Identity: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.resolve_probe"),
			Version:         "1.0.0",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  "sha256:" + strings.Repeat("b", 64),
		},
		Implementation: "probe",
	}, nil
}

func (fakeBuilders) CatalogFingerprint() (string, error) { return "catalog.builders", nil }

type fakeDependencies struct{}

func (fakeDependencies) Resolve(req buildintent.DependencyRequirement) (plugin.BuildDependency, error) {
	return plugin.BuildDependency{
		Kind:           req.Kind,
		ID:             req.ID,
		ArtifactDigest: "sha256:" + strings.Repeat("c", 64),
	}, nil
}

func (fakeDependencies) CatalogFingerprint() (string, error) { return "catalog.dependencies", nil }

type fakeTargets struct{}

func (fakeTargets) Check(target, profile string) error { return nil }

func (fakeTargets) CatalogFingerprint() (string, error) { return "catalog.targets", nil }

var errNotFound = notFoundError{}

type notFoundError struct{}

func (notFoundError) Error() string { return "source not found" }

const probeDeclaration = `{
  "formatVersion": 1,
  "subject": {"kind": "plugin", "id": "tei.plugin.resolve_probe", "version": "1.0.0", "definitionSource": "definition"},
  "sources": [{"kind": "plugin-definition", "id": "definition", "ref": "plugin.tei"}],
  "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
  "recipe": {"kind": "go-package", "entrypoint": "plugin.tei"},
  "outputKind": "plugin-package",
  "target": "host/linux/amd64",
  "profile": "release"
}`

func probeContext() buildintent.ResolutionContext {
	return buildintent.ResolutionContext{
		Sources:      fakeSources{},
		Builders:     fakeBuilders{},
		Dependencies: fakeDependencies{},
		Targets:      fakeTargets{},
	}
}

// The resolver derives a plan from internal/ and marks the receipt as a derived
// cache, not a canonical source.
func TestResolveDerivesAPlanFromInternal(t *testing.T) {
	intent, err := buildintent.ParseIntent([]byte(probeDeclaration))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	receipt, err := buildintent.Resolve(intent, probeContext())
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if receipt.Authority != buildintent.AuthorityDerivedCache {
		t.Fatalf("receipt authority is %q, want %q", receipt.Authority, buildintent.AuthorityDerivedCache)
	}
	if receipt.Plan.Plugin.ID != intent.Subject.ID {
		t.Fatalf("plan plugin %q, want %q", receipt.Plan.Plugin.ID, intent.Subject.ID)
	}
	if receipt.PlanFingerprint == "" || receipt.IntentFingerprint == "" {
		t.Fatal("the receipt carries no fingerprints")
	}
	if receipt.PlanFingerprint == receipt.IntentFingerprint {
		t.Fatal("the plan fingerprint and the intent fingerprint must be different values")
	}
	if receipt.Plan.OutputKind != plugin.BuildOutputPluginPackage {
		t.Fatalf("plan output kind %q", receipt.Plan.OutputKind)
	}
	if receipt.SourceClosureFingerprint == "" || receipt.ResolutionContextFingerprint == "" {
		t.Fatal("the receipt is missing context fingerprints")
	}
}

// The same intent and the same context produce the same receipt, so a caller can
// compare two derivations instead of trusting one.
func TestResolveIsDeterministic(t *testing.T) {
	intent, err := buildintent.ParseIntent([]byte(probeDeclaration))
	if err != nil {
		t.Fatal(err)
	}
	first, err := buildintent.Resolve(intent, probeContext())
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildintent.Resolve(intent, probeContext())
	if err != nil {
		t.Fatal(err)
	}
	if first.PlanFingerprint != second.PlanFingerprint {
		t.Fatalf("two derivations disagree: %s vs %s", first.PlanFingerprint, second.PlanFingerprint)
	}
	if first.ResolutionContextFingerprint != second.ResolutionContextFingerprint {
		t.Fatal("the context fingerprint is not stable")
	}

	// MarshalReceipt is byte-stable, which is what VerifyDerivedReceipt relies on.
	a, err := buildintent.MarshalReceipt(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := buildintent.MarshalReceipt(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatal("the marshalled receipt is not stable")
	}
	if err := buildintent.VerifyDerivedReceipt(intent, probeContext(), a); err != nil {
		t.Fatalf("a freshly derived receipt did not verify: %v", err)
	}
	// A stored receipt that has been edited is refused.
	var tampered map[string]any
	if err := json.Unmarshal(a, &tampered); err != nil {
		t.Fatal(err)
	}
	tampered["planFingerprint"] = "sha256:not-the-plan"
	edited, err := json.Marshal(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if err := buildintent.VerifyDerivedReceipt(intent, probeContext(), edited); err == nil {
		t.Fatal("an edited receipt verified")
	}
}

// An incomplete context is refused instead of resolving against half the world.
func TestResolveRefusesAnIncompleteContext(t *testing.T) {
	intent, err := buildintent.ParseIntent([]byte(probeDeclaration))
	if err != nil {
		t.Fatal(err)
	}
	ctx := probeContext()
	ctx.Targets = nil
	if _, err := buildintent.Resolve(intent, ctx); err == nil {
		t.Fatal("a context without a target catalog resolved")
	}
}
