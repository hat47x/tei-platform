//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tei/internal/sourceacceptance"
	"tei/internal/sourceacceptance/hoststore"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// D9 (deep defence): the effect Grant path re-checks the Acceptance kind too.
func TestHostEffectGrantRefusesAChangedAcceptanceKind(t *testing.T) {
	member, _ := sourceindex.NewArtifact("applications/order-entry/application.tei", []byte("application order_entry {}\n"))
	index := sourceindex.Index{Artifacts: []sourceindex.Artifact{member}}
	accepted := sourceacceptance.Record{ApplicationID: "order_entry", DecisionRef: "d", AuthorityRevision: "a",
		ModelRevision: sourceindex.ModelRevision(index), Artifacts: index.Artifacts, AcceptanceKind: sourceacceptance.AcceptanceKindProvisional}
	source := &requestSourceReader{record: accepted}
	grants := &syntheticEffectReader{current: true}
	check := func(startup sourceacceptance.Record) error {
		return requireCurrentHostEffectGrant(context.Background(), source, grants,
			"order_entry", "subject", "order_entry.submit_order", index, startup)
	}
	if err := check(accepted); err != nil {
		t.Fatalf("unchanged kind refused: %v", err)
	}
	for _, kind := range []string{"", sourceacceptance.AcceptanceKindUnspecified} { // downgraded after startup
		source.record.AcceptanceKind = kind
		before := grants.calls
		if err := check(accepted); !errors.Is(err, errHostEffectGrantUnavailable) || grants.calls != before {
			t.Fatalf("kind %q: changed kind admitted (grant reads %d -> %d): %v", kind, before, grants.calls, err)
		}
	}
	// A hand-built startup Record without a kind equals the reported "unspecified".
	legacy := accepted
	legacy.AcceptanceKind = ""
	source.record.AcceptanceKind = ""
	if err := check(legacy); err != nil {
		t.Fatalf("unspecified vs unspecified refused: %v", err)
	}
}

// assertKindHeader checks the header on a health probe, an unknown path and a
// POST, not only /healthz.
func assertKindHeader(t *testing.T, client *http.Client, base, want string) {
	t.Helper()
	for _, c := range []struct{ method, path string }{{http.MethodGet, "/healthz"}, {http.MethodGet, "/no-such-path"}, {http.MethodPost, "/scenario/submit"}} {
		request, err := http.NewRequest(c.method, base+c.path, strings.NewReader("{}"))
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if got := response.Header.Get("X-TEI-Source-Acceptance-Kind"); got != want {
			t.Fatalf("%s %s: header %q, want %q", c.method, c.path, got, want)
		}
	}
}

// A Decision signed before the marker existed serves, but is announced and
// labelled "unspecified", never as a formal Acceptance.
func TestShippedCLIAnnouncesALegacyUnmarkedAcceptanceAsUnspecified(t *testing.T) {
	repoRoot := filepath.Dir(filepath.Dir(referenceWorkspacePath(t)))
	web := buildShipped(t, repoRoot, "./tools/reference-first-web-target", "first-web-target")
	admin := buildShipped(t, repoRoot, "./tools/reference-host-acceptance", "host-acceptance")
	r1 := workspaceWithUnitPriceRule(t)
	dir := t.TempDir()
	key, trust, store, decision := filepath.Join(dir, "issuer.key"), filepath.Join(dir, "trust.json"), filepath.Join(dir, "store"), filepath.Join(dir, "decision.json")
	trustJSON, err := runTool(t, admin, "keygen", "-private-key", key, "-issuer", "test-issuer", "-key-id", "k1", "-application", "order_entry")
	if err != nil {
		t.Fatalf("keygen: %v %s", err, trustJSON)
	}
	if err := os.WriteFile(trust, []byte(trustJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	// Sign the way the pre-D9 tool did: no acceptanceKind (v0 payload).
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(mustRead(t, key))))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(r1)
	if err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndexFromLoad(r1, loaded)
	if err != nil {
		t.Fatal(err)
	}
	d, err := hoststore.Sign(hoststore.Decision{Issuer: "test-issuer", Record: hoststore.RecordForIndex("order_entry", "legacy-1", "legacy-rev", index)},
		"k1", ed25519.NewKeyFromSeed(seed))
	if err != nil || d.AcceptanceKind != "" {
		t.Fatalf("legacy sign: %v kind=%q", err, d.AcceptanceKind)
	}
	encoded, _ := json.Marshal(d)
	if err := os.WriteFile(decision, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := runTool(t, admin, "ingest", "-store", store, "-trust", trust, "-decision", decision); err != nil || !strings.Contains(out, "ACCEPTANCE_KIND=unspecified") {
		t.Fatalf("ingest: %v %s", err, out)
	}
	if out, err := runTool(t, admin, "show", "-store", store, "-trust", trust, "-application", "order_entry"); err != nil || !strings.Contains(out, "ACCEPTANCE_KIND=unspecified") {
		t.Fatalf("show: %v %s", err, out)
	}

	address := freeAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	command := exec.CommandContext(ctx, web, "-workspace", r1, "-application", "order_entry", "-baseline-id", "B",
		"-baseline-revision", testBaselineRevision, "-target-revision", testTargetRevision, "-listen", address,
		"-host-acceptance-store", store, "-host-acceptance-trust", trust)
	command.Dir = repoRoot
	logfile, err := os.CreateTemp(t.TempDir(), "web-*.log")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	command.Stdout, command.Stderr = logfile, logfile
	if err := command.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	defer func() { cancel(); <-done; logfile.Close() }()
	base := "http://" + address
	client := &http.Client{Timeout: 5 * time.Second}
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		if response, err := client.Get(base + "/healthz"); err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("legacy-accepted Source did not serve: %s", mustRead(t, logfile.Name()))
		}
	}
	logs := string(mustRead(t, logfile.Name()))
	if !strings.Contains(logs, "SOURCE_ACCEPTANCE_UNSPECIFIED") || strings.Contains(logs, "SOURCE_ACCEPTANCE_PROVISIONAL") ||
		!strings.Contains(logs, "not a formal human Acceptance") {
		t.Fatalf("legacy acceptance not announced as unspecified: %s", logs)
	}
	assertKindHeader(t, client, base, "unspecified")
}
