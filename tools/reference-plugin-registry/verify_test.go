package registryreference

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
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

type wireCase struct {
	ID      string `json:"id"`
	Find    string `json:"find"`
	Replace string `json:"replace"`
	SHA256  string `json:"sha256"`
	Accept  bool   `json:"accept"`
}

func TestSharedRawWireCases(t *testing.T) {
	original, err := os.ReadFile("fixture.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("wire-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var suite struct {
		Cases []wireCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &suite); err != nil {
		t.Fatal(err)
	}
	if len(suite.Cases) < 10 {
		t.Fatalf("missing shared wire cases: %d", len(suite.Cases))
	}
	for _, tc := range suite.Cases {
		t.Run(tc.ID, func(t *testing.T) {
			wire := original
			if tc.Find != "" {
				if n := bytes.Count(original, []byte(tc.Find)); n != 1 {
					t.Fatalf("mutation anchor count=%d, want 1", n)
				}
				wire = bytes.Replace(original, []byte(tc.Find), []byte(tc.Replace), 1)
			}
			sum := sha256.Sum256(wire)
			if hex.EncodeToString(sum[:]) != tc.SHA256 {
				t.Fatalf("shared raw-wire digest mismatch for %s", tc.ID)
			}
			f, err := ParseFixture(wire)
			accepted := false
			if err == nil && Verify(f, DefaultPolicy()) == nil {
				accepted = true
			}
			if accepted != tc.Accept {
				t.Fatalf("wire acceptance mismatch case=%s got=%v want=%v (parse error=%v)", tc.ID, accepted, tc.Accept, err)
			}
		})
	}
}
