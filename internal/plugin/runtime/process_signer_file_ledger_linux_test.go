//go:build linux

package runtime

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestLinuxFileSignerOperationLedgerRestartAndReplay(t *testing.T) {
	directory := t.TempDir()
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	ledger, err := NewLinuxFileSignerOperationLedger(directory)
	if err != nil {
		t.Fatal(err)
	}
	op := LinuxSignerOperationReservation{
		HostID: "host/reference", OperationID: "op-one",
		Principal: "host-admin", SignerRef: "signed/tool", Action: "allow",
		KeySHA256: linuxSignerKeyFingerprint(make(ed25519.PublicKey, ed25519.PublicKeySize)),
	}
	ok, err := ledger.ReserveSignerOperation(context.Background(), op)
	if !ok || err != nil {
		t.Fatalf("first durable reservation failed: %v", err)
	}
	// Reconstructing the ledger represents a new process opening the same
	// durable files; no in-memory deduplication state is shared.
	restarted, err := NewLinuxFileSignerOperationLedger(directory)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := restarted.ReserveSignerOperation(context.Background(), op); ok || err != nil {
		t.Fatalf("replayed reservation passed a new ledger instance: ok=%v err=%v", ok, err)
	}
	changed := op
	changed.Action, changed.KeySHA256 = "revoke", ""
	if ok, err := restarted.ReserveSignerOperation(context.Background(), changed); ok || err != nil {
		t.Fatalf("changed payload bypassed reserved ID: ok=%v err=%v", ok, err)
	}
	secondHost := op
	secondHost.HostID = "host/other"
	if ok, err := restarted.ReserveSignerOperation(context.Background(), secondHost); !ok || err != nil {
		t.Fatalf("different Host was not isolated: ok=%v err=%v", ok, err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 2 {
		t.Fatalf("wrong number of reservation records: count=%d err=%v", len(entries), err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") || strings.Contains(entry.Name(), "op-one") {
			t.Fatalf("operation identifier leaked into the filename: %q", entry.Name())
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var evidence LinuxSignerOperationReservation
		if err := json.Unmarshal(data, &evidence); err != nil {
			t.Fatalf("reservation was not valid recovery evidence: %v", err)
		}
		if evidence.OperationID != "op-one" || evidence.Principal != "host-admin" {
			t.Fatalf("stored operation does not match: %+v", evidence)
		}
	}
}

func TestLinuxFileSignerOperationLedgerConcurrentInstances(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	op := LinuxSignerOperationReservation{
		HostID: "host/reference", OperationID: "op-concurrent",
		Principal: "host-admin", SignerRef: "signed/tool", Action: "revoke",
	}
	const callers = 8
	start := make(chan struct{})
	outcomes := make(chan bool, callers)
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l, err := NewLinuxFileSignerOperationLedger(dir)
			if err != nil {
				errs <- err
				return
			}
			<-start
			ok, err := l.ReserveSignerOperation(context.Background(), op)
			if err != nil {
				errs <- err
			} else {
				outcomes <- ok
			}
		}()
	}
	close(start)
	wg.Wait()
	close(outcomes)
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent reservation failed: %v", err)
	}
	winners := 0
	for ok := range outcomes {
		if ok {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("concurrent Hosts in same filesystem reserved identical ID %d times", winners)
	}
}

func TestLinuxFileSignerOperationLedgerRejectsUnsafeDirectory(t *testing.T) {
	dir := t.TempDir()
	loose := filepath.Join(dir, "loose")
	if err := os.Mkdir(loose, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := NewLinuxFileSignerOperationLedger(loose); err == nil {
		t.Fatal("world-readable Host directory was accepted")
	}
	private := filepath.Join(dir, "private")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(private, link); err != nil {
		t.Fatal(err)
	}
	if _, err := NewLinuxFileSignerOperationLedger(link); err == nil {
		t.Fatal("symlink directory was accepted")
	}
	if _, err := NewLinuxFileSignerOperationLedger("relative"); err == nil {
		t.Fatal("relative storage path was accepted")
	}
}

func TestLinuxFileSignerOperationLedgerFailsClosed(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	ledger, err := NewLinuxFileSignerOperationLedger(dir)
	if err != nil {
		t.Fatal(err)
	}
	op := LinuxSignerOperationReservation{
		HostID: "host/reference", OperationID: "op-cancelled",
		Principal: "host-admin", SignerRef: "signed/tool", Action: "revoke",
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if ok, err := ledger.ReserveSignerOperation(ctx, op); ok || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled reservation was accepted: ok=%v err=%v", ok, err)
	}
	if ok, err := ledger.ReserveSignerOperation(context.Background(), op); !ok || err != nil {
		t.Fatalf("canceled attempt wrongly reserved ID: ok=%v err=%v", ok, err)
	}
	if ok, err := ledger.ReserveSignerOperation(context.Background(), LinuxSignerOperationReservation{
		HostID: "host/reference", OperationID: "op-invalid",
		Principal: "host-admin", SignerRef: "signed/tool", Action: "unknown",
	}); ok || err == nil {
		t.Fatalf("unknown mutation action was accepted: ok=%v err=%v", ok, err)
	}
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if ok, err := ledger.ReserveSignerOperation(context.Background(), op); ok || err == nil {
		t.Fatalf("storage permission change was not refused: ok=%v err=%v", ok, err)
	}
}

func TestLinuxFileSignerOperationLedgerWiredThroughAuditedAdmin(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	ledger, err := NewLinuxFileSignerOperationLedger(dir)
	if err != nil {
		t.Fatal(err)
	}
	admin, trust := newAuditedSignerFixture(&recordingLinuxSignerSink{}, true)
	admin.Operations = ledger
	req := signerOperationRequest()
	if err := admin.Allow(context.Background(), "host-admin", "op-durable", req); err != nil {
		t.Fatal(err)
	}
	oldEpoch, _ := admin.snapshot(req.SignerRef)
	newLedger, err := NewLinuxFileSignerOperationLedger(dir)
	if err != nil {
		t.Fatal(err)
	}
	admin.Operations = newLedger
	if err := admin.Allow(context.Background(), "host-admin", "op-durable", req); !errors.Is(err, ErrLinuxSignerOperationReplay) {
		t.Fatalf("reconstructed Host ledger allowed replay: %v", err)
	}
	if !trust.current(req.SignerRef, oldEpoch) {
		t.Fatal("replayed operation changed previously authorized signer")
	}
}
