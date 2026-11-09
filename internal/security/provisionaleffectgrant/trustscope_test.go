package provisionaleffectgrant

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"tei/internal/sourceacceptance/hoststore"
)

const cap1 = "order_entry.submit_order"

const outOfScope = "PROVISIONAL_GRANT_OUT_OF_TRUST_SCOPE"

func scopedTrust(t *testing.T, k keys, caps, res []string) Trust {
	t.Helper()
	raw, err := json.Marshal(TrustFile{Schema: TrustSchema, Issuers: []ScopedIssuer{{
		TrustedIssuer: entry(k, "dev-issuer", "k1", nil), Capabilities: caps, ResourcePatterns: res,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	tr, err := ParseTrust(raw)
	if err != nil {
		t.Fatalf("%s: %v", raw, err)
	}
	return tr.WithClock(at(0))
}

func verifyWith(t *testing.T, k keys, tr Trust, capability, resource string) error {
	t.Helper()
	g := signed(t, k, func(g *Grant) { g.Capability, g.Resource = capability, resource })
	q := baseQuery()
	q.Capability, q.Resource = capability, resource
	_, err := VerifyScoped(g, q, tr, at(0))
	return err
}

func TestScopeAllowsInsideAndRejectsOutside(t *testing.T) {
	k := newKeys(t)
	tr := scopedTrust(t, k, []string{cap1}, []string{"order/*", "invoice/7"})
	if err := verifyWith(t, k, tr, cap1, "order/42"); err != nil {
		t.Fatal(err)
	}
	if err := verifyWith(t, k, tr, cap1, "invoice/7"); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string][2]string{
		"other capability":      {"order_entry.cancel_order", "order/42"},
		"capability case":       {"Order_entry.submit_order", "order/42"},
		"other resource":        {cap1, "customer/1"},
		"resource case":         {cap1, "Order/42"},
		"exact is not prefix":   {cap1, "invoice/70"},
		"wildcard not sibling":  {cap1, "orders/1"},
		"wildcard needs a rest": {cap1, "order/"},
		"wildcard no bare stem": {cap1, "order"},
		"dot segment":           {cap1, "order/../customer/1"},
		"substring only":        {cap1, "x/order/42"},
	} {
		if err := verifyWith(t, k, tr, c[0], c[1]); err == nil || err.Error() != outOfScope {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestExactPatternIsCaseSensitiveAndWildcardNeedsOneChar(t *testing.T) {
	k := newKeys(t)
	exact := scopedTrust(t, k, nil, []string{"invoice/7"})
	if err := verifyWith(t, k, exact, cap1, "invoice/7"); err != nil {
		t.Fatal(err)
	}
	if err := verifyWith(t, k, exact, cap1, "Invoice/7"); err == nil || err.Error() != outOfScope {
		t.Fatalf("exact pattern folded case: %v", err)
	}
	wild := scopedTrust(t, k, nil, []string{"order/*"})
	if err := verifyWith(t, k, wild, cap1, "order/x"); err != nil {
		t.Fatalf("one char after prefix: %v", err)
	}
	if err := verifyWith(t, k, wild, cap1, "order/"); err == nil || err.Error() != outOfScope {
		t.Fatalf("boundary order/: %v", err)
	}
	if err := verifyWith(t, k, wild, cap1, "Order/x"); err == nil {
		t.Fatal("wildcard prefix folded case")
	}
}

func TestMissingScopeRecordFailsClosed(t *testing.T) {
	k := newKeys(t)
	// A Trust assembled without ParseTrust has no scope record for the key.
	bare := Trust{Policy: policyOf(t, entry(k, "dev-issuer", "k1", nil))}
	if _, err := VerifyScoped(signed(t, k, nil), baseQuery(), bare, at(0)); err == nil || err.Error() != outOfScope {
		t.Fatalf("missing scope record accepted: %v", err)
	}
	if _, err := VerifyScoped(signed(t, k, nil), baseQuery(), Trust{Policy: bare.Policy, scopes: map[string]Scope{}}, at(0)); err == nil || err.Error() != outOfScope {
		t.Fatalf("empty scope map accepted: %v", err)
	}
}

func TestScopeExactPatternBoundaries(t *testing.T) {
	k := newKeys(t)
	tr := scopedTrust(t, k, nil, []string{"order/1"})
	if err := verifyWith(t, k, tr, cap1, "order/1"); err != nil {
		t.Fatal(err)
	}
	for _, r := range []string{"order/10", "xorder/1", "order/"} {
		if err := verifyWith(t, k, tr, cap1, r); err == nil || err.Error() != outOfScope {
			t.Errorf("%q: %v", r, err)
		}
	}
	if validResourcePattern("order/../x") || validResourcePattern("./x") {
		t.Fatal("dot segment pattern accepted")
	}
}

func TestScopeDimensionsAreIndependent(t *testing.T) {
	k := newKeys(t)
	capOnly := scopedTrust(t, k, []string{cap1}, nil)
	if err := verifyWith(t, k, capOnly, cap1, "anything/at/all"); err != nil {
		t.Fatal(err)
	}
	if err := verifyWith(t, k, capOnly, "other.cap", "order/42"); err == nil {
		t.Fatal("capability not checked")
	}
	resOnly := scopedTrust(t, k, nil, []string{"order/*"})
	if err := verifyWith(t, k, resOnly, "any.capability", "order/42"); err != nil {
		t.Fatal(err)
	}
	if err := verifyWith(t, k, resOnly, "any.capability", "x/42"); err == nil {
		t.Fatal("resource not checked")
	}
	// A capability inside the list but a resource outside is refused, and the
	// reverse: both dimensions must hold.
	both := scopedTrust(t, k, []string{cap1}, []string{"order/*"})
	if err := verifyWith(t, k, both, cap1, "x/1"); err == nil {
		t.Fatal("resource outside accepted")
	}
	if err := verifyWith(t, k, both, "other.cap", "order/1"); err == nil {
		t.Fatal("capability outside accepted")
	}
}

func TestUnscopedKeyKeepsOldBehaviour(t *testing.T) {
	k := newKeys(t)
	tr := scopedTrust(t, k, nil, nil)
	if s, ok := tr.ScopeFor("dev-issuer", "k1"); !ok || s.Restricted() {
		t.Fatalf("%+v %v", s, ok)
	}
	if err := verifyWith(t, k, tr, "whatever.cap", "whatever/resource"); err != nil {
		t.Fatal(err)
	}
	g := signed(t, k, nil)
	if _, err := Verify(g, baseQuery(), tr.Policy, at(0)); err != nil {
		t.Fatal(err)
	}
}

func TestScopeIsPerKeyAndOtherApplicationStillRefused(t *testing.T) {
	k1, k2 := newKeys(t), newKeys(t)
	raw, _ := json.Marshal(TrustFile{Schema: TrustSchema, Issuers: []ScopedIssuer{
		{TrustedIssuer: entry(k1, "dev-issuer", "k1", nil), ResourcePatterns: []string{"order/*"}},
		{TrustedIssuer: entry(k2, "dev-issuer", "k2", nil)},
	}})
	tr, err := ParseTrust(raw)
	if err != nil {
		t.Fatal(err)
	}
	tr = tr.WithClock(at(0))
	q := baseQuery()
	q.Resource = "customer/1"
	g2 := signed(t, k2, func(g *Grant) { g.KeyID, g.Resource = "k2", "customer/1" })
	if _, err := VerifyScoped(g2, q, tr, at(0)); err != nil {
		t.Fatalf("unscoped k2: %v", err)
	}
	g1 := signed(t, k1, func(g *Grant) { g.Resource = "customer/1" })
	if _, err := VerifyScoped(g1, q, tr, at(0)); err == nil || err.Error() != outOfScope {
		t.Fatalf("%v", err)
	}
	other := Trust{Policy: policyOf(t, entry(k1, "dev-issuer", "k1", func(e *hoststore.TrustedIssuer) { e.Applications = []string{"repair_intake"} })), scopes: map[string]Scope{}}
	if _, err := VerifyScoped(signed(t, k1, nil), baseQuery(), other, at(0)); err == nil || err.Error() != "HOST_STORE_ISSUER_NOT_TRUSTED" {
		t.Fatalf("%v", err)
	}
}

func TestForgeryOnScopedKeyReportsSignatureNotScope(t *testing.T) {
	k, imposter := newKeys(t), newKeys(t)
	tr := scopedTrust(t, k, []string{cap1}, []string{"order/*"})
	g := signed(t, imposter, func(g *Grant) { g.Resource = "customer/1" })
	q := baseQuery()
	q.Resource = "customer/1"
	if _, err := VerifyScoped(g, q, tr, at(0)); err == nil || err.Error() != "HOST_STORE_SIGNATURE_INVALID" {
		t.Fatalf("%v", err)
	}
}

func TestScopedTrustParseIsStrict(t *testing.T) {
	k := newKeys(t)
	pub := base64.StdEncoding.EncodeToString(k.pub)
	doc := func(member string) string {
		return `{"schema":"` + TrustSchema + `","issuers":[{"issuer":"i","keyId":"k","publicKey":"` + pub + `","applications":["a"]` + member + `}]}`
	}
	long := strings.Repeat("x", 300)
	many := make([]string, 65)
	for i := range many {
		many[i] = "r/" + strings.Repeat("a", i+1)
	}
	over, _ := json.Marshal(many)
	max, _ := json.Marshal(many[:64])
	bad := map[string]string{
		"empty caps":          `,"capabilities":[]`,
		"null caps":           `,"capabilities":null`,
		"empty res":           `,"resourcePatterns":[]`,
		"caps not array":      `,"capabilities":"a"`,
		"caps not strings":    `,"capabilities":[1]`,
		"empty string cap":    `,"capabilities":[""]`,
		"blank cap":           `,"capabilities":[" "]`,
		"control cap":         `,"capabilities":["a\u0007b"]`,
		"bidi cap":            `,"capabilities":["a‮b"]`,
		"star cap":            `,"capabilities":["a*"]`,
		"long cap":            `,"capabilities":["` + long + `"]`,
		"dup cap":             `,"capabilities":["a","a"]`,
		"bare star":           `,"resourcePatterns":["*"]`,
		"star in middle":      `,"resourcePatterns":["a*b"]`,
		"nested star":         `,"resourcePatterns":["order/**"]`,
		"two stars":           `,"resourcePatterns":["a/*/*"]`,
		"stem without slash":  `,"resourcePatterns":["order*"]`,
		"leading star":        `,"resourcePatterns":["*/x"]`,
		"dotdot":              `,"resourcePatterns":["order/../*"]`,
		"dot":                 `,"resourcePatterns":["./x"]`,
		"empty pattern":       `,"resourcePatterns":[""]`,
		"control pattern":     `,"resourcePatterns":["a\tb"]`,
		"dup pattern":         `,"resourcePatterns":["a/*","a/*"]`,
		"too many":            `,"resourcePatterns":` + string(over),
		"dup member":          `,"capabilities":["a"],"capabilities":["b"]`,
		"miscased member":     `,"Capabilities":["a"]`,
		"unknown member":      `,"resourcePattern":["a"]`,
		"unknown but similar": `,"capability":["a"]`,
	}
	for name, member := range bad {
		if _, err := ParseTrust([]byte(doc(member))); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	good := map[string]string{
		"none":   ``,
		"caps":   `,"capabilities":["a.b","c"]`,
		"exact":  `,"resourcePatterns":["order/42"]`,
		"prefix": `,"resourcePatterns":["order/*","x/y/*"]`,
		"both":   `,"capabilities":["a"],"resourcePatterns":["o/*"]`,
		"max 64": `,"resourcePatterns":` + string(max),
	}
	for name, member := range good {
		if _, err := ParseTrust([]byte(doc(member))); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	// The Source Acceptance parser still refuses these members.
	acc := `{"schema":"tei.host-acceptance-trust/v0","issuers":[{"issuer":"i","keyId":"k","publicKey":"` + pub + `","applications":["a"],"capabilities":["a"]}]}`
	if _, err := hoststore.ParseTrustPolicy([]byte(acc)); err == nil {
		t.Fatal("acceptance trust accepted a scope member")
	}
	if _, err := ParseTrust([]byte(strings.Replace(doc(""), TrustSchema, "tei.host-acceptance-trust/v0", 1))); err == nil {
		t.Fatal("wrong schema")
	}
	if _, err := ParseTrust([]byte(doc("") + " {}")); err == nil {
		t.Fatal("trailing data")
	}
	if _, err := ParseTrust([]byte(`{"schema":"` + TrustSchema + `","schema":"` + TrustSchema + `","issuers":[]}`)); err == nil {
		t.Fatal("dup top key")
	}
}

func TestFileSourceEnforcesScopeOnEveryRead(t *testing.T) {
	k := newKeys(t)
	g := signed(t, k, nil)
	mk := func(pattern string) []byte {
		raw, _ := json.Marshal(TrustFile{Schema: TrustSchema, Issuers: []ScopedIssuer{{TrustedIssuer: entry(k, "dev-issuer", "k1", nil), ResourcePatterns: []string{pattern}}}})
		return raw
	}
	gp, tp := writeFiles(t, g, mk("order/*"))
	s := FileSource{GrantPath: gp, TrustPath: tp, Now: at(0)}
	if _, err := s.Read(t.Context(), baseQuery()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tp, mk("invoice/*"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(t.Context(), baseQuery()); err == nil || err.Error() != outOfScope {
		t.Fatalf("%v", err)
	}
}
