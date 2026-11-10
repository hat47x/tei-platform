package hashid

import (
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

func indexedReadFixture(t *testing.T) (sourceindex.Index, ReadClaim, []byte) {
	t.Helper()
	a := []byte("data Order { attribute orderId type std.identifier }")
	b := []byte("data Customer { attribute customerId type std.identifier }")
	artifactA, err := sourceindex.NewArtifact("semantic/order.tei", a)
	if err != nil {
		t.Fatal(err)
	}
	artifactB, err := sourceindex.NewArtifact("semantic/customer.tei", b)
	if err != nil {
		t.Fatal(err)
	}
	index := sourceindex.Index{Artifacts: []sourceindex.Artifact{artifactA, artifactB}}
	claim := referenceReadClaim(a)
	claim.SourceRevision = sourceindex.ModelRevision(index)
	return index, claim, a
}

func TestExistingSourceIndexBindingIsOnlyCallerIndexConsistency(t *testing.T) {
	index, claim, bytes := indexedReadFixture(t)
	report, err := CompareDeclaredSourceIndexRead(index, "semantic/order.tei", claim, bytes)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "caller-index-bytes-consistent-only" || report.Authority != "none" ||
		report.SourceIndexRevision != claim.SourceRevision ||
		report.ArtifactFingerprint != sourceindex.Fingerprint(bytes) ||
		report.SourceReaderVerified || report.QualifiedSubjectVerified ||
		report.SourceScopeVerified || report.ReadAttemptVerified ||
		report.SourceAccepted || report.AllSourceBytesVerified {
		t.Fatalf("read-only comparison was promoted to Source/Owner authority: %+v", report)
	}
	// The Source Index ModelRevision covers path+fingerprints for the whole
	// managed set; it is not the selected file's raw content digest.
	if claim.SourceRevision == "sha256:"+claim.SourceSHA256 {
		t.Fatal("whole-model revision was conflated with a file fingerprint")
	}
	// Artifact order does not affect the deterministic source revision.
	index.Artifacts[0], index.Artifacts[1] = index.Artifacts[1], index.Artifacts[0]
	if again, err := CompareDeclaredSourceIndexRead(index, "semantic/order.tei", claim, bytes); err != nil ||
		again.SourceIndexRevision != claim.SourceRevision {
		t.Fatalf("order-only index change should preserve revision: %+v %v", again, err)
	}
}

func TestExistingSourceIndexRejectsStaleSiblingOrPathRevision(t *testing.T) {
	index, claim, a := indexedReadFixture(t)
	// The selected artifact remains byte-for-byte unchanged. Changes to an
	// unrelated managed Source nevertheless change the true ModelRevision.
	bNew, err := sourceindex.NewArtifact("semantic/customer.tei", []byte("data Customer { attribute customerId type std.string }"))
	if err != nil {
		t.Fatal(err)
	}
	index.Artifacts[1] = bNew
	if _, err := CompareDeclaredSourceIndexRead(index, "semantic/order.tei", claim, a); err == nil ||
		!strings.Contains(err.Error(), "ModelRevision") {
		t.Fatalf("changing sibling source must invalidate whole-model revision: %v", err)
	}
	// A renamed file with the same content also changes the managed set.
	index, claim, a = indexedReadFixture(t)
	index.Artifacts[1].Path = "semantic/renamed-customer.tei"
	if _, err := CompareDeclaredSourceIndexRead(index, "semantic/order.tei", claim, a); err == nil {
		t.Fatal("renaming unrelated source must invalidate whole-model revision")
	}
	// Even a syntactically updated revision does not mean the entire index
	// has been read atomically or authenticated by a trusted Source Reader.
	claim.SourceRevision = sourceindex.ModelRevision(index)
	if diagnostic, err := CompareDeclaredSourceIndexRead(index, "semantic/order.tei", claim, a); err != nil ||
		diagnostic.SourceReaderVerified || diagnostic.AllSourceBytesVerified || diagnostic.SourceAccepted {
		t.Fatalf("updated caller revision was mistaken for Reader proof: %+v %v", diagnostic, err)
	}
}

func TestExistingSourceIndexRejectsMalformedOrAmbiguousProjections(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*sourceindex.Index, *ReadClaim, *[]byte)
	}{
		{"missing artifact", func(i *sourceindex.Index, c *ReadClaim, a *[]byte) { i.Artifacts = i.Artifacts[1:] }},
		{"duplicate artifact", func(i *sourceindex.Index, c *ReadClaim, a *[]byte) { i.Artifacts = append(i.Artifacts, i.Artifacts[0]) }},
		{"noncanonical artifact path", func(i *sourceindex.Index, c *ReadClaim, a *[]byte) {
			i.Artifacts[0].Path = "semantic/../semantic/order.tei"
		}},
		{"traversing artifact path", func(i *sourceindex.Index, c *ReadClaim, a *[]byte) { i.Artifacts[0].Path = "../outside.tei" }},
		{"invalid digest spelling", func(i *sourceindex.Index, c *ReadClaim, a *[]byte) {
			i.Artifacts[0].Fingerprint = strings.ToUpper(i.Artifacts[0].Fingerprint)
		}},
		{"changed consumed bytes", func(i *sourceindex.Index, c *ReadClaim, a *[]byte) { *a = []byte("DIFFERENT") }},
		{"changed caller digest", func(i *sourceindex.Index, c *ReadClaim, a *[]byte) { c.SourceSHA256 = strings.Repeat("0", 64) }},
		{"changed caller revision", func(i *sourceindex.Index, c *ReadClaim, a *[]byte) {
			c.SourceRevision = "sha256:" + strings.Repeat("f", 64)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			index, claim, actual := indexedReadFixture(t)
			tc.mutate(&index, &claim, &actual)
			if diag, err := CompareDeclaredSourceIndexRead(index, "semantic/order.tei", claim, actual); err == nil ||
				diag.Status != "" {
				t.Fatalf("invalid source index projection accepted: %+v %v", diag, err)
			}
		})
	}
	index, claim, data := indexedReadFixture(t)
	for _, path := range []string{"semantic/../semantic/order.tei", "../semantic/order.tei", "semantic\\order.tei"} {
		if diag, err := CompareDeclaredSourceIndexRead(index, path, claim, data); err == nil ||
			diag.Status != "" {
			t.Fatalf("noncanonical source lookup accepted: path=%q diagnostic=%+v err=%v", path, diag, err)
		}
	}
}

