package hashid

import (
	"strings"
	"testing"
)

func decimalTarget() TargetProfile {
	return TargetProfile{
		Namespace:       "example.records",
		ProfileVersion:  "ascii-v1",
		AllowedAlphabet: "0123456789",
		ExactLength:     24,
	}
}

func decimalScheme() Scheme {
	s := fixtureScheme()
	s.Algorithm = AlgorithmSHAKE256
	s.Mode = ModeExactCharacters
	s.OutputBits = 0
	s.Encoding = ""
	s.Alphabet = "0123456789"
	s.Length = 24
	return s
}

func TestTargetExactCharactersAndIdempotency(t *testing.T) {
	s := decimalScheme()
	r, err := Derive(s, []byte("record-A"))
	if err != nil {
		t.Fatal(err)
	}
	profile := decimalTarget()
	key, err := CheckTargetBinding(r, "customer:a", profile, nil)
	if err != nil || key != r.Text {
		t.Fatalf("valid decimal target rejected: key=%q, err=%v", key, err)
	}
	_, err = CheckTargetBinding(r, "customer:a", profile, []TargetBinding{
		{Namespace: profile.Namespace, ProfileVersion: profile.ProfileVersion, Subject: "customer:a", Scheme: r.Scheme, Text: r.Text},
	})
	if err != nil {
		t.Fatalf("idempotent physical allocation rejected: %v", err)
	}
	_, err = CheckTargetBinding(r, "customer:b", profile, []TargetBinding{
		{Namespace: profile.Namespace, ProfileVersion: profile.ProfileVersion, Subject: "customer:a", Scheme: r.Scheme, Text: r.Text},
	})
	if err == nil {
		t.Fatal("same physical ID for different source must be rejected")
	}
	_, err = CheckTargetBinding(r, "customer:a", profile, []TargetBinding{
		{Namespace: profile.Namespace, ProfileVersion: profile.ProfileVersion, Subject: "customer:a", Scheme: r.Scheme, Text: strings.Repeat("0", 24)},
	})
	if err == nil {
		t.Fatal("silent physical reassignment must be rejected")
	}

	_, err = CheckTargetBinding(r, "customer:a", profile, []TargetBinding{
		{Namespace: profile.Namespace, ProfileVersion: "ascii-v0", Subject: "customer:a", Scheme: r.Scheme, Text: r.Text},
	})
	if err == nil {
		t.Fatal("different version of target collation rules must not be silently accepted")
	}
	otherScheme := r.Scheme
	otherScheme.Purpose = "another.purpose"
	_, err = CheckTargetBinding(r, "customer:a", profile, []TargetBinding{
		{Namespace: profile.Namespace, ProfileVersion: profile.ProfileVersion, Subject: "customer:a", Scheme: otherScheme, Text: r.Text},
	})
	if err == nil {
		t.Fatal("idempotent allocation must reject an existing different hash scheme")
	}
	// A distinct target namespace does not share a physical uniqueness key.
	_, err = CheckTargetBinding(r, "customer:b", profile, []TargetBinding{
		{Namespace: "other.target", ProfileVersion: profile.ProfileVersion, Subject: "customer:a", Scheme: r.Scheme, Text: r.Text},
	})
	if err != nil {
		t.Fatalf("unrelated physical namespace rejected: %v", err)
	}
}

func TestTargetCaseInsensitiveAndReserved(t *testing.T) {
	s := fixtureScheme()
	r, err := Derive(s, []byte("record-A"))
	if err != nil {
		t.Fatal(err)
	}
	profile := TargetProfile{
		Namespace:       "example.sqlite.casefold",
		ProfileVersion:  "ascii-v1",
		AllowedAlphabet: "0123456789abcdefABCDEF",
		ExactLength:     64,
		CaseFoldASCII:   true,
	}
	if _, err := CheckTargetBinding(r, "record:a", profile, nil); err != nil {
		t.Fatal(err)
	}
	_, err = CheckTargetBinding(r, "record:b", profile, []TargetBinding{
		{Namespace: profile.Namespace, ProfileVersion: profile.ProfileVersion, Subject: "record:a", Scheme: r.Scheme, Text: strings.ToUpper(r.Text)},
	})
	if err == nil {
		t.Fatal("case-insensitive collision was accepted")
	}
	profile.Reserved = []string{strings.ToUpper(r.Text)}
	if _, err := CheckTargetBinding(r, "record:a", profile, nil); err == nil {
		t.Fatal("reserved name must be rejected under case folding")
	}
}

