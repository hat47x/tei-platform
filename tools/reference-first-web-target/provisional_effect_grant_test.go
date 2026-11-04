//go:build !tei_order_only && !tei_repair_only

package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tei/internal/security/effectcommit"
	"tei/internal/security/provisionaleffectgrant"
	"tei/internal/sourceacceptance"
	"tei/internal/sourceacceptance/hoststore"
	"tei/internal/sourceindex"
)

var provT0 = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

type provEnv struct {
	grantPath, trustPath string
	priv                 ed25519.PrivateKey
	clock                time.Time
}

func (e *provEnv) now() time.Time { return e.clock }

func newProvEnv(t *testing.T, mutate func(*provisionaleffectgrant.Grant)) *provEnv {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	g := provisionaleffectgrant.Grant{
		Issuer: "dev", KeyID: "k1", ApplicationID: "order_entry", Capability: "order_entry.submit_order",
		SubjectID: "test-only-logged-in-subject", Resource: "order/42",
		Epochs:    provisionaleffectgrant.Epochs{Source: "local-auth-store", SubjectLink: 1, Membership: 2, Policy: 3},
		NotBefore: provT0.Add(-time.Minute).Format(time.RFC3339), NotAfter: provT0.Add(time.Hour).Format(time.RFC3339),
	}
	if mutate != nil {
		mutate(&g)
	}
	signed, err := provisionaleffectgrant.Sign(g, priv)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	e := &provEnv{grantPath: filepath.Join(dir, "grant.json"), trustPath: filepath.Join(dir, "trust.json"), priv: priv, clock: provT0}
	raw, _ := json.Marshal(signed)
	if err := os.WriteFile(e.grantPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	e.writeTrust(t, func(*hoststore.TrustedIssuer) {}, pub)
	return e
}

func (e *provEnv) writeTrust(t *testing.T, mutate func(*hoststore.TrustedIssuer), pub ed25519.PublicKey) {
	t.Helper()
	if pub == nil {
		pub = e.priv.Public().(ed25519.PublicKey)
	}
	issuer := hoststore.TrustedIssuer{Issuer: "dev", KeyID: "k1", PublicKey: base64.StdEncoding.EncodeToString(pub), Applications: []string{"order_entry"}}
	mutate(&issuer)
	raw, _ := json.Marshal(hoststore.TrustPolicy{Schema: provisionaleffectgrant.TrustSchema, Issuers: []hoststore.TrustedIssuer{issuer}})
	if err := os.WriteFile(e.trustPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func (e *provEnv) flags() provisionalEffectGrantFlags {
	return provisionalEffectGrantFlags{grantPath: e.grantPath, trustPath: e.trustPath, developmentOnly: true}
}

func TestProvisionalFlagsAreNotSelectedByDefault(t *testing.T) {
	r, err := selectProvisionalEffectGrant(provisionalEffectGrantFlags{}, "order_entry", nil, false, time.Now)
	if r != nil || err != nil {
		t.Fatalf("default path selected a Reader: %v %v", r, err)
	}
}

func TestProvisionalFlagsRequireAllThreeAndNoConflict(t *testing.T) {
	e := newProvEnv(t, nil)
	full := e.flags()
	for name, f := range map[string]provisionalEffectGrantFlags{
		"no ack":      {grantPath: full.grantPath, trustPath: full.trustPath},
		"no trust":    {grantPath: full.grantPath, developmentOnly: true},
		"no grant":    {trustPath: full.trustPath, developmentOnly: true},
		"only ack":    {developmentOnly: true},
		"only grant":  {grantPath: full.grantPath},
		"only trust":  {trustPath: full.trustPath},
		"grant+trust": {grantPath: full.grantPath, trustPath: full.trustPath},
	} {
		r, err := selectProvisionalEffectGrant(f, "order_entry", nil, false, e.now)
		if r != nil || err == nil || !strings.HasPrefix(err.Error(), "PROVISIONAL_EFFECT_GRANT_FLAGS") {
			t.Fatalf("%s: %v %v", name, r, err)
		}
	}
	if r, err := selectProvisionalEffectGrant(full, "order_entry", &syntheticEffectReader{}, false, e.now); r != nil || err == nil || !strings.HasPrefix(err.Error(), "PROVISIONAL_EFFECT_GRANT_CONFLICT") {
		t.Fatalf("embedding Host Reader not protected: %v %v", r, err)
	}
	if r, err := selectProvisionalEffectGrant(full, "order_entry", nil, true, e.now); r != nil || err == nil {
		t.Fatalf("diagnostic bypass allowed: %v %v", r, err)
	}
	if r, err := selectProvisionalEffectGrant(full, "repair_intake", nil, false, e.now); r != nil || err == nil {
		t.Fatalf("wrong Application accepted: %v %v", r, err)
	}
}

func TestProvisionalSelectionVerifiesAtStartupAndWarns(t *testing.T) {
	e := newProvEnv(t, nil)
	var logs bytes.Buffer
	log.SetOutput(&logs)
	defer log.SetOutput(os.Stderr)
	r, err := selectProvisionalEffectGrant(e.flags(), "order_entry", nil, false, e.now)
	if err != nil || r == nil {
		t.Fatalf("%v %v", r, err)
	}
	// Selecting is silent; the warning is printed by announce(), which the
	// startup calls only after every gate has passed.
	if strings.Contains(logs.String(), "PROVISIONAL_EFFECT_GRANT_ACTIVE") {
		t.Fatalf("selection printed the warning too early: %s", logs.String())
	}
	r.(interface{ announce() }).announce()
	for _, want := range []string{"PROVISIONAL_EFFECT_GRANT_ACTIVE", "development-only", "not G2 evidence", "provisional-grant"} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("warning lacks %q: %s", want, logs.String())
		}
	}
	// Expired at startup.
	e.clock = provT0.Add(2 * time.Hour)
	if r, err := selectProvisionalEffectGrant(e.flags(), "order_entry", nil, false, e.now); r != nil || err == nil {
		t.Fatalf("expired Grant selected: %v %v", r, err)
	}
	// Capability other than the submit Command.
	other := newProvEnv(t, func(g *provisionaleffectgrant.Grant) { g.Capability = "order_entry.cancel_order" })
	if r, err := selectProvisionalEffectGrant(other.flags(), "order_entry", nil, false, other.now); r != nil || err == nil {
		t.Fatalf("other capability selected: %v %v", r, err)
	}
}

func provSourceFixture(t *testing.T) (*requestSourceReader, sourceindex.Index, sourceacceptance.Record) {
	t.Helper()
	member, err := sourceindex.NewArtifact("applications/order-entry/application.tei", []byte("application order_entry {}\n"))
	if err != nil {
		t.Fatal(err)
	}
	index := sourceindex.Index{Artifacts: []sourceindex.Artifact{member}}
	accepted := sourceacceptance.Record{
		ApplicationID: "order_entry", DecisionRef: "test-only-source-decision", AuthorityRevision: "test-only-source-authority",
		ModelRevision: sourceindex.ModelRevision(index), Artifacts: index.Artifacts,
	}
	return &requestSourceReader{record: accepted}, index, accepted
}

func TestProvisionalReaderAdmitsOnlyExactSubjectAndResourceAndReverifiesEachTime(t *testing.T) {
	e := newProvEnv(t, nil)
	var logs bytes.Buffer
	log.SetOutput(&logs)
	defer log.SetOutput(os.Stderr)
	reader, err := selectProvisionalEffectGrant(e.flags(), "order_entry", nil, false, e.now)
	if err != nil {
		t.Fatal(err)
	}
	source, index, accepted := provSourceFixture(t)
	check := func(subject, resource string) error {
		return requireCurrentHostEffectGrantFor(context.Background(), source, reader, "order_entry", subject,
			"order_entry.submit_order", resource, index, accepted)
	}
	if err := check("test-only-logged-in-subject", "order/42"); err != nil {
		t.Fatal(err)
	}
	if err := check("another-user", "order/42"); !errors.Is(err, errHostEffectGrantUnavailable) {
		t.Fatalf("other subject admitted: %v", err)
	}
	if err := check("test-only-logged-in-subject", "order/43"); !errors.Is(err, errHostEffectGrantUnavailable) {
		t.Fatalf("other resource admitted: %v", err)
	}
	if err := check("test-only-logged-in-subject", ""); !errors.Is(err, errHostEffectGrantUnavailable) {
		t.Fatalf("resource-less request admitted: %v", err)
	}
	// Revocation takes effect at the next effect, with no restart.
	e.writeTrust(t, func(i *hoststore.TrustedIssuer) { i.Revoked = true }, nil)
	if err := check("test-only-logged-in-subject", "order/42"); !errors.Is(err, errHostEffectGrantUnavailable) {
		t.Fatalf("revoked key still admitted: %v", err)
	}
	e.writeTrust(t, func(*hoststore.TrustedIssuer) {}, nil)
	if err := check("test-only-logged-in-subject", "order/42"); err != nil {
		t.Fatal(err)
	}
	// Expiry takes effect too.
	e.clock = provT0.Add(time.Hour)
	if err := check("test-only-logged-in-subject", "order/42"); !errors.Is(err, errHostEffectGrantUnavailable) {
		t.Fatalf("expired Grant admitted: %v", err)
	}
	// The refusal is logged by fixed code only.
	if !strings.Contains(logs.String(), "PROVISIONAL_EFFECT_GRANT_REFUSED") {
		t.Fatalf("no refusal logged: %s", logs.String())
	}
	seed := base64.StdEncoding.EncodeToString(e.priv.Seed())
	raw, _ := os.ReadFile(e.grantPath)
	var doc struct{ Signature string }
	_ = json.Unmarshal(raw, &doc)
	if strings.Contains(logs.String(), seed) || strings.Contains(logs.String(), doc.Signature) {
		t.Fatalf("log leaks key material or signature: %s", logs.String())
	}
}

type provSink struct{ got []effectcommit.Evidence }

func (s *provSink) Record(ev effectcommit.Evidence) { s.got = append(s.got, ev) }

func TestProvisionalGrantFeedsRunnerAndEvidenceIsMarkedProvisional(t *testing.T) {
	e := newProvEnv(t, nil)
	p := newFakeSubmit()
	sink := &provSink{}
	runner := effectcommit.Runner{Provider: p, Audit: provisionalAuditConfig(sink)}
	grantSource := newProvisionalSubmitGrantSource(
		provisionaleffectgrant.FileSource{GrantPath: e.grantPath, TrustPath: e.trustPath, Now: e.now},
		"order_entry", "test-only-logged-in-subject")
	store, err := newRunnerOrderSubmitStore(runner, grantSource, nil, &statusReader{})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Submit(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	if len(sink.got) != 1 {
		t.Fatalf("%+v", sink.got)
	}
	ev := sink.got[0]
	if ev.Provenance != effectcommit.ProvenanceProvisionalGrant || ev.Provenance == effectcommit.ProvenanceRealPG {
		t.Fatalf("provenance=%q", ev.Provenance)
	}
	if ev.Scope.Subject != "test-only-logged-in-subject" || ev.Scope.Resource != "order/42" || ev.Epochs.Policy != 3 {
		t.Fatalf("%+v", ev)
	}
	// Another order has no Grant: the Runner is not reached and nothing is written.
	writes := p.tx.writeCalls
	if err := store.Submit(context.Background(), 43); err == nil || p.tx.writeCalls != writes || len(sink.got) != 1 {
		t.Fatalf("grant for order/42 used for order/43: %v", err)
	}
	if got := provisionalAuditConfig(nil).Provenance; got != "provisional-grant" {
		t.Fatalf("%q", got)
	}
}

func TestProvisionalFlagsConflictBeforeAnythingStarts(t *testing.T) {
	e := newProvEnv(t, nil)
	base := []string{"-baseline-id", "B", "-baseline-revision", "r", "-target-revision", "t"}
	err := runCombinedWebTarget(context.Background(), nil, append(base, "-provisional-effect-grant", e.grantPath))
	if err == nil || !strings.HasPrefix(err.Error(), "PROVISIONAL_EFFECT_GRANT_FLAGS") {
		t.Fatalf("partial flags: %v", err)
	}
	err = runCombinedWebTarget(context.Background(), nil, append(base, "-diagnostic-unaccepted-source",
		"-provisional-effect-grant", e.grantPath, "-provisional-effect-grant-trust", e.trustPath, "-provisional-effect-grant-development-only"))
	if err == nil || !strings.HasPrefix(err.Error(), "SOURCE_ACCEPTANCE_DIAGNOSTIC_CONFLICT") {
		t.Fatalf("diagnostic combination: %v", err)
	}
}
