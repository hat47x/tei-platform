package provisionaleffectgrant

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tei/internal/sourceacceptance/hoststore"
)

var t0 = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

func at(d time.Duration) func() time.Time { return func() time.Time { return t0.Add(d) } }

func ts(d time.Duration) string { return t0.Add(d).Format(time.RFC3339) }

type keys struct {
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
}

func newKeys(t *testing.T) keys {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return keys{pub, priv}
}

func entry(k keys, issuer, keyID string, mutate func(*hoststore.TrustedIssuer)) hoststore.TrustedIssuer {
	e := hoststore.TrustedIssuer{
		Issuer: issuer, KeyID: keyID, PublicKey: base64.StdEncoding.EncodeToString(k.pub),
		Applications: []string{"order_entry"},
	}
	if mutate != nil {
		mutate(&e)
	}
	return e
}

func trustJSON(t *testing.T, schema string, issuers ...hoststore.TrustedIssuer) []byte {
	t.Helper()
	raw, err := json.Marshal(hoststore.TrustPolicy{Schema: schema, Issuers: issuers})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func policyOf(t *testing.T, issuers ...hoststore.TrustedIssuer) hoststore.TrustPolicy {
	t.Helper()
	p, err := hoststore.ParseTrustPolicyWithSchema(trustJSON(t, TrustSchema, issuers...), TrustSchema)
	if err != nil {
		t.Fatal(err)
	}
	return p.WithClock(at(0))
}

func baseGrant() Grant {
	return Grant{
		Issuer: "dev-issuer", KeyID: "k1", ApplicationID: "order_entry",
		Capability: "order_entry.submit_order", SubjectID: "editor", Resource: "order/42",
		Epochs:    Epochs{Source: "local-auth-store", SubjectLink: 10, Membership: 20, Policy: 30},
		NotBefore: ts(-time.Minute), NotAfter: ts(time.Hour),
	}
}

func baseQuery() Query {
	return Query{ApplicationID: "order_entry", Capability: "order_entry.submit_order", SubjectID: "editor", Resource: "order/42"}
}

func signed(t *testing.T, k keys, mutate func(*Grant)) Grant {
	t.Helper()
	g := baseGrant()
	if mutate != nil {
		mutate(&g)
	}
	out, err := Sign(g, k.priv)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || err.Error() != code {
		t.Fatalf("want %s, got %v", code, err)
	}
}

func TestVerifyAcceptsExactBindingAndMarksProvisional(t *testing.T) {
	k := newKeys(t)
	g := signed(t, k, nil)
	if !g.Provisional || g.Schema != Schema || !strings.Contains(g.Schema, "provisional") {
		t.Fatalf("not marked provisional: %+v", g)
	}
	v, err := Verify(g, baseQuery(), policyOf(t, entry(k, "dev-issuer", "k1", nil)), at(0))
	if err != nil {
		t.Fatal(err)
	}
	if v.Provenance != "provisional-grant" || !strings.HasPrefix(v.GrantRef, "provisional-grant:sha256:") {
		t.Fatalf("%+v", v)
	}
	a := v.Grant
	if a.SubjectID != "editor" || a.Action != "order_entry.submit_order" || a.Resource != "order/42" ||
		a.Epochs == nil || a.Epochs.Source != "local-auth-store" || a.Epochs.SubjectLink != 10 || a.Epochs.Membership != 20 || a.Epochs.Policy != 30 {
		t.Fatalf("%+v", a)
	}
}

func TestVerifyRejectsTamperingOfEveryBoundField(t *testing.T) {
	k := newKeys(t)
	policy := policyOf(t, entry(k, "dev-issuer", "k1", nil))
	mutations := map[string]func(*Grant){
		"capability": func(g *Grant) { g.Capability = "order_entry.cancel_order" },
		"subject":    func(g *Grant) { g.SubjectID = "admin" },
		"resource":   func(g *Grant) { g.Resource = "order/43" },
		"source":     func(g *Grant) { g.Epochs.Source = "other" },
		"subjectLnk": func(g *Grant) { g.Epochs.SubjectLink++ },
		"membership": func(g *Grant) { g.Epochs.Membership++ },
		"policy":     func(g *Grant) { g.Epochs.Policy++ },
		"notAfter":   func(g *Grant) { g.NotAfter = ts(2 * time.Hour) },
		"notBefore":  func(g *Grant) { g.NotBefore = ts(-2 * time.Minute) },
	}
	for name, mutate := range mutations {
		g := signed(t, k, nil)
		mutate(&g)
		_, err := Verify(g, Query{ApplicationID: g.ApplicationID, Capability: g.Capability, SubjectID: g.SubjectID, Resource: g.Resource}, policy, at(0))
		if err == nil || err.Error() != "HOST_STORE_SIGNATURE_INVALID" {
			t.Fatalf("%s: want signature invalid, got %v", name, err)
		}
	}
	// A flipped signature bit.
	g := signed(t, k, nil)
	raw, _ := base64.StdEncoding.DecodeString(g.Signature)
	raw[0] ^= 1
	g.Signature = base64.StdEncoding.EncodeToString(raw)
	_, err := Verify(g, baseQuery(), policy, at(0))
	wantCode(t, err, "HOST_STORE_SIGNATURE_INVALID")
	g.Signature = "not base64!"
	_, err = Verify(g, baseQuery(), policy, at(0))
	wantCode(t, err, "PROVISIONAL_GRANT_SIGNATURE_MALFORMED")
}

func TestVerifyRejectsExpiredFutureAndClock(t *testing.T) {
	k := newKeys(t)
	policy := policyOf(t, entry(k, "dev-issuer", "k1", nil))
	g := signed(t, k, nil)
	_, err := Verify(g, baseQuery(), policy, at(time.Hour)) // notAfter is exclusive
	wantCode(t, err, "PROVISIONAL_GRANT_EXPIRED")
	_, err = Verify(g, baseQuery(), policy, at(-2*time.Minute))
	wantCode(t, err, "PROVISIONAL_GRANT_NOT_YET_VALID")
	if _, err = Verify(g, baseQuery(), policy, at(-time.Minute)); err != nil { // notBefore inclusive
		t.Fatal(err)
	}
	_, err = Verify(g, baseQuery(), policy, func() time.Time { return time.Time{} })
	wantCode(t, err, "PROVISIONAL_GRANT_CLOCK_INVALID")
	_, err = Verify(g, baseQuery(), policy, func() time.Time { panic("boom") })
	wantCode(t, err, "PROVISIONAL_GRANT_CLOCK_INVALID")
}

func TestVerifyRejectsRevokedAndWindowedKeys(t *testing.T) {
	k := newKeys(t)
	g := signed(t, k, nil)
	revoked := policyOf(t, entry(k, "dev-issuer", "k1", func(e *hoststore.TrustedIssuer) { e.Revoked = true }))
	_, err := Verify(g, baseQuery(), revoked, at(0))
	wantCode(t, err, "HOST_STORE_KEY_REVOKED")
	past := ts(-time.Second)
	expired := policyOf(t, entry(k, "dev-issuer", "k1", func(e *hoststore.TrustedIssuer) { e.NotAfter = &past }))
	_, err = Verify(g, baseQuery(), expired, at(0))
	wantCode(t, err, "HOST_STORE_KEY_EXPIRED")
	future := ts(time.Second)
	early := policyOf(t, entry(k, "dev-issuer", "k1", func(e *hoststore.TrustedIssuer) { e.NotBefore = &future }))
	_, err = Verify(g, baseQuery(), early, at(0))
	wantCode(t, err, "HOST_STORE_KEY_NOT_YET_VALID")
}

func TestVerifyRejectsOtherIssuerKeyIDKeyAndApplication(t *testing.T) {
	k, other := newKeys(t), newKeys(t)
	g := signed(t, k, nil)
	cases := map[string]hoststore.TrustPolicy{
		"other issuer":      policyOf(t, entry(k, "someone-else", "k1", nil)),
		"other keyId":       policyOf(t, entry(k, "dev-issuer", "k2", nil)),
		"other application": policyOf(t, entry(k, "dev-issuer", "k1", func(e *hoststore.TrustedIssuer) { e.Applications = []string{"repair_intake"} })),
	}
	for name, p := range cases {
		_, err := Verify(g, baseQuery(), p, at(0))
		if err == nil || err.Error() != "HOST_STORE_ISSUER_NOT_TRUSTED" {
			t.Fatalf("%s: %v", name, err)
		}
	}
	// Same issuer and keyId, different key material.
	_, err := Verify(g, baseQuery(), policyOf(t, entry(other, "dev-issuer", "k1", nil)), at(0))
	wantCode(t, err, "HOST_STORE_SIGNATURE_INVALID")
}

func TestVerifyRejectsBindingMismatch(t *testing.T) {
	k := newKeys(t)
	policy := policyOf(t, entry(k, "dev-issuer", "k1", nil))
	g := signed(t, k, nil)
	for name, q := range map[string]Query{
		"application": {"repair_intake", "order_entry.submit_order", "editor", "order/42"},
		"capability":  {"order_entry", "order_entry.cancel_order", "editor", "order/42"},
		"subject":     {"order_entry", "order_entry.submit_order", "viewer", "order/42"},
		"resource":    {"order_entry", "order_entry.submit_order", "editor", "order/43"},
		"empty":       {},
	} {
		_, err := Verify(g, q, policy, at(0))
		if err == nil || err.Error() != "PROVISIONAL_GRANT_BINDING_MISMATCH" {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestSignAndVerifyRejectMalformedGrants(t *testing.T) {
	k := newKeys(t)
	policy := policyOf(t, entry(k, "dev-issuer", "k1", nil))
	bad := map[string]struct {
		mutate func(*Grant)
		code   string
	}{
		"over 24h":          {func(g *Grant) { g.NotBefore, g.NotAfter = ts(0), ts(24*time.Hour+time.Second) }, "PROVISIONAL_GRANT_PERIOD_TOO_LONG"},
		"exactly 24h ok?":   {nil, ""},
		"reversed":          {func(g *Grant) { g.NotBefore, g.NotAfter = ts(time.Hour), ts(0) }, "PROVISIONAL_GRANT_PERIOD_INVALID"},
		"empty period":      {func(g *Grant) { g.NotAfter = g.NotBefore }, "PROVISIONAL_GRANT_PERIOD_INVALID"},
		"missing notBefore": {func(g *Grant) { g.NotBefore = "" }, "PROVISIONAL_GRANT_PERIOD_INVALID"},
		"missing notAfter":  {func(g *Grant) { g.NotAfter = "" }, "PROVISIONAL_GRANT_PERIOD_INVALID"},
		"offset time":       {func(g *Grant) { g.NotAfter = t0.Add(time.Hour).In(time.FixedZone("x", 3600)).Format(time.RFC3339) }, "PROVISIONAL_GRANT_PERIOD_INVALID"},
		"zero epoch":        {func(g *Grant) { g.Epochs.Policy = 0 }, "PROVISIONAL_GRANT_INVALID"},
		"empty source":      {func(g *Grant) { g.Epochs.Source = " " }, "PROVISIONAL_GRANT_INVALID"},
		"empty subject":     {func(g *Grant) { g.SubjectID = "" }, "PROVISIONAL_GRANT_INVALID"},
		"empty resource":    {func(g *Grant) { g.Resource = "" }, "PROVISIONAL_GRANT_INVALID"},
		"padded capability": {func(g *Grant) { g.Capability = " order_entry.submit_order" }, "PROVISIONAL_GRANT_INVALID"},
		"newline":           {func(g *Grant) { g.SubjectID = "a\nb" }, "PROVISIONAL_GRANT_INVALID"},
	}
	for name, c := range bad {
		g := baseGrant()
		if name == "exactly 24h ok?" {
			g.NotBefore, g.NotAfter = ts(-time.Minute), ts(24*time.Hour-time.Minute)
			if _, err := Sign(g, k.priv); err != nil {
				t.Fatalf("exactly 24h must be allowed: %v", err)
			}
			continue
		}
		c.mutate(&g)
		_, err := Sign(g, k.priv)
		if err == nil || err.Error() != c.code {
			t.Fatalf("Sign %s: %v", name, err)
		}
	}
	// Verify must refuse the same structures even when someone signed them with a real key.
	g := signed(t, k, nil)
	g.Provisional = false
	_, err := Verify(g, baseQuery(), policy, at(0))
	wantCode(t, err, "PROVISIONAL_GRANT_INVALID")
	g = signed(t, k, nil)
	g.Schema = "tei.host-effect-grant/v0"
	_, err = Verify(g, baseQuery(), policy, at(0))
	wantCode(t, err, "PROVISIONAL_GRANT_INVALID")
	// A signer that bypasses Sign and signs a 48h document is still refused at read time.
	long := baseGrant()
	long.Schema, long.Provisional = Schema, true
	long.NotBefore, long.NotAfter = ts(-time.Minute), ts(48*time.Hour)
	payload, _ := SigningPayload(long)
	long.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(k.priv, payload))
	_, err = Verify(long, baseQuery(), policy, at(0))
	wantCode(t, err, "PROVISIONAL_GRANT_PERIOD_TOO_LONG")
	if _, err := Sign(baseGrant(), ed25519.PrivateKey("short")); err == nil {
		t.Fatal("short key accepted")
	}
}

func TestParseIsStrict(t *testing.T) {
	k := newKeys(t)
	g := signed(t, k, nil)
	raw, _ := json.Marshal(g)
	if _, err := Parse(raw); err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	m["extra"] = true
	withExtra, _ := json.Marshal(m)
	for name, doc := range map[string][]byte{
		"unknown member": withExtra,
		"trailing":       append(append([]byte{}, raw...), []byte(`{}`)...),
		"not json":       []byte("nope"),
		"too large":      []byte(`{"schema":"` + strings.Repeat("a", maxDocumentBytes) + `"}`),
	} {
		if _, err := Parse(doc); err == nil || err.Error() != "PROVISIONAL_GRANT_INVALID" {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestTrustFileRules(t *testing.T) {
	k := newKeys(t)
	// Duplicate key material under two entries would bypass a revocation.
	dup := trustJSON(t, TrustSchema, entry(k, "a", "k1", nil), entry(k, "b", "k2", nil))
	if _, err := hoststore.ParseTrustPolicyWithSchema(dup, TrustSchema); err == nil {
		t.Fatal("duplicate public key accepted")
	}
	// The Source Acceptance trust schema is not an effect-grant trust file, and back.
	acceptance := trustJSON(t, "tei.host-acceptance-trust/v0", entry(k, "a", "k1", nil))
	if _, err := hoststore.ParseTrustPolicyWithSchema(acceptance, TrustSchema); err == nil {
		t.Fatal("acceptance trust file accepted as effect-grant trust")
	}
	if _, err := hoststore.ParseTrustPolicy(trustJSON(t, TrustSchema, entry(k, "a", "k1", nil))); err == nil {
		t.Fatal("effect-grant trust file accepted as acceptance trust")
	}
	if _, err := hoststore.ParseTrustPolicyWithSchema(acceptance, ""); err == nil {
		t.Fatal("empty schema accepted")
	}
}

func TestErrorsNeverCarrySecretsOrPayload(t *testing.T) {
	k := newKeys(t)
	g := signed(t, k, nil)
	payload, _ := SigningPayload(g)
	seed := base64.StdEncoding.EncodeToString(k.priv.Seed())
	forged := g
	forged.SubjectID = "secret-subject-xyz"
	policies := []hoststore.TrustPolicy{
		policyOf(t, entry(k, "dev-issuer", "k1", nil)),
		policyOf(t, entry(k, "dev-issuer", "k1", func(e *hoststore.TrustedIssuer) { e.Revoked = true })),
		policyOf(t, entry(k, "x", "y", nil)),
	}
	queries := []Query{baseQuery(), {"order_entry", "c", "s", "r"}}
	for _, p := range policies {
		for _, c := range []Grant{g, forged} {
			for _, q := range queries {
				for _, d := range []time.Duration{0, 5 * time.Hour, -time.Hour} {
					_, err := Verify(c, q, p, at(d))
					if err == nil {
						continue
					}
					msg := err.Error()
					for _, leak := range []string{g.Signature, seed, string(payload), "secret-subject-xyz", "order/42", "editor", base64.StdEncoding.EncodeToString(k.pub)} {
						if strings.Contains(msg, leak) {
							t.Fatalf("error leaks %q: %s", leak, msg)
						}
					}
				}
			}
		}
	}
}

func writeFiles(t *testing.T, g Grant, trust []byte) (string, string) {
	t.Helper()
	dir := t.TempDir()
	gp, tp := filepath.Join(dir, "grant.json"), filepath.Join(dir, "trust.json")
	raw, _ := json.Marshal(g)
	if err := os.WriteFile(gp, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tp, trust, 0o600); err != nil {
		t.Fatal(err)
	}
	return gp, tp
}

func TestFileSourceVerifiesOnEveryRead(t *testing.T) {
	k := newKeys(t)
	g := signed(t, k, nil)
	good := trustJSON(t, TrustSchema, entry(k, "dev-issuer", "k1", nil))
	gp, tp := writeFiles(t, g, good)
	clock := t0
	s := FileSource{GrantPath: gp, TrustPath: tp, Now: func() time.Time { return clock }}
	if _, err := s.Read(context.Background(), baseQuery()); err != nil {
		t.Fatal(err)
	}
	// Revocation in the trust file applies at the very next Read (no restart).
	if err := os.WriteFile(tp, trustJSON(t, TrustSchema, entry(k, "dev-issuer", "k1", func(e *hoststore.TrustedIssuer) { e.Revoked = true })), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(context.Background(), baseQuery()); err == nil || err.Error() != "HOST_STORE_KEY_REVOKED" {
		t.Fatalf("revocation not applied: %v", err)
	}
	if err := os.WriteFile(tp, good, 0o600); err != nil {
		t.Fatal(err)
	}
	// Time passing applies too.
	clock = t0.Add(2 * time.Hour)
	if _, err := s.Read(context.Background(), baseQuery()); err == nil || err.Error() != "PROVISIONAL_GRANT_EXPIRED" {
		t.Fatalf("%v", err)
	}
	clock = t0
	// A tampered Grant file is refused.
	g.Resource = "order/99"
	raw, _ := json.Marshal(g)
	if err := os.WriteFile(gp, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(context.Background(), baseQuery()); err == nil {
		t.Fatal("tampered file accepted")
	}
	// Missing files, cancelled context and unset paths fail closed with fixed codes.
	if _, err := (FileSource{GrantPath: gp + ".missing", TrustPath: tp, Now: s.Now}).Read(context.Background(), baseQuery()); err == nil || err.Error() != "PROVISIONAL_GRANT_UNREADABLE" {
		t.Fatalf("%v", err)
	}
	if _, err := (FileSource{GrantPath: gp, TrustPath: tp + ".missing", Now: s.Now}).Read(context.Background(), baseQuery()); err == nil || err.Error() != "HOST_STORE_TRUST_UNREADABLE" {
		t.Fatalf("%v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Read(ctx, baseQuery()); err == nil {
		t.Fatal("cancelled context accepted")
	}
	if _, err := (FileSource{}).Read(context.Background(), baseQuery()); err == nil {
		t.Fatal("empty source accepted")
	}
}

func TestSigningPayloadGolden(t *testing.T) {
	g := baseGrant()
	g.Schema, g.Provisional = Schema, true
	got, err := SigningPayload(g)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"domain":"tei.host-effect-grant-provisional/v0","provisional":true,"issuer":"dev-issuer","keyId":"k1",` +
		`"applicationId":"order_entry","capability":"order_entry.submit_order","subjectId":"editor","resource":"order/42",` +
		`"epochs":{"source":"local-auth-store","subjectLink":10,"membership":20,"policy":30},` +
		`"notBefore":"2026-11-03T11:59:00Z","notAfter":"2026-11-03T13:00:00Z"}`
	if string(got) != want {
		t.Fatalf("signing payload changed:\n got %s\nwant %s", got, want)
	}
}

// A signature made over the same fields under another domain (none, the Source
// Acceptance domain, a near-miss) must never verify as an effect Grant.
func TestOtherDomainSignaturesAreNotEffectGrants(t *testing.T) {
	k := newKeys(t)
	policy := policyOf(t, entry(k, "dev-issuer", "k1", nil))
	for _, domain := range []string{"", "tei.host-acceptance-decision/v0", Schema + "x", "tei.host-effect-grant-provisional/v1"} {
		g := baseGrant()
		g.Schema, g.Provisional = Schema, true
		payload, err := json.Marshal(signedPayload{
			Domain: domain, Provisional: true, Issuer: g.Issuer, KeyID: g.KeyID, ApplicationID: g.ApplicationID,
			Capability: g.Capability, SubjectID: g.SubjectID, Resource: g.Resource, Epochs: g.Epochs,
			NotBefore: g.NotBefore, NotAfter: g.NotAfter,
		})
		if err != nil {
			t.Fatal(err)
		}
		g.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(k.priv, payload))
		if _, err := Verify(g, baseQuery(), policy, at(0)); err == nil || err.Error() != "HOST_STORE_SIGNATURE_INVALID" {
			t.Fatalf("domain %q: %v", domain, err)
		}
	}
	// A Source Acceptance Decision signature (same key) is not an effect Grant either.
	d := hoststore.Decision{Issuer: "dev-issuer"}
	decisionSigned, err := hoststore.Sign(d, "k1", k.priv)
	if err != nil {
		t.Fatal(err)
	}
	var ev struct{ Signature string }
	_ = json.Unmarshal(decisionSigned.Evidence, &ev)
	g := baseGrant()
	g.Schema, g.Provisional, g.Signature = Schema, true, ev.Signature
	if _, err := Verify(g, baseQuery(), policy, at(0)); err == nil || err.Error() != "HOST_STORE_SIGNATURE_INVALID" {
		t.Fatalf("decision signature accepted: %v", err)
	}
}

func TestFieldsRejectControlFormatAndOversizeText(t *testing.T) {
	k := newKeys(t)
	bad := map[string]string{
		"tab":         "a\tb",
		"escape":      "a\x1bb",
		"bidi":        "a‮b",
		"zero width":  "a​b",
		"nbsp":        "a b",
		"line sep":    "a b",
		"bad utf8":    "a\xffb",
		"private use": "ab",
		"oversize":    strings.Repeat("a", maxFieldBytes+1),
	}
	for name, v := range bad {
		for field, set := range map[string]func(*Grant){
			"subject":  func(g *Grant) { g.SubjectID = v },
			"resource": func(g *Grant) { g.Resource = v },
			"source":   func(g *Grant) { g.Epochs.Source = v },
		} {
			g := baseGrant()
			set(&g)
			if _, err := Sign(g, k.priv); err == nil || err.Error() != "PROVISIONAL_GRANT_INVALID" {
				t.Fatalf("%s/%s: %v", name, field, err)
			}
		}
	}
	g := baseGrant()
	g.SubjectID, g.Resource = "a b", "order/日本語"
	if _, err := Sign(g, k.priv); err != nil {
		t.Fatalf("inner space and non-ASCII letters must be allowed: %v", err)
	}
	if _, err := Sign(func() Grant { g := baseGrant(); g.Resource = strings.Repeat("a", maxFieldBytes); return g }(), k.priv); err != nil {
		t.Fatalf("max length must be allowed: %v", err)
	}
}

func TestPeriodHasALowerBound(t *testing.T) {
	k := newKeys(t)
	g := baseGrant()
	g.NotBefore, g.NotAfter = ts(0), ts(59*time.Second)
	if _, err := Sign(g, k.priv); err == nil || err.Error() != "PROVISIONAL_GRANT_PERIOD_TOO_SHORT" {
		t.Fatalf("%v", err)
	}
	g.NotAfter = ts(time.Minute)
	if _, err := Sign(g, k.priv); err != nil {
		t.Fatalf("exactly one minute must be allowed: %v", err)
	}
}

func TestParseRejectsDuplicateAndMiscasedKeys(t *testing.T) {
	k := newKeys(t)
	g := signed(t, k, nil)
	raw, _ := json.Marshal(g)
	s := string(raw)
	for name, doc := range map[string]string{
		"duplicate top":     strings.Replace(s, `"resource":"order/42"`, `"resource":"order/42","resource":"order/43"`, 1),
		"duplicate epochs":  strings.Replace(s, `"policy":30`, `"policy":30,"policy":31`, 1),
		"miscased top":      strings.Replace(s, `"subjectId"`, `"SubjectID"`, 1),
		"miscased epochs":   strings.Replace(s, `"membership"`, `"Membership"`, 1),
		"unknown in epochs": strings.Replace(s, `"policy":30`, `"policy":30,"extra":1`, 1),
		"array":             "[" + s + "]",
	} {
		if doc == s {
			t.Fatalf("%s: fixture did not change", name)
		}
		if _, err := Parse([]byte(doc)); err == nil || err.Error() != "PROVISIONAL_GRANT_INVALID" {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if _, err := Parse(raw); err != nil {
		t.Fatal(err)
	}
}
