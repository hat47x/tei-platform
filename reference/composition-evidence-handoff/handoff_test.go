package handoff

import (
	"fmt"
	"strings"
	"testing"

	"tei/internal/composition"
)

var read = composition.Contract{ID: "test.read", Version: "1", Shape: "read-shape"}
var app = composition.Contract{ID: "test.app", Version: "1", Shape: "app-shape"}

func proposal(t *testing.T) (composition.WiringSetFinding, *composition.Registry) {
	t.Helper()
	r := composition.NewRegistry()
	_, err := r.Apply(0, []composition.Provider{
		{ID: "database", Scope: "tenant-a", Offers: read},
		{ID: "consumer", Scope: "tenant-a", Offers: app, Requires: []composition.Dependency{{Contract: read}}},
		{ID: "outsider", Scope: "tenant-b", Offers: read},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r.InspectWiringSet([]composition.WiringRequirement{{Contract: app, Scope: "tenant-a"}}), r
}
func ref(target composition.VerificationTarget) EvidenceReference {
	return EvidenceReference{
		Subject:        target,
		AuthorityScope: "application-runtime", PluginID: "std.reader", PluginVersion: "1.0.0", ImplementationID: "impl",
		ArtifactDigest: "sha256:" + strings.Repeat("a", 64), ContractFingerprint: "contract-fingerprint", ConformanceProfile: "reference-profile", EvidenceFingerprint: "evidence-id", ResolutionFingerprint: "resolution-id",
	}
}
func TestHandoffMissingAndAllExactReferences(t *testing.T) {
	set, _ := proposal(t)
	got := Reconcile(set, nil)
	if got.Blocked || len(got.Subjects) != 2 || got.Subjects[0].Target.ProviderID != "consumer" || got.Subjects[1].Target.ProviderID != "database" {
		t.Fatalf("wrong deduplicated order %+v", got)
	}
	for _, subject := range got.Subjects {
		if subject.State != ReferenceMissing || subject.Reference != nil {
			t.Fatalf("missing reference accepted %+v", subject)
		}
	}
	refs := []EvidenceReference{ref(got.Subjects[0].Target), ref(got.Subjects[1].Target)}
	linked := Reconcile(set, refs)
	for i, subject := range linked.Subjects {
		if subject.State != ReferenceLinked || subject.Reference == nil {
			t.Fatalf("%d: %+v", i, subject)
		}
	}
	refs[0].PluginID = "tampered"
	if linked.Subjects[0].Reference.PluginID == "tampered" {
		t.Fatal("result reference aliases input")
	}
}
func TestHandoffStaleMismatchIncompleteAndAmbiguous(t *testing.T) {
	set, _ := proposal(t)
	target := set.Findings[0].VerificationTargets[0]
	stale := ref(target)
	stale.Subject.Epoch++
	wrong := ref(target)
	wrong.Subject.Contract.Shape = "different"
	partial := ref(target)
	partial.EvidenceFingerprint = ""
	duplicate := ref(target)
	cases := []struct {
		name string
		refs []EvidenceReference
		want Diagnostic
	}{
		{"stale", []EvidenceReference{stale}, ReferenceStale},
		{"mismatch", []EvidenceReference{wrong}, ReferenceMismatch},
		{"incomplete", []EvidenceReference{partial}, ReferenceIncomplete},
		{"ambiguous", []EvidenceReference{duplicate, duplicate}, ReferenceAmbiguous},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Reconcile(set, tc.refs)
			if got.Subjects[0].State != tc.want || got.Subjects[0].Reference != nil {
				t.Fatalf("got %+v", got.Subjects[0])
			}
		})
	}
}
func TestHandoffBlockedStructuralConflictAndCrossScope(t *testing.T) {
	set, r := proposal(t)
	target := set.Findings[0].VerificationTargets[0]
	wrongScope := ref(target)
	wrongScope.Subject.Scope = "tenant-b"
	result := Reconcile(set, []EvidenceReference{wrongScope})
	if result.Blocked || result.Subjects[0].State != ReferenceMissing {
		t.Fatalf("cross scope accepted %+v", result)
	}
	conflict := r.InspectWiringSet([]composition.WiringRequirement{
		{Contract: read, Scope: "tenant-a", ProviderID: "database", SlotID: "one"},
		{Contract: app, Scope: "tenant-a", ProviderID: "consumer", SlotID: "one"},
	})
	result = Reconcile(conflict, []EvidenceReference{ref(target)})
	if !result.Blocked || len(result.Subjects) != 0 {
		t.Fatalf("conflicting proposal produced handoff %+v", result)
	}
	set.Findings[0].VerificationTargets[0].Epoch++
	result = Reconcile(set, []EvidenceReference{ref(target)})
	if !result.Blocked {
		t.Fatalf("inconsistent snapshot produced handoff %+v", result)
	}
}
func TestHandoffDeduplicatesSubjectsAcrossMultipleConsumers(t *testing.T) {
	r := composition.NewRegistry()
	_, err := r.Apply(0, []composition.Provider{{ID: "database", Scope: "tenant-a", Offers: read}, {ID: "one", Scope: "tenant-a", Offers: app, Requires: []composition.Dependency{{Contract: read}}}, {ID: "two", Scope: "tenant-a", Offers: composition.Contract{ID: "other", Version: "1", Shape: "other"}, Requires: []composition.Dependency{{Contract: read}}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	set := r.InspectWiringSet([]composition.WiringRequirement{{Contract: app, Scope: "tenant-a"}, {Contract: composition.Contract{ID: "other", Version: "1", Shape: "other"}, Scope: "tenant-a"}})
	got := Reconcile(set, nil)
	if got.Blocked || len(got.Subjects) != 3 {
		t.Fatalf("shared dependency not deduplicated %+v", got)
	}
}

func TestHandoffRejectsMissingOrDuplicatedDependencyTargets(t *testing.T) {
	for _, modify := range []struct {
		name  string
		apply func(*composition.WiringSetFinding)
	}{
		{"missing", func(s *composition.WiringSetFinding) {
			s.Findings[0].VerificationTargets = s.Findings[0].VerificationTargets[:1]
		}},
		{"duplicate", func(s *composition.WiringSetFinding) {
			s.Findings[0].VerificationTargets[1] = s.Findings[0].VerificationTargets[0]
		}},
		{"wrong-epoch", func(s *composition.WiringSetFinding) {
			s.Findings[0].VerificationTargets[1].Epoch++
		}},
	} {
		t.Run(modify.name, func(t *testing.T) {
			set, _ := proposal(t)
			modify.apply(&set)
			got := Reconcile(set, nil)
			if !got.Blocked || len(got.Subjects) != 0 {
				t.Fatalf("forged closure produced subjects: %+v", got)
			}
		})
	}
}

func TestHandoffIncompleteArtifactDigestAndFingerprint(t *testing.T) {
	set, _ := proposal(t)
	target := set.Findings[0].VerificationTargets[0]
	for _, modify := range []struct {
		name  string
		apply func(*EvidenceReference)
	}{
		{"bad-digest", func(r *EvidenceReference) { r.ArtifactDigest = "sha256:invalid" }},
		{"missing-resolution", func(r *EvidenceReference) { r.ResolutionFingerprint = "" }},
		{"missing-implementation", func(r *EvidenceReference) { r.ImplementationID = "" }},
	} {
		t.Run(modify.name, func(t *testing.T) {
			candidate := ref(target)
			modify.apply(&candidate)
			got := Reconcile(set, []EvidenceReference{candidate})
			if got.Blocked || got.Subjects[0].State != ReferenceIncomplete || got.Subjects[0].Reference != nil {
				t.Fatalf("incomplete provenance accepted: %+v", got)
			}
		})
	}
}

func TestHandoffManyDependenciesAndUnrelatedReferences(t *testing.T) {
	const count = 64
	registry := composition.NewRegistry()
	providers := make([]composition.Provider, 0, count)
	for i := 0; i < count; i++ {
		contract := composition.Contract{
			ID: fmt.Sprintf("scale.contract.%03d", i), Version: "1", Shape: "v1",
		}
		provider := composition.Provider{
			ID: fmt.Sprintf("component-%03d", i), Scope: "tenant-a", Offers: contract,
		}
		if i > 0 {
			provider.Requires = []composition.Dependency{{Contract: providers[i-1].Offers}}
		}
		providers = append(providers, provider)
	}
	if _, err := registry.Apply(0, providers, nil); err != nil {
		t.Fatal(err)
	}
	set := registry.InspectWiringSet([]composition.WiringRequirement{{
		Contract: providers[count-1].Offers, Scope: "tenant-a",
	}})
	if !registry.ProposalCurrent(set) {
		t.Fatalf("unexpected stale dependency closure: %+v", set)
	}
	start := ReconcileCurrent(registry, set, nil)
	if start.Blocked || len(start.Subjects) != count {
		t.Fatalf("incomplete transitive verification targets: %+v", start)
	}
	refs := make([]EvidenceReference, 0, count*4)
	for _, target := range start.Subjects {
		refs = append(refs, ref(target.Target))
		// Unrelated scope must never influence a matched subject.
		otherTenant := ref(target.Target)
		otherTenant.Subject.Scope = "tenant-b"
		refs = append(refs, otherTenant)
		// Wrong contract and obsolete epoch may exist alongside the exact
		// reference, but cannot supersede it or make it ambiguous.
		wrongContract := ref(target.Target)
		wrongContract.Subject.Contract.Shape = "wrong"
		refs = append(refs, wrongContract)
		oldEpoch := ref(target.Target)
		oldEpoch.Subject.Epoch++
		refs = append(refs, oldEpoch)
	}
	report := ReconcileCurrent(registry, set, refs)
	if report.Blocked || len(report.Subjects) != count {
		t.Fatalf("unexpected blocked or incomplete report: %+v", report)
	}
	for i, item := range report.Subjects {
		if item.State != ReferenceLinked || item.Reference == nil ||
			item.Target.ProviderID != fmt.Sprintf("component-%03d", i) {
			t.Fatalf("reference %d not exactly matched: %+v", i, item)
		}
	}
	duplicate := ref(report.Subjects[count/2].Target)
	refs = append(refs, duplicate)
	ambiguous := ReconcileCurrent(registry, set, refs)
	for i, item := range ambiguous.Subjects {
		want := ReferenceLinked
		if i == count/2 {
			want = ReferenceAmbiguous
		}
		if item.State != want {
			t.Fatalf("reference %d got %s, want %s", i, item.State, want)
		}
	}
}

func BenchmarkHandoffManyDependencies(b *testing.B) {
	const count = 96
	registry := composition.NewRegistry()
	providers := make([]composition.Provider, 0, count)
	for i := 0; i < count; i++ {
		p := composition.Provider{
			ID: fmt.Sprintf("p%03d", i), Scope: "tenant-a",
			Offers: composition.Contract{ID: fmt.Sprintf("c%03d", i), Version: "1", Shape: "shape"},
		}
		if i > 0 {
			p.Requires = []composition.Dependency{{Contract: providers[i-1].Offers}}
		}
		providers = append(providers, p)
	}
	if _, err := registry.Apply(0, providers, nil); err != nil {
		b.Fatal(err)
	}
	set := registry.InspectWiringSet([]composition.WiringRequirement{{Contract: providers[count-1].Offers, Scope: "tenant-a"}})
	first := Reconcile(set, nil)
	refs := make([]EvidenceReference, 0, count*6)
	for _, subject := range first.Subjects {
		refs = append(refs, ref(subject.Target))
		for i := 0; i < 5; i++ {
			unrelated := ref(subject.Target)
			unrelated.Subject.Scope = fmt.Sprintf("other-%d", i)
			refs = append(refs, unrelated)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := Reconcile(set, refs); got.Blocked || len(got.Subjects) != count {
			b.Fatal("unexpected reconciliation result")
		}
	}
}

func TestHandoffUnrelatedCatalogCannotChangeOrExpandSubjects(t *testing.T) {
	set, registry := proposal(t)
	baseline := ReconcileCurrent(registry, set, nil)
	if baseline.Blocked || len(baseline.Subjects) != 2 {
		t.Fatalf("unexpected baseline: %+v", baseline)
	}
	// A large number of unrelated providers and scopes must not influence
	// the exact evidence references for this proposal.
	refs := make([]EvidenceReference, 0, 12002)
	for i := 0; i < 12000; i++ {
		target := baseline.Subjects[i%2].Target
		if i%2 == 0 {
			target.Scope = fmt.Sprintf("unrelated-tenant-%d", i)
		} else {
			target.ProviderID = fmt.Sprintf("unrelated-provider-%d", i)
		}
		refs = append(refs, ref(target))
	}
	onlyNoise := ReconcileCurrent(registry, set, refs)
	if onlyNoise.Blocked || len(onlyNoise.Subjects) != 2 {
		t.Fatalf("unrelated catalog affected structural result: %+v", onlyNoise)
	}
	for _, item := range onlyNoise.Subjects {
		if item.State != ReferenceMissing || item.Reference != nil {
			t.Fatalf("unrelated evidence affected provider status: %+v", item)
		}
	}
	for _, subject := range baseline.Subjects {
		refs = append(refs, ref(subject.Target))
	}
	matched := ReconcileCurrent(registry, set, refs)
	if matched.Blocked || len(matched.Subjects) != 2 {
		t.Fatalf("exact refs missing after unrelated catalog: %+v", matched)
	}
	for _, item := range matched.Subjects {
		if item.State != ReferenceLinked || item.Reference == nil {
			t.Fatalf("unrelated locator interfered with exact ref: %+v", item)
		}
	}
}

func TestHandoffManyExactReferencesRemainAmbiguous(t *testing.T) {
	set, registry := proposal(t)
	target := set.Findings[0].VerificationTargets[0]
	refs := make([]EvidenceReference, 0, 10004)
	for i := 0; i < 10000; i++ {
		refs = append(refs, ref(target))
	}
	report := ReconcileCurrent(registry, set, refs)
	if report.Blocked || len(report.Subjects) != 2 ||
		report.Subjects[0].State != ReferenceAmbiguous ||
		report.Subjects[0].Reference != nil {
		t.Fatalf("repeated exact locators were silently accepted: %+v", report)
	}
	// Adding unrelated, incorrect and obsolete references must not change
	// the exact-reference ambiguity, regardless of their position.
	wrong := ref(target)
	wrong.Subject.Contract.Shape = "different"
	stale := ref(target)
	stale.Subject.Epoch++
	foreign := ref(target)
	foreign.Subject.Scope = "another-tenant"
	refs = append([]EvidenceReference{wrong, stale, foreign}, refs...)
	report = ReconcileCurrent(registry, set, refs)
	if report.Blocked || report.Subjects[0].State != ReferenceAmbiguous {
		t.Fatalf("unrelated locators masked exact ambiguity: %+v", report)
	}
	// One exact locator takes precedence over other ineligible locators.
	report = ReconcileCurrent(registry, set, []EvidenceReference{stale, wrong, ref(target), foreign})
	if report.Blocked || report.Subjects[0].State != ReferenceLinked || report.Subjects[0].Reference == nil {
		t.Fatalf("single exact locator was not selected: %+v", report)
	}
}
