package handoff

import (
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
