// E33 ties the actual #1567 PR Source fingerprint to a purely illustrative
// Composition registry proposal. It does NOT publish a Service Contract,
// register a product Provider, run the plugin or grant Runtime authority.
package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"tei/internal/composition"
	handoff "tei/reference/composition-evidence-handoff"
)

type e33File struct {
	Path       string `json:"path"`
	Status     string `json:"status"`
	GitBlobSHA string `json:"gitBlobSha"`
}

type e33Snapshot struct {
	Kind      string    `json:"kind"`
	HeadSHA   string    `json:"headSha"`
	BaseSHA   string    `json:"baseSha"`
	Files     []e33File `json:"files"`
}

func e33ActualDraftSource(t *testing.T) []e33File {
	t.Helper()
	data, err := os.ReadFile("e32_project_plugin_pr1567_snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot e33Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Kind != "tei-e32-github-pr-files-evidence-v0" ||
		snapshot.HeadSHA != "be193266f1556e090b59cf2c237a23c0ca60236e" ||
		snapshot.BaseSHA != "aa63f482b35e9d169f150aaaac801d56db288d72" ||
		len(snapshot.Files) != 20 {
		t.Fatalf("E33 pinned PR evidence drifted: %+v", snapshot)
	}
	// The prior E32 GitHub observation is evidence of file identity, not a
	// trusted release artifact or #951 Provider publication.
	expected := map[string]string{
		"plugins/reference-project-customer-advice/advice.go": "0e1adb1dc471ff46daa7f608dd892eca1e3ac871",
		"tools/reference-project-advice-lit/main.go": "3cb210363d59c0052ed5da145d5895c504c7cef7",
		"tools/reference-project-advice-lit/order_entry_context.go": "6e05e787025d22895ba89fc664f76e9a58fc8ef3",
		"tools/reference-project-advice-lit/web/advice-element.mjs": "0f415f06f9e325b3d2f544cbbe7836f0c3f3e65f",
	}
	for _, f := range snapshot.Files {
		if f.Status != "added" ||
			strings.HasPrefix(f.Path, "internal/composition/") ||
			strings.HasPrefix(f.Path, "reference/workspace/") {
			t.Fatalf("E32 source snapshot was mistaken for product wiring: %+v", f)
		}
		if digest, tracked := expected[f.Path]; tracked {
			if f.GitBlobSHA != digest {
				t.Fatalf("pinned real source content changed: %+v", f)
			}
			delete(expected, f.Path)
		}
	}
	if len(expected) != 0 {
		t.Fatalf("missing actual Go/Lit Source paths: %+v", expected)
	}
	return snapshot.Files
}

func e33IllustrativeContract() composition.Contract {
	// Test-only structural shape, not an attested fingerprint or publication.
	// The actual #1567 code returns advisory JSON directly from a Go Host.
	return composition.Contract{
		ID:      "reference.project.customer-advice",
		Version: "test-only-v0",
		Shape:   "illustrative:(customerCode,note)->(kind,message,advisory)",
	}
}

func e33Requirement(scope, provider string) composition.WiringRequirement {
	return composition.WiringRequirement{
		Contract: e33IllustrativeContract(),
		Scope: scope, ProviderID: provider, SlotID: "project-advice",
	}
}

func TestE33RealProjectGoLitPRDoesNotDeclareAcceptedProvider(t *testing.T) {
	e33ActualDraftSource(t)
	registry := composition.NewRegistry()
	missing := registry.InspectWiring(e33Requirement("project-a", ""))
	if missing.Verdict != composition.ProposalUnknown ||
		missing.Reason != composition.ReasonNoKnownProvider ||
		missing.Selected != nil || len(missing.VerificationTargets) != 0 {
		t.Fatalf("real draft files were silently accepted as Provider: %+v", missing)
	}
	// The diagnostic Go Host's direct import is not evidence of a published
	// Service Contract, a Provider selection or current Host permission.
}

func TestE33SyntheticProviderStillNeedsIndependentContractEvidence(t *testing.T) {
	e33ActualDraftSource(t)
	registry := composition.NewRegistry()
	_, err := registry.Apply(0, []composition.Provider{{
		ID: "project-a-advice-candidate", Scope: "project-a",
		Offers: e33IllustrativeContract(),
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	set := registry.InspectWiringSet([]composition.WiringRequirement{
		e33Requirement("project-a", "project-a-advice-candidate"),
	})
	if set.Verdict != composition.ProposalCandidate ||
		len(set.Findings) != 1 || set.Findings[0].Selected == nil ||
		len(set.Findings[0].VerificationTargets) != 1 ||
		!registry.ProposalCurrent(set) {
		t.Fatalf("illustrative structural proposal absent: %+v", set)
	}
	report := handoff.ReconcileCurrent(registry, set, nil)
	if report.Blocked || len(report.Subjects) != 1 ||
		report.Subjects[0].State != handoff.ReferenceMissing {
		t.Fatalf("no #951 evidence was mistaken for a verified provider: %+v", report)
	}
}

func TestE33UntrustedLocatorCannotBecomeConformanceOrAuthority(t *testing.T) {
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, []composition.Provider{{
		ID: "project-a-advice-candidate", Scope: "project-a",
		Offers: e33IllustrativeContract(),
	}}, nil); err != nil {
		t.Fatal(err)
	}
	set := registry.InspectWiringSet([]composition.WiringRequirement{
		e33Requirement("project-a", "project-a-advice-candidate"),
	})
	// All of the following identifiers are fabricated. The existing
	// reconciler only links them; it intentionally does not authenticate them.
	ref := handoff.EvidenceReference{
		Subject: set.Findings[0].VerificationTargets[0],
		AuthorityScope: "application-execution",
		PluginID: "test-only-project-advice",
		PluginVersion: "0",
		ImplementationID: "test-only-Evaluate",
		ArtifactDigest: "sha256:" + strings.Repeat("a", 64),
		ContractFingerprint: "test-only-fingerprint",
		ConformanceProfile: "test-only",
		EvidenceFingerprint: "test-only-evidence",
		ResolutionFingerprint: "test-only-resolution",
	}
	linked := handoff.ReconcileCurrent(registry, set, []handoff.EvidenceReference{ref})
	if linked.Blocked || len(linked.Subjects) != 1 ||
		linked.Subjects[0].State != handoff.ReferenceLinked {
		t.Fatalf("expected only an explicitly unverified locator: %+v", linked)
	}
	// A linked locator is NOT a true #951 resolver result, published contract,
	// trusted artifact, Host guard or business Acceptance.
}

func TestE33AmbiguousAndCrossScopeProviderCannotBeSilentlyChosen(t *testing.T) {
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, []composition.Provider{
		{ID: "project-a-one", Scope: "project-a", Offers: e33IllustrativeContract()},
		{ID: "project-a-two", Scope: "project-a", Offers: e33IllustrativeContract()},
		{ID: "project-b-one", Scope: "project-b", Offers: e33IllustrativeContract()},
	}, nil); err != nil {
		t.Fatal(err)
	}
	ambiguous := registry.InspectWiring(e33Requirement("project-a", ""))
	if ambiguous.Verdict != composition.ProposalConflict ||
		ambiguous.Reason != composition.ReasonAmbiguousProviders ||
		ambiguous.Selected != nil {
		t.Fatalf("ambiguous project providers auto-selected: %+v", ambiguous)
	}
	cross := registry.InspectWiring(e33Requirement("project-a", "project-b-one"))
	if cross.Verdict != composition.ProposalUnknown ||
		cross.Selected != nil {
		t.Fatalf("foreign project Provider selected: %+v", cross)
	}
	pinned := registry.InspectWiring(e33Requirement("project-a", "project-a-one"))
	if pinned.Verdict != composition.ProposalCandidate ||
		pinned.Selected == nil || pinned.Selected.ProviderID != "project-a-one" {
		t.Fatalf("an exact explicit review candidate cannot be inspected: %+v", pinned)
	}
}

func TestE33ChangedDeclarationInvalidatesOldReviewWithoutAcceptingPlugin(t *testing.T) {
	registry := composition.NewRegistry()
	first, err := registry.Apply(0, []composition.Provider{{
		ID: "project-a-advice-candidate", Scope: "project-a",
		Offers: e33IllustrativeContract(),
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	set := registry.InspectWiringSet([]composition.WiringRequirement{
		e33Requirement("project-a", "project-a-advice-candidate"),
	})
	if !registry.ProposalCurrent(set) {
		t.Fatal("current test-only proposal was not present")
	}
	second, err := registry.Apply(first.Revision, []composition.Provider{{
		ID: "unrelated", Scope: "project-b", Offers: e33IllustrativeContract(),
	}}, nil)
	if err != nil || !registry.ProposalCurrent(set) {
		t.Fatalf("unrelated project invalidated valid candidate: %v", err)
	}
	_, err = registry.Apply(second.Revision, []composition.Provider{{
		ID: "project-a-advice-candidate", Scope: "project-a",
		Offers: composition.Contract{ID: e33IllustrativeContract().ID,
			Version: "test-only-v1", Shape: "changed"},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if registry.ProposalCurrent(set) {
		t.Fatal("changed local declaration kept old proposal current")
	}
	stale := handoff.ReconcileCurrent(registry, set, nil)
	if !stale.Blocked || len(stale.Subjects) != 0 {
		t.Fatalf("stale test-only provider still linked to current evidence: %+v", stale)
	}
}