func TestTargetRejectsInvalidAndAmbiguousScheme(t *testing.T) {
	s := fixtureScheme()
	r, err := Derive(s, []byte("record-A"))
	if err != nil {
		t.Fatal(err)
	}
	// 64-character hex cannot be represented by this 32-character target.
	profile := TargetProfile{
		Namespace:       "example.database",
		ProfileVersion:  "ascii-v1",
		AllowedAlphabet: "0123456789abcdef",
		MaxLength:       32,
	}
	if _, err := CheckTargetBinding(r, "record:a", profile, nil); err == nil {
		t.Fatal("target max length must reject overlong encoded digest")
	}
	profile.MaxLength = 64
	bad := r
	bad.Text = strings.ToUpper(r.Text)
	if _, err := CheckTargetBinding(bad, "record:a", profile, nil); err == nil {
		t.Fatal("noncanonical result text must be rejected")
	}
	bad = r
	bad.Digest = bad.Digest[:10]
	if _, err := CheckTargetBinding(bad, "record:a", profile, nil); err == nil {
		t.Fatal("malformed digest must be rejected")
	}
	profile.AllowedAlphabet = "0123456789abcdefA"
	if _, err := CheckTargetBinding(r, "record:a", profile, nil); err != nil {
		t.Fatalf("an unused uppercase character must not reject lowercase SHA-256: %v", err)
	}
	profile.AllowedAlphabet = strings.ReplaceAll("0123456789abcdef", r.Text[:1], "")
	if _, err := CheckTargetBinding(r, "record:a", profile, nil); err == nil {
		t.Fatal("missing allowed character should reject source")
	}

	s.Encoding = EncodingBase64URL
	r, err = Derive(s, []byte("record-A"))
	if err != nil {
		t.Fatal(err)
	}
	profile.AllowedAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	profile.MaxLength = 0
	profile.CaseFoldASCII = true
	if _, err := CheckTargetBinding(r, "record:a", profile, nil); err == nil {
		t.Fatal("base64url effective alphabet collapses on case-insensitive target")
	}
}

func TestTargetSnapshotAndPolicyFailClosed(t *testing.T) {
	s := decimalScheme()
	r, err := Derive(s, []byte("record-A"))
	if err != nil {
		t.Fatal(err)
	}
	p := decimalTarget()
	tests := []struct {
		name string
		edit func(*TargetProfile)
	}{
		{"missing target namespace", func(p *TargetProfile) { p.Namespace = "" }},
		{"missing target version", func(p *TargetProfile) { p.ProfileVersion = "" }},
		{"duplicate allowed alphabet", func(p *TargetProfile) { p.AllowedAlphabet = "00123456789" }},
		{"unicode allowed alphabet", func(p *TargetProfile) { p.AllowedAlphabet = "0123456789β" }},
		{"length conflicts", func(p *TargetProfile) { p.MinLength = 2 }},
		{"impossible bounds", func(p *TargetProfile) { p.ExactLength = 0; p.MaxLength = 2; p.MinLength = 3 }},
		{"reserved current value", func(p *TargetProfile) { p.Reserved = []string{r.Text} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bad := p
			tt.edit(&bad)
			if _, err := CheckTargetBinding(r, "record:a", bad, nil); err == nil {
				t.Fatal("invalid policy must reject")
			}
		})
	}
	if _, err := CheckTargetBinding(r, "", p, nil); err == nil {
		t.Fatal("missing source identity must reject")
	}
	if _, err := CheckTargetBinding(r, "record:a", p, []TargetBinding{
		{Namespace: p.Namespace, ProfileVersion: p.ProfileVersion, Subject: "", Scheme: r.Scheme, Text: r.Text},
	}); err == nil {
		t.Fatal("unqualified existing subject must reject")
	}
	if _, err := CheckTargetBinding(r, "record:a", p, []TargetBinding{
		{Namespace: p.Namespace, ProfileVersion: p.ProfileVersion, Subject: "record:b", Scheme: r.Scheme, Text: "abc"},
	}); err == nil {
		t.Fatal("invalid allocation snapshot must reject")
	}
}

func TestTargetRejectsCorruptUnrelatedAllocationSnapshot(t *testing.T) {
	s := decimalScheme()
	r, err := Derive(s, []byte("candidate-independent"))
	if err != nil {
		t.Fatal(err)
	}
	p := decimalTarget()
	prior := strings.Repeat("0", 24)
	if prior == r.Text {
		prior = strings.Repeat("1", 24)
	}
	base := TargetBinding{
		Namespace:      p.Namespace,
		ProfileVersion: p.ProfileVersion,
		Subject:        "other:first",
		Scheme:         s,
		Text:           prior,
	}
	second := base
	second.Subject = "other:second"
	if _, err := CheckTargetBinding(r, "candidate:new", p, []TargetBinding{base, second}); err == nil {
		t.Fatal("collision between unrelated old subjects was not detected")
	}
	second = base
	second.Text = strings.Repeat("2", 24)
	if _, err := CheckTargetBinding(r, "candidate:new", p, []TargetBinding{base, second}); err == nil {
		t.Fatal("duplicate existing subject with a different text was accepted")
	}
	second = base
	second.Subject = "other:second"
	second.Scheme = Scheme{}
	if _, err := CheckTargetBinding(r, "candidate:new", p, []TargetBinding{base, second}); err == nil {
		t.Fatal("invalid preexisting scheme was accepted")
	}
	// Existing data from another physical namespace cannot poison this one.
	base.Namespace = "another.namespace"
	second.Namespace = "another.namespace"
	if _, err := CheckTargetBinding(r, "candidate:new", p, []TargetBinding{base, second}); err != nil {
		t.Fatalf("unrelated namespace must be ignored: %v", err)
	}
}
