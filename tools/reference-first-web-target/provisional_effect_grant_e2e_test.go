//go:build !tei_order_only && !tei_repair_only

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"tei/internal/security/effectcommit"
	"tei/internal/security/provisionaleffectgrant"
	"tei/internal/sourceacceptance"
	"tei/internal/sourceacceptance/hoststore"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// writeGrant re-signs the Grant file with the environment's key, valid around
// the REAL clock (the shipped flag path always reads time.Now).
func (e *provEnv) writeGrant(t *testing.T, subject, resource string) {
	t.Helper()
	g := provisionaleffectgrant.Grant{
		Issuer: "dev", KeyID: "k1", ApplicationID: "order_entry", Capability: "order_entry.submit_order",
		SubjectID: subject, Resource: resource,
		Epochs:    provisionaleffectgrant.Epochs{Source: "local-auth-store", SubjectLink: 1, Membership: 2, Policy: 3},
		NotBefore: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
		NotAfter:  time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
	}
	signed, err := provisionaleffectgrant.Sign(g, e.priv)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(signed)
	if err := os.WriteFile(e.grantPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// startProvisionalWebTarget runs the REAL runCombinedWebTarget (real flag
// parsing, Source gate, hostEffect wiring) in this process with the reference
// workspace accepted by a test-only Source Reader. Only the Source Reader is
// injected; the effect Grant Reader comes from the three flags.
func startProvisionalWebTarget(t *testing.T, extra []string) (base string, logs *lockedBuffer, stop func()) {
	t.Helper()
	root := filepath.Dir(filepath.Dir(referenceWorkspacePath(t)))
	workspacePath := referenceWorkspacePath(t)
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndexFromLoad(workspacePath, loaded)
	if err != nil {
		t.Fatal(err)
	}
	record := sourceacceptance.Record{
		ApplicationID: "order_entry", DecisionRef: "test-only-synthetic-decision",
		AuthorityRevision: "test-only-no-human-or-host-authority",
		ModelRevision:     sourceindex.ModelRevision(index), Artifacts: index.Artifacts,
	}
	publication := filepath.Join(t.TempDir(), "publication.json")
	raw, _ := json.Marshal(record)
	if err := os.WriteFile(publication, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	reservation, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := reservation.Addr().String()
	_ = reservation.Close()
	logs = &lockedBuffer{}
	log.SetOutput(logs)
	args := append([]string{
		"-workspace", workspacePath, "-application", "order_entry",
		"-baseline-id", "B", "-baseline-revision", testBaselineRevision, "-target-revision", testTargetRevision,
		"-legacy-users", filepath.Join(root, "reference", "legacy-auth", "users.json"),
		"-legacy-authorization", filepath.Join(root, "reference", "legacy-auth", "authorization.json"),
		"-listen", address,
	}, extra...)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runCombinedWebTarget(ctx, syntheticStartupReader{publication: publication}, args)
	}()
	stop = func() {
		cancel()
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			t.Error("Web Target did not stop")
		}
		log.SetOutput(os.Stderr)
	}
	base = "http://" + address
	client := &http.Client{Timeout: 5 * time.Second}
	// Generous bounds: under a parallel `go test ./...` the first start can be slow.
	for deadline := time.Now().Add(60 * time.Second); ; {
		select {
		case err := <-done:
			log.SetOutput(os.Stderr)
			t.Fatalf("Web Target exited before serving: %v", err)
		default:
		}
		if response, err := client.Get(base + "/healthz"); err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return base, logs, stop
			}
		}
		if time.Now().After(deadline) {
			stop()
			t.Fatal("Web Target did not publish HTTP within the bounded deadline")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func postSubmit(t *testing.T, base, loginID, credential string) (int, string) {
	t.Helper()
	client := &http.Client{Timeout: 15 * time.Second}
	response, err := client.Post(base+"/scenario/submit", "application/json", strings.NewReader(
		`{"loginId":"`+loginID+`","credential":"`+credential+`","from":"2026-09-18T08:30:00Z","row":0}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(body)
}

// The whole wiring, through the real flags and the real HTTP Command: the Grant
// must be bound to the authenticated subject AND to order/1001, and is
// re-verified at every effect (trust revocation applies without a restart).
func TestProvisionalFlagsGateTheHTTPCommandEndToEnd(t *testing.T) {
	e := newProvEnv(t, nil)
	e.writeGrant(t, "subject-order-editor", "order/1001")
	base, logs, stop := startProvisionalWebTarget(t, []string{
		"-provisional-effect-grant", e.grantPath, "-provisional-effect-grant-trust", e.trustPath,
		"-provisional-effect-grant-development-only"})
	defer stop()
	if strings.Count(logs.String(), "PROVISIONAL_EFFECT_GRANT_ACTIVE") != 1 {
		t.Fatalf("expected one ACTIVE warning after a successful start: %s", logs.String())
	}
	denied := func(name string) {
		t.Helper()
		status, body := postSubmit(t, base, "legacy-order-editor", "editor-secret")
		if status != http.StatusUnprocessableEntity || !strings.Contains(body, "HOST_EFFECT_GRANT_UNAVAILABLE") ||
			strings.Contains(body, "editor-secret") {
			t.Fatalf("%s: want 422 HOST_EFFECT_GRANT_UNAVAILABLE, got %d %s", name, status, body)
		}
	}
	// Another resource: a Grant for order/1002 must not allow order/1001.
	e.writeGrant(t, "subject-order-editor", "order/1002")
	denied("other resource")
	// Another subject.
	e.writeGrant(t, "subject-order-viewer", "order/1001")
	denied("other subject")
	// The exact Grant, but its issuer key is revoked in the trust file.
	e.writeGrant(t, "subject-order-editor", "order/1001")
	e.writeTrust(t, func(i *hoststore.TrustedIssuer) { i.Revoked = true }, nil)
	denied("revoked key")
	// Back to a trusted key: the same running process admits the Command.
	e.writeTrust(t, func(*hoststore.TrustedIssuer) {}, nil)
	status, body := postSubmit(t, base, "legacy-order-editor", "editor-secret")
	if status != http.StatusOK || !strings.Contains(body, `"completed"`) || strings.Contains(body, "editor-secret") {
		t.Fatalf("exact Grant was not admitted: %d %s", status, body)
	}
}

// A start that fails before the Target is published must not announce that the
// development-only Grant is active.
func TestProvisionalWarningIsNotPrintedWhenStartupFails(t *testing.T) {
	e := newProvEnv(t, nil)
	e.writeGrant(t, "subject-order-editor", "order/1001")
	logs := &lockedBuffer{}
	log.SetOutput(logs)
	defer log.SetOutput(os.Stderr)
	root := filepath.Dir(filepath.Dir(referenceWorkspacePath(t)))
	// No Source Reader: the Source gate refuses after the Grant flags are accepted.
	err := runCombinedWebTarget(context.Background(), nil, []string{
		"-workspace", referenceWorkspacePath(t), "-baseline-id", "B", "-baseline-revision", testBaselineRevision,
		"-target-revision", testTargetRevision,
		"-legacy-users", filepath.Join(root, "reference", "legacy-auth", "users.json"),
		"-legacy-authorization", filepath.Join(root, "reference", "legacy-auth", "authorization.json"),
		"-provisional-effect-grant", e.grantPath, "-provisional-effect-grant-trust", e.trustPath,
		"-provisional-effect-grant-development-only"})
	if err == nil || !strings.Contains(err.Error(), "SOURCE_ACCEPTANCE_READER_REQUIRED") {
		t.Fatalf("expected the Source gate to refuse: %v", err)
	}
	if strings.Contains(logs.String(), "PROVISIONAL_EFFECT_GRANT_ACTIVE") {
		t.Fatalf("a failed start claimed the provisional Grant is active: %s", logs.String())
	}
}

func TestProvisionalStoreConstructorFixesProvenance(t *testing.T) {
	e := newProvEnv(t, nil)
	e.writeGrant(t, "subject-order-editor", "order/1001")
	sink := &provSink{}
	source := provisionaleffectgrant.FileSource{GrantPath: e.grantPath, TrustPath: e.trustPath}
	if _, err := newProvisionalRunnerOrderSubmitStore(nil, source, "order_entry", "subject-order-editor", sink, nil, &statusReader{}); err == nil {
		t.Fatal("nil provider accepted")
	}
	if _, err := newProvisionalRunnerOrderSubmitStore(newFakeSubmit(), source, "order_entry", "subject-order-editor", nil, nil, &statusReader{}); err == nil {
		t.Fatal("a store without an Evidence sink (so without the fixed Provenance) was accepted")
	}
	store, err := newProvisionalRunnerOrderSubmitStore(newFakeSubmit(), source, "order_entry", "subject-order-editor", sink, nil, &statusReader{})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Submit(context.Background(), 1001); err != nil {
		t.Fatal(err)
	}
	if len(sink.got) != 1 || sink.got[0].Provenance != effectcommit.ProvenanceProvisionalGrant {
		t.Fatalf("%+v", sink.got)
	}
	// Nothing else in the package may pair a provisional Grant source with another Runner.
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || f == "provisional_effect_grant.go" {
			continue
		}
		src, _ := os.ReadFile(f)
		if bytes.Contains(src, []byte("newProvisionalSubmitGrantSource(")) {
			t.Fatalf("%s builds a provisional grant source outside the fixed-Provenance constructors", f)
		}
	}
}
