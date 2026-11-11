package cacheinvalidation

import (
	"strings"
	"testing"
)

func TestReuseArtifact_SourceDigestChanged(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:old"}
	curr := CacheKey{SourceDigest: "sha256:new"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseArtifact(prev, curr, closure)
	if dec.Allowed {
		t.Fatal("source digest change must deny artifact reuse")
	}
	if !dec.MustRecomputeArtifact {
		t.Fatal("artifact must be recomputed")
	}
	found := false
	for _, r := range dec.StaleReasons {
		if r == "source digest changed sha256:old -> sha256:new" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected source digest change reason, got: %v", dec.StaleReasons)
	}
}

func TestReuseArtifact_ModelRevisionChanged(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1"}
	curr := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R2"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseArtifact(prev, curr, closure)
	if dec.Allowed {
		t.Fatal("model revision change must deny artifact reuse")
	}
}

func TestReuseArtifact_WorldFingerprintChanged(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp1"}
	curr := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp2"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseArtifact(prev, curr, closure)
	if dec.Allowed {
		t.Fatal("world fingerprint change must deny artifact reuse")
	}
}

func TestReuseArtifact_ArtifactRevisionChanged(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{ArtifactRevision: "A1"}
	curr := CacheKey{ArtifactRevision: "A2"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseArtifact(prev, curr, closure)
	if dec.Allowed {
		t.Fatal("artifact revision change must deny artifact reuse")
	}
}

func TestReuseArtifact_ClosureUnknownDeniesReuse(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp1"}
	curr := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp1"}
	closure := Closure{Digest: "sha256:closure", Unknown: true, Missing: []string{"source"}}
	dec := e.ReuseArtifact(prev, curr, closure)
	if dec.Allowed {
		t.Fatal("unknown closure must deny artifact reuse")
	}
}

func TestReuseArtifact_ExactMatchAllowsReuse(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseArtifact(prev, curr, closure)
	if !dec.Allowed {
		t.Fatalf("exact match must allow artifact reuse: %s", dec.Reason)
	}
}

func TestReuseBuild_SourceChangedDeniesBuildSkip(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:old"}
	curr := CacheKey{SourceDigest: "sha256:new"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseBuild(prev, curr, closure, false, StatusComplete)
	if dec.Allowed {
		t.Fatal("source change must deny build skip")
	}
	if !dec.MustRecomputeBuild {
		t.Fatal("build must be recomputed")
	}
}

func TestReuseBuild_OperatorPinChanged(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc", OperatorPin: "op-v1"}
	curr := CacheKey{SourceDigest: "sha256:abc", OperatorPin: "op-v2"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseBuild(prev, curr, closure, false, StatusComplete)
	if dec.Allowed {
		t.Fatal("operator pin change must deny build skip")
	}
}

func TestReuseBuild_PluginPinChanged(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc", PluginPin: "pl-v1"}
	curr := CacheKey{SourceDigest: "sha256:abc", PluginPin: "pl-v2"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseBuild(prev, curr, closure, false, StatusComplete)
	if dec.Allowed {
		t.Fatal("plugin pin change must deny build skip")
	}
}

func TestReuseBuild_GeneratorVersionChanged(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc", GeneratorVersion: "gen-v1"}
	curr := CacheKey{SourceDigest: "sha256:abc", GeneratorVersion: "gen-v2"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseBuild(prev, curr, closure, false, StatusComplete)
	if dec.Allowed {
		t.Fatal("generator version change must deny build skip")
	}
}

func TestReuseBuild_DependencyAddedDeniesBuildSkip(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc"}
	curr := CacheKey{SourceDigest: "sha256:abc"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseBuild(prev, curr, closure, true, StatusComplete)
	if dec.Allowed {
		t.Fatal("dependency topology change must deny build skip")
	}
}

func TestReuseBuild_DependencyRemovedDeniesBuildSkip(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc"}
	curr := CacheKey{SourceDigest: "sha256:abc"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseBuild(prev, curr, closure, true, StatusComplete)
	if dec.Allowed {
		t.Fatal("dependency topology change must deny build skip")
	}
}

func TestReuseBuild_MidFlightFailureDeniesBuildSkip(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc"}
	curr := CacheKey{SourceDigest: "sha256:abc"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseBuild(prev, curr, closure, false, StatusPartial)
	if dec.Allowed {
		t.Fatal("partial build must deny build skip")
	}
	found := false
	for _, r := range dec.StaleReasons {
		if r == "previous build status is partial" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected partial build reason, got: %v", dec.StaleReasons)
	}
}

func TestReuseBuild_FailedBuildDeniesBuildSkip(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc"}
	curr := CacheKey{SourceDigest: "sha256:abc"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseBuild(prev, curr, closure, false, StatusFailed)
	if dec.Allowed {
		t.Fatal("failed build must deny build skip")
	}
}

func TestReuseBuild_ExactMatchAllowsBuildSkip(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", OperatorPin: "op-v1", PluginPin: "pl-v1", GeneratorVersion: "gen-v1"}
	curr := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", OperatorPin: "op-v1", PluginPin: "pl-v1", GeneratorVersion: "gen-v1"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseBuild(prev, curr, closure, false, StatusComplete)
	if !dec.Allowed {
		t.Fatalf("exact match must allow build skip: %s", dec.Reason)
	}
}

func TestReuseEvidence_ScopeChanged(t *testing.T) {
	e := NewEngine(nil)
	prev := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := EvidenceRecord{Scope: "other", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseEvidence(prev, curr, closure, false)
	if dec.Allowed {
		t.Fatal("evidence scope change must deny evidence reuse")
	}
}

func TestReuseEvidence_RevisionChanged(t *testing.T) {
	e := NewEngine(nil)
	prev := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := EvidenceRecord{Scope: "default", Revision: "E2", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseEvidence(prev, curr, closure, false)
	if dec.Allowed {
		t.Fatal("evidence revision change must deny evidence reuse")
	}
}

func TestReuseEvidence_SubjectChanged(t *testing.T) {
	e := NewEngine(nil)
	prev := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.quote", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseEvidence(prev, curr, closure, false)
	if dec.Allowed {
		t.Fatal("evidence subject change must deny evidence reuse")
	}
}

func TestReuseEvidence_ABACycleWithoutProofDeniesReuse(t *testing.T) {
	e := NewEngine(nil)
	prev := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A2"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseEvidence(prev, curr, closure, false)
	if dec.Allowed {
		t.Fatal("A->B->A without continuity proof must deny evidence reuse")
	}
	found := false
	for _, r := range dec.StaleReasons {
		if strings.Contains(r, "A->B->A cycle") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected A->B->A reason, got: %v", dec.StaleReasons)
	}
}

func TestReuseEvidence_ABACycleWithProofAllowsReuse(t *testing.T) {
	// Continuity proof means the world went through intermediate changes but
	// returned to the same semantic state. Evidence may be reused only when
	// the artifact revision is also unchanged; otherwise the evidence was
	// collected for a different artifact.
	e := NewEngine(nil)
	prev := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseEvidence(prev, curr, closure, true)
	if !dec.Allowed {
		t.Fatalf("A->B->A with continuity proof and unchanged artifact must allow evidence reuse: %s", dec.Reason)
	}
}

func TestReuseEvidence_UnknownClosureDeniesReuse(t *testing.T) {
	e := NewEngine(nil)
	prev := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	closure := Closure{Digest: "sha256:closure", Unknown: true, Missing: []string{"source"}}
	dec := e.ReuseEvidence(prev, curr, closure, false)
	if dec.Allowed {
		t.Fatal("unknown closure must deny evidence reuse")
	}
}

func TestReuseEvidence_ExactMatchAllowsReuse(t *testing.T) {
	e := NewEngine(nil)
	prev := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseEvidence(prev, curr, closure, false)
	if !dec.Allowed {
		t.Fatalf("exact match must allow evidence reuse: %s", dec.Reason)
	}
}

func TestComputeClosure_EmptyRevisions(t *testing.T) {
	e := NewEngine([]DependencyEdge{})
	closure, err := e.ComputeClosure(nil)
	if err != nil {
		t.Fatalf("empty closure must not error: %v", err)
	}
	if closure.Unknown {
		t.Fatal("empty closure must not be unknown")
	}
	if len(closure.Members) != 0 {
		t.Fatalf("empty closure must have no members, got %d", len(closure.Members))
	}
}

func TestComputeClosure_MissingRequiredDependency(t *testing.T) {
	e := NewEngine([]DependencyEdge{
		{From: RevisionKindArtifact, To: RevisionKindSource, Scope: "default", Required: true},
	})
	_, err := e.ComputeClosure(nil)
	if err == nil {
		t.Fatal("missing required dependency must error")
	}
}

func TestComputeClosure_DeterministicOrdering(t *testing.T) {
	e := NewEngine([]DependencyEdge{
		{From: RevisionKindArtifact, To: RevisionKindSource, Scope: "default", Required: true},
		{From: RevisionKindArtifact, To: RevisionKindModel, Scope: "default", Required: true},
	})
	revs := []Revision{
		{Kind: RevisionKindModel, Scope: "default", Digest: "sha256:mod1"},
		{Kind: RevisionKindSource, Scope: "default", Digest: "sha256:src1"},
	}
	c1, err := e.ComputeClosure(revs)
	if err != nil {
		t.Fatalf("closure must not error: %v", err)
	}
	c2, err := e.ComputeClosure(revs)
	if err != nil {
		t.Fatalf("closure must not error on second call: %v", err)
	}
	if c1.Digest != c2.Digest {
		t.Fatalf("closure digest must be deterministic: %s != %s", c1.Digest, c2.Digest)
	}
}

func TestDependencyChanged_AddsEdge(t *testing.T) {
	prev := []DependencyEdge{{From: RevisionKindArtifact, To: RevisionKindSource, Scope: "default", Required: true}}
	curr := append([]DependencyEdge(nil), prev...)
	curr = append(curr, DependencyEdge{From: RevisionKindArtifact, To: RevisionKindModel, Scope: "default", Required: true})
	if !DependencyChanged(prev, curr) {
		t.Fatal("added dependency must be detected")
	}
}

func TestDependencyChanged_RemovesEdge(t *testing.T) {
	prev := []DependencyEdge{
		{From: RevisionKindArtifact, To: RevisionKindSource, Scope: "default", Required: true},
		{From: RevisionKindArtifact, To: RevisionKindModel, Scope: "default", Required: true},
	}
	curr := []DependencyEdge{{From: RevisionKindArtifact, To: RevisionKindSource, Scope: "default", Required: true}}
	if !DependencyChanged(prev, curr) {
		t.Fatal("removed dependency must be detected")
	}
}

func TestDependencyChanged_NoChange(t *testing.T) {
	deps := []DependencyEdge{{From: RevisionKindArtifact, To: RevisionKindSource, Scope: "default", Required: true}}
	if DependencyChanged(deps, append([]DependencyEdge(nil), deps...)) {
		t.Fatal("identical dependency sets must not be reported as changed")
	}
}

func TestReuseArtifact_SourceByteMatchAloneIsInsufficient(t *testing.T) {
	// Source bytes match but world fingerprint differs. The engine must not
	// reuse the artifact based on source bytes alone.
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp2", ArtifactRevision: "A1"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseArtifact(prev, curr, closure)
	if dec.Allowed {
		t.Fatal("same source bytes with different world fingerprint must deny artifact reuse")
	}
}

func TestReuseEvidence_PartialRetrievalUnknownDependency(t *testing.T) {
	e := NewEngine([]DependencyEdge{
		{From: RevisionKindEvidence, To: RevisionKindArtifact, Scope: "default", Required: true},
		{From: RevisionKindEvidence, To: RevisionKindSource, Scope: "default", Required: true},
	})
	prev := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	closure := Closure{Digest: "sha256:closure", Unknown: true, Missing: []string{"source"}}
	dec := e.ReuseEvidence(prev, curr, closure, false)
	if dec.Allowed {
		t.Fatal("partial retrieval with unknown dependency must deny evidence reuse")
	}
}

func TestReuseBuild_EmptyDependencySet(t *testing.T) {
	e := NewEngine([]DependencyEdge{})
	prev := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	closure := Closure{Digest: "sha256:closure", Members: []Revision{}}
	dec := e.ReuseBuild(prev, curr, closure, false, StatusComplete)
	if !dec.Allowed {
		t.Fatalf("explicit empty dependency set must allow build skip: %s", dec.Reason)
	}
}

func TestReuseBuild_SourceByteMatchAloneIsInsufficient(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp1"}
	curr := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp2"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseBuild(prev, curr, closure, false, StatusComplete)
	if dec.Allowed {
		t.Fatal("same source bytes with different world fingerprint must deny build skip")
	}
}

func TestReuseArtifact_ModelRevisionUnchanged(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseArtifact(prev, curr, closure)
	if !dec.Allowed {
		t.Fatalf("unchanged model revision must allow artifact reuse: %s", dec.Reason)
	}
}

func TestReuseEvidence_WorldFingerprintUnchanged(t *testing.T) {
	e := NewEngine(nil)
	prev := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseEvidence(prev, curr, closure, false)
	if !dec.Allowed {
		t.Fatalf("unchanged world fingerprint must allow evidence reuse: %s", dec.Reason)
	}
}

func TestComputeClosure_SingleDependency(t *testing.T) {
	e := NewEngine([]DependencyEdge{
		{From: RevisionKindArtifact, To: RevisionKindSource, Scope: "default", Required: true},
	})
	revs := []Revision{{Kind: RevisionKindSource, Scope: "default", Digest: "sha256:src1"}}
	closure, err := e.ComputeClosure(revs)
	if err != nil {
		t.Fatalf("closure must not error: %v", err)
	}
	if len(closure.Members) != 1 || closure.Members[0].Digest != "sha256:src1" {
		t.Fatalf("unexpected closure members: %+v", closure.Members)
	}
}

func TestDependencyChanged_RequiredFlagDiff(t *testing.T) {
	prev := []DependencyEdge{{From: RevisionKindArtifact, To: RevisionKindSource, Scope: "default", Required: true}}
	curr := []DependencyEdge{{From: RevisionKindArtifact, To: RevisionKindSource, Scope: "default", Required: false}}
	if !DependencyChanged(prev, curr) {
		t.Fatal("required flag change must be detected")
	}
}

func TestDependencyChanged_ScopeDiff(t *testing.T) {
	prev := []DependencyEdge{{From: RevisionKindArtifact, To: RevisionKindSource, Scope: "default", Required: true}}
	curr := []DependencyEdge{{From: RevisionKindArtifact, To: RevisionKindSource, Scope: "other", Required: true}}
	if !DependencyChanged(prev, curr) {
		t.Fatal("scope change must be detected")
	}
}

func BenchmarkReuseArtifact(b *testing.B) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	closure := Closure{Digest: "sha256:closure"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.ReuseArtifact(prev, curr, closure)
	}
}

func BenchmarkReuseBuild(b *testing.B) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc", OperatorPin: "op-v1", PluginPin: "pl-v1", GeneratorVersion: "gen-v1"}
	curr := CacheKey{SourceDigest: "sha256:abc", OperatorPin: "op-v1", PluginPin: "pl-v1", GeneratorVersion: "gen-v1"}
	closure := Closure{Digest: "sha256:closure"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.ReuseBuild(prev, curr, closure, false, StatusComplete)
	}
}

func BenchmarkReuseEvidence(b *testing.B) {
	e := NewEngine(nil)
	prev := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	closure := Closure{Digest: "sha256:closure"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.ReuseEvidence(prev, curr, closure, false)
	}
}

func TestReuseArtifact_MultipleStaleReasons(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc", ModelRevision: "R1", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := CacheKey{SourceDigest: "sha256:new", ModelRevision: "R2", WorldFingerprint: "sha256:wp2", ArtifactRevision: "A2"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseArtifact(prev, curr, closure)
	if dec.Allowed {
		t.Fatal("multiple changes must deny artifact reuse")
	}
	if len(dec.StaleReasons) != 4 {
		t.Fatalf("expected 4 stale reasons, got %d: %v", len(dec.StaleReasons), dec.StaleReasons)
	}
}

func TestReuseBuild_MultipleStaleReasons(t *testing.T) {
	e := NewEngine(nil)
	prev := CacheKey{SourceDigest: "sha256:abc", OperatorPin: "op-v1", PluginPin: "pl-v1", GeneratorVersion: "gen-v1"}
	curr := CacheKey{SourceDigest: "sha256:new", OperatorPin: "op-v2", PluginPin: "pl-v2", GeneratorVersion: "gen-v2"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseBuild(prev, curr, closure, true, StatusPartial)
	if dec.Allowed {
		t.Fatal("multiple changes must deny build skip")
	}
	if len(dec.StaleReasons) != 6 {
		t.Fatalf("expected 6 stale reasons, got %d: %v", len(dec.StaleReasons), dec.StaleReasons)
	}
}

func TestReuseEvidence_MultipleStaleReasons(t *testing.T) {
	e := NewEngine(nil)
	prev := EvidenceRecord{Scope: "default", Revision: "E1", Subject: "sales.order", WorldFingerprint: "sha256:wp1", ArtifactRevision: "A1"}
	curr := EvidenceRecord{Scope: "other", Revision: "E2", Subject: "sales.quote", WorldFingerprint: "sha256:wp2", ArtifactRevision: "A2"}
	closure := Closure{Digest: "sha256:closure"}
	dec := e.ReuseEvidence(prev, curr, closure, false)
	if dec.Allowed {
		t.Fatal("multiple changes must deny evidence reuse")
	}
	if len(dec.StaleReasons) != 5 {
		t.Fatalf("expected 5 stale reasons, got %d: %v", len(dec.StaleReasons), dec.StaleReasons)
	}
}
