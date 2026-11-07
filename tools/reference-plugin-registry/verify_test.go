package registryreference

import (
	"encoding/json"
	"testing"
)

func fixture(t *testing.T) Fixture {
	t.Helper()
	f, e := ReadFixture("fixture.json")
	if e != nil {
		t.Fatal(e)
	}
	return f
}
func clone(t *testing.T, f Fixture) Fixture {
	t.Helper()
	data, e := json.Marshal(f)
	if e != nil {
		t.Fatal(e)
	}
	var out Fixture
	if e = json.Unmarshal(data, &out); e != nil {
		t.Fatal(e)
	}
	return out
}
func TestPositiveAndNegativeClaims(t *testing.T) {
	f := fixture(t)
	if e := Verify(f, DefaultPolicy()); e != nil {
		t.Fatalf("positive fixture rejected: %v", e)
	}
	for _, tc := range []struct {
		name   string
		change func(*Fixture, *Policy)
	}{
		{"tamper-artifact", func(f *Fixture, p *Policy) { f.Artifacts[0].BytesBase64 = "dGFtcGVyZWQ=" }},
		{"missing-member", func(f *Fixture, p *Policy) { f.Artifacts = f.Artifacts[:1] }},
		{"duplicate-member", func(f *Fixture, p *Policy) { f.Artifacts[1].Role = "core" }},
		{"malformed-base64", func(f *Fixture, p *Policy) { f.Artifacts[0].BytesBase64 = "%%%" }},
		{"tamper-release", func(f *Fixture, p *Policy) { f.ReleaseText += "extra\n" }},
		{"wrong-attestation-subject", func(f *Fixture, p *Policy) { f.Attestation.Subject = "sha256:bad" }},
		{"wrong-target", func(f *Fixture, p *Policy) { f.Attestation.Target = "wasm32-wasi" }},
		{"tamper-profile", func(f *Fixture, p *Policy) { f.ProfileText += "altered" }},
		{"revoked-publisher", func(f *Fixture, p *Policy) { p.PublisherTrusted = false }},
		{"revoked-verifier", func(f *Fixture, p *Policy) { p.VerifierTrusted = false }},
		{"stale-trust", func(f *Fixture, p *Policy) { p.Fresh = false }},
		{"changed-owner-epoch", func(f *Fixture, p *Policy) { p.OwnerEpoch++ }},
		{"forged-publisher-signature", func(f *Fixture, p *Policy) { f.PublisherSignatureBase64 = "dGVzdA==" }},
		{"forged-verifier-signature", func(f *Fixture, p *Policy) { f.Attestation.SignatureBase64 = "dGVzdA==" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f2 := clone(t, f)
			p := DefaultPolicy()
			tc.change(&f2, &p)
			if e := Verify(f2, p); e == nil {
				t.Fatal("unexpected acceptance")
			}
		})
	}
}
