package hoststore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
)

// acceptAll stands in for a Host issuer policy ONLY in tests. It is not an
// authority: the package ships no verifier.
func acceptAll(context.Context, Decision) error { return nil }

func members(t *testing.T, content string) sourceindex.Index {
	t.Helper()
	a, err := sourceindex.NewArtifact("applications/order-entry/application.tei", []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	b, err := sourceindex.NewArtifact("semantic/sales/order.tei", []byte("data sales.order {}\n"))
	if err != nil {
		t.Fatal(err)
	}
	return sourceindex.Index{Artifacts: []sourceindex.Artifact{a, b}}
}

func decision(ref string, index sourceindex.Index) Decision {
	return Decision{
		Issuer: "test-issuer",
		Record: sourceacceptance.Record{
			ApplicationID: "order_entry", DecisionRef: ref, AuthorityRevision: "host-" + ref,
			ModelRevision: sourceindex.ModelRevision(index), Artifacts: index.Artifacts,
		},
	}
}

func openStore(t *testing.T, verifier DecisionVerifier) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "store"), verifier)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func verify(store *Store, index sourceindex.Index) error {
	_, err := sourceacceptance.VerifyCurrent(context.Background(), store, "order_entry", index)
	return err
}

func TestCurrentDecisionVerifiesAgainstLoadedSource(t *testing.T) {
	r1 := members(t, "application order_entry {}\n")
	store := openStore(t, acceptAll)
	if err := verify(store, r1); err == nil || !errors.Is(err, sourceacceptance.ErrCurrentAcceptanceUnavailable) {
		t.Fatalf("empty store must fail closed: %v", err)
	}
	if err := store.Ingest(context.Background(), decision("d1", r1)); err != nil {
		t.Fatal(err)
	}
	if err := verify(store, r1); err != nil {
		t.Fatalf("R1 rejected: %v", err)
	}
	// R0 (different bytes) and a semantically empty R2 (trailing newline)
	// share nothing but their meaning with R1; both must be refused.
	for name, content := range map[string]string{
		"R0": "application order_entry { changed }\n",
		"R2": "application order_entry {}\n\n",
	} {
		if err := verify(store, members(t, content)); err == nil || !strings.Contains(err.Error(), "SOURCE_FINGERPRINT_MISMATCH") {
			t.Fatalf("%s must be refused by fingerprint: %v", name, err)
		}
	}
}

func TestReadOnlyStoreCannotIngest(t *testing.T) {
	store := openStore(t, nil)
	err := store.Ingest(context.Background(), decision("d1", members(t, "x")))
	if !errors.Is(err, ErrVerifierRequired) {
		t.Fatalf("a Store without a Host verifier must not accept a Decision: %v", err)
	}
	if _, err := store.ReadCurrentAcceptedSource(context.Background(), "order_entry"); err == nil {
		t.Fatal("nothing was ingested")
	}
}

func TestVerifierRefusalStoresNothing(t *testing.T) {
	store := openStore(t, func(context.Context, Decision) error { return errors.New("issuer policy secret=hunter2") })
	err := store.Ingest(context.Background(), decision("d1", members(t, "x")))
	if !errors.Is(err, ErrIngestRefused) || strings.Contains(err.Error(), "hunter2") {
		t.Fatalf("refusal must be opaque: %v", err)
	}
	if _, err := store.ReadCurrentAcceptedSource(context.Background(), "order_entry"); err == nil {
		t.Fatal("a refused Decision became current")
	}
}

func TestRevokeIsSeenByTheNextRead(t *testing.T) {
	r1 := members(t, "application order_entry {}\n")
	store := openStore(t, acceptAll)
	if err := store.Ingest(context.Background(), decision("d1", r1)); err != nil {
		t.Fatal(err)
	}
	if err := verify(store, r1); err != nil {
		t.Fatal(err)
	}
	// A second Store handle on the same directory models another process.
	other, err := Open(store.dir, acceptAll)
	if err != nil {
		t.Fatal(err)
	}
	if err := verify(other, r1); err != nil {
		t.Fatalf("independent handle does not see the current Decision: %v", err)
	}
	if err := store.Revoke("order_entry", "d1"); err != nil {
		t.Fatal(err)
	}
	if err := verify(other, r1); !errors.Is(err, sourceacceptance.ErrCurrentAcceptanceUnavailable) {
		t.Fatalf("revocation not visible through an independent handle: %v", err)
	}
	if err := store.Revoke("order_entry", "d1"); !errors.Is(err, ErrNoCurrentAcceptance) {
		t.Fatalf("revoking a non-current Decision must fail: %v", err)
	}
}

