package externaladoptionauthority

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	semanticcompositionpreflight "tei/reference/semantic-composition-preflight"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindRepo(wd)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPinnedTableHasOneRevisionAndFourBlobs(t *testing.T) {
	p, err := ReadPinnedTable(filepath.Join(repoRoot(t), "reference", "sui-adoption", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if p.Repository != "hat47x/sui-sensemaking" || len(p.Revision) != 40 || len(p.Blobs) != 4 {
		t.Fatalf("pinned table: %+v", p)
	}
}

func TestCheckExternalPinStopsOnAnyBlobButNotOnRevision(t *testing.T) {
	p, err := ReadPinnedTable(filepath.Join(repoRoot(t), "reference", "sui-adoption", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range pinnedFiles {
		o := PinnedSource{Repository: p.Repository, Revision: p.Revision, Blobs: map[string]string{}}
		for k, v := range p.Blobs {
			o.Blobs[k] = v
		}
		o.Blobs[name] = "changed"
		if _, err := CheckExternalPin(p, o); !errors.Is(err, ErrInputMismatch) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	o := p
	o.Revision = strings.Repeat("f", 40)
	pc, err := CheckExternalPin(p, o)
	if err != nil || !pc.RevisionAdvanced {
		t.Fatalf("revision only: %+v %v", pc, err)
	}
}

func syntheticChain(t *testing.T) *Chain {
	t.Helper()
	f := false
	c1 := Carried{
		Source:      PinnedSource{Repository: "r", Revision: "v", Blobs: map[string]string{"a": "1", "b": "2", "c": "3", "d": "4"}},
		InputDigest: "i", MappingReportDigest: "m", Namespace: "sui", Subject: "semantic-data:sui.island",
		Lossless: &f, ParentPresence: "unresolved", LossRecordDigest: "l",
	}
	c2 := c1
	c2.SemanticDigest, c2.SemanticSetFingerprint, c2.SemanticLockDigest = "s", "set", "lock"
	c2.ValidationPlanFingerprint, c2.ExtensionNamespace, c2.NamespaceOwner, c2.OwnerIssuer = "plan", "ns", "owner", "issuer"
	c3 := c2
	c3.ArtifactDigest, c3.PlanFingerprint, c3.Package, c3.SignerRef, c3.SignerKeyFingerprint = "a", "p", "pkg", "signer", "key"
	ch := NewChain()
	for i, c := range []Carried{c1, c2, c3, c3} {
		if err := ch.Append(i+1, c, map[string]any{}); err != nil {
			t.Fatal(err)
		}
	}
	return ch
}

func TestVerifyThroughNeedsEveryStageOneIdentityAndLosslessFalse(t *testing.T) {
	ch := syntheticChain(t)
	if r := VerifyThrough(*ch); !r.Complete {
		t.Fatalf("synthetic chain: %+v", r)
	}
	tr := true
	forged := ch.Clone()
	forged.Records[2].Carried.Lossless = &tr
	forged.Relink()
	if r := VerifyThrough(forged); r.Complete {
		t.Fatal("lossless=true at stage 3 was not detected")
	}
	forged = ch.Clone()
	forged.Records[3].Carried.ArtifactDigest = "other"
	forged.Relink()
	if r := VerifyThrough(forged); r.Complete {
		t.Fatal("swapped artifact was not detected")
	}
	forged = ch.Clone()
	forged.Records = forged.Records[:3]
	if r := VerifyThrough(forged); r.Complete || len(r.MissingStages) != 1 {
		t.Fatalf("missing stage: %+v", r)
	}
	if _, err := BuildCompletionReport(forged); err == nil {
		t.Fatal("completion report was built from an incomplete chain")
	}
	rep, err := BuildCompletionReport(*ch)
	if err != nil {
		t.Fatal(err)
	}
	if mm := VerifyCompletionReport(*ch, rep); len(mm) != 1 || !strings.Contains(mm[0], "parent-presence") {
		// The synthetic chain has no stage 1 detail, so only the
		// unresolved list is missing.
		t.Fatalf("report mismatches: %v", mm)
	}
}

func TestSignerLookupJudgesOnlyThePointItIsAskedAbout(t *testing.T) {
	k1 := NewSignerKey(SignerRef, "1.0.0", 1)
	on, err := NewGrant(SignerAuthorityClass, OwnerScope("sui"), issuerOf(k1), "p", true)
	if err != nil {
		t.Fatal(err)
	}
	off, err := NewGrant(SignerAuthorityClass, OwnerScope("sui"), issuerOf(k1), "p", false)
	if err != nil {
		t.Fatal(err)
	}
	var h TrustHistory
	p1 := h.Add(TrustPoint{Label: "on", SignerGrants: ownerList(on)}.WithKeys(k1))
	p2 := h.Add(TrustPoint{Label: "off", SignerGrants: ownerList(off)}.WithKeys(k1))
	sub := semanticcompositionpreflight.PublisherSubject{SignerRef: SignerRef, NamespaceID: "sui"}
	if _, err := p1.SignerLookup()(sub); err != nil {
		t.Fatal(err)
	}
	if _, err := p2.SignerLookup()(sub); err == nil || !strings.Contains(err.Error(), "revoked") {
		t.Fatalf("revoked grant: %v", err)
	}
	if _, err := p1.SignerLookup()(sub); err != nil {
		t.Fatalf("earlier point changed: %v", err)
	}
	if NewSignerKey(SignerRef, "1.0.0", 1).Fingerprint() != k1.Fingerprint() {
		t.Fatal("signer key is not deterministic")
	}
	if !p1.Now.Before(p2.Now) || p2.Now.After(PublicationValidUntil) {
		t.Fatalf("logical times: %v %v", p1.Now, p2.Now)
	}
}