func TestExistingSourceIndexAlsoCannotDetectABAAcceptance(t *testing.T) {
	indexA, claimA, bytesA := indexedReadFixture(t)
	// A -> B -> A can restore both the per-file fingerprint and the
	// whole-model revision if no independently monotonic epoch was preserved.
	other, err := sourceindex.NewArtifact("semantic/order.tei", []byte("B"))
	if err != nil {
		t.Fatal(err)
	}
	indexB := indexA
	indexB.Artifacts = append([]sourceindex.Artifact(nil), indexA.Artifacts...)
	indexB.Artifacts[0] = other
	if indexARevision, indexBRevision := sourceindex.ModelRevision(indexA), sourceindex.ModelRevision(indexB); indexARevision == indexBRevision {
		t.Fatal("different managed source content must change source index revision")
	}
	if _, err := CompareDeclaredSourceIndexRead(indexB, "semantic/order.tei", claimA, []byte("B")); err == nil {
		t.Fatal("stale A claim was accepted against intermediate B")
	}
	restored := indexA
	report, err := CompareDeclaredSourceIndexRead(restored, "semantic/order.tei", claimA, bytesA)
	if err != nil || report.SourceAccepted || report.ReadAttemptVerified {
		t.Fatalf("restored index and content must not imply monotonic acceptance: %+v %v", report, err)
	}
}

func TestSourceIndexCannotIndependentlyVerifyScopeSubjectOrAttempt(t *testing.T) {
	index, claim, bytes := indexedReadFixture(t)
	// Index.Artifacts only contains file-level path/fingerprint metadata.
	// No SourceScope, QualifiedSubject, or ReadAttemptID is certified by
	// this read-only artifact projection, even with a valid ModelRevision.
	claim.QualifiedSubject = "another-claimed-subject"
	claim.SourceScope = "workspace:unrelated"
	claim.ReadAttemptID = "attempt:replayed"
	result, err := CompareDeclaredSourceIndexRead(index, "semantic/order.tei", claim, bytes)
	if err != nil {
		t.Fatalf("local index/byte consistency must not pretend to verify unrelated labels: %v", err)
	}
	if result.QualifiedSubjectVerified || result.SourceScopeVerified ||
		result.ReadAttemptVerified || result.SourceReaderVerified ||
		result.SourceAccepted || result.Authority != "none" {
		t.Fatalf("an artifact index incorrectly certified scope/subject/attempt: %+v", result)
	}
}