func TestReplaceSupersedesAndRejectsTheOldWorld(t *testing.T) {
	r1 := members(t, "application order_entry {}\n")
	r3 := members(t, "application order_entry { v3 }\n")
	store := openStore(t, acceptAll)
	if err := store.Ingest(context.Background(), decision("d1", r1)); err != nil {
		t.Fatal(err)
	}
	if err := store.Ingest(context.Background(), decision("d3", r3)); err != nil {
		t.Fatal(err)
	}
	if err := verify(store, r3); err != nil {
		t.Fatalf("replacement not current: %v", err)
	}
	if err := verify(store, r1); err == nil || !strings.Contains(err.Error(), "SOURCE_FINGERPRINT_MISMATCH") {
		t.Fatalf("old World must be refused after replace: %v", err)
	}
	// The superseded Decision cannot be revoked as if it were current, and
	// its DecisionRef cannot be reused for different content.
	if err := store.Revoke("order_entry", "d1"); !errors.Is(err, ErrNoCurrentAcceptance) {
		t.Fatalf("superseded Decision revoked: %v", err)
	}
	if err := store.Ingest(context.Background(), decision("d1", members(t, "application order_entry { reuse }\n"))); !errors.Is(err, ErrIngestRefused) {
		t.Fatalf("DecisionRef reuse accepted: %v", err)
	}
}

func TestIngestRejectsInconsistentRecords(t *testing.T) {
	r1 := members(t, "application order_entry {}\n")
	store := openStore(t, acceptAll)
	cases := map[string]func(*Decision){
		"no issuer":       func(d *Decision) { d.Issuer = "" },
		"no decision ref": func(d *Decision) { d.Record.DecisionRef = "" },
		"no authority":    func(d *Decision) { d.Record.AuthorityRevision = "" },
		"no members":      func(d *Decision) { d.Record.Artifacts = nil },
		"forged revision": func(d *Decision) { d.Record.ModelRevision = sourceindex.Fingerprint([]byte("forged")) },
		"traversal member": func(d *Decision) {
			d.Record.Artifacts = []sourceindex.Artifact{{Path: "../x", Fingerprint: d.Record.Artifacts[0].Fingerprint}}
		},
		"duplicate members": func(d *Decision) { d.Record.Artifacts = append(d.Record.Artifacts, d.Record.Artifacts[0]) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			d := decision("d-"+name, r1)
			d.Record.Artifacts = append([]sourceindex.Artifact(nil), d.Record.Artifacts...)
			mutate(&d)
			if err := store.Ingest(context.Background(), d); !errors.Is(err, ErrIngestRefused) {
				t.Fatalf("accepted: %v", err)
			}
		})
	}
}

func TestCorruptOrForeignFilesFailClosedWithoutLeakingPaths(t *testing.T) {
	r1 := members(t, "application order_entry {}\n")
	store := openStore(t, acceptAll)
	if err := store.Ingest(context.Background(), decision("d1", r1)); err != nil {
		t.Fatal(err)
	}
	path := store.path("order_entry")
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string][]byte{
		"truncated":      good[:len(good)/2],
		"empty":          nil,
		"not json":       []byte("not json"),
		"wrong schema":   []byte(strings.Replace(string(good), schemaVersion, "other/v9", 1)),
		"other app file": []byte(strings.Replace(string(good), `"applicationId": "order_entry",`+"\n  \"entries\"", `"applicationId": "repair_intake",`+"\n  \"entries\"", 1)),
		"two current":    twoCurrent(t, good),
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(path, content, 0o600); err != nil {
				t.Fatal(err)
			}
			err := verify(store, r1)
			if !errors.Is(err, sourceacceptance.ErrCurrentAcceptanceUnavailable) {
				t.Fatalf("must fail closed: %v", err)
			}
			if strings.Contains(err.Error(), store.dir) || strings.Contains(err.Error(), filepath.Base(path)) {
				t.Fatalf("store path leaked: %v", err)
			}
		})
	}
}

func twoCurrent(t *testing.T, good []byte) []byte {
	t.Helper()
	var state file
	if err := json.Unmarshal(good, &state); err != nil {
		t.Fatal(err)
	}
	second := state.Entries[0]
	second.Decision.Record.DecisionRef = "other"
	state.Entries = append(state.Entries, second)
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestApplicationMismatchAndHostedIDsAreNotPaths(t *testing.T) {
	r1 := members(t, "application order_entry {}\n")
	store := openStore(t, acceptAll)
	if err := store.Ingest(context.Background(), decision("d1", r1)); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "repair_intake", "../order_entry", `..\order_entry`, "order_entry/../order_entry"} {
		if _, err := store.ReadCurrentAcceptedSource(context.Background(), id); err == nil {
			t.Fatalf("application %q resolved to a record", id)
		}
	}
	entries, err := os.ReadDir(store.dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("unexpected files in store: %d", len(entries))
	}
}

func TestReturnedRecordDoesNotAliasStoredState(t *testing.T) {
	r1 := members(t, "application order_entry {}\n")
	store := openStore(t, acceptAll)
	d := decision("d1", r1)
	if err := store.Ingest(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	record, err := store.ReadCurrentAcceptedSource(context.Background(), "order_entry")
	if err != nil {
		t.Fatal(err)
	}
	record.Artifacts[0].Path = "modified-by-caller"
	again, err := store.ReadCurrentAcceptedSource(context.Background(), "order_entry")
	if err != nil || again.Artifacts[0].Path == "modified-by-caller" {
		t.Fatalf("aliased: %v", err)
	}
}

func TestCancelledContextIsNotAnAcceptance(t *testing.T) {
	store := openStore(t, acceptAll)
	if err := store.Ingest(context.Background(), decision("d1", members(t, "x"))); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.ReadCurrentAcceptedSource(ctx, "order_entry"); err == nil {
		t.Fatal("cancelled read returned a record")
	}
}
