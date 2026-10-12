//go:build linux

package runtime

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"runtime"
	"testing"
	"time"
)

// waitSignerCondition keeps a failed concurrency invariant bounded: no test
// should hang indefinitely because a goroutine failed to reach a lock.
func waitSignerCondition(t *testing.T, description string, test func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !test() {
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for %s", description)
		}
		runtime.Gosched()
	}
}

func signerTestKey(t *testing.T) ed25519.PublicKey {
	t.Helper()
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return public
}

func TestLinuxSignerRevokeDrainsAdmittedInvocation(t *testing.T) {
	trust := NewLinuxHostSignerTrust()
	const ref = "reference/signer"
	key := signerTestKey(t)
	if err := trust.Allow(ref, key); err != nil {
		t.Fatal(err)
	}
	_, epoch, err := trust.resolve(ref)
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	invoked := make(chan error, 1)
	go func() {
		_, err := trust.invokeCurrent(ref, epoch, func() (Response, error) {
			close(entered)
			<-release
			return Response{}, nil
		})
		invoked <- err
	}()
	<-entered
	revoked := make(chan bool, 1)
	go func() { revoked <- trust.Revoke(ref) }()
	waitSignerCondition(t, "revoke entry removal", func() bool { return !trust.current(ref, epoch) })
	// The old generation is no longer admitted, even before its already
	// admitted call has finished.
	if _, err := trust.invokeCurrent(ref, epoch, func() (Response, error) {
		t.Error("old signer was admitted after revocation")
		return Response{}, nil
	}); err == nil {
		t.Fatal("revoked signer admitted a new invocation")
	}
	select {
	case <-revoked:
		t.Fatal("Revoke returned while its admitted invocation was still running")
	default:
	}
	// A slow or unresponsive provider must not block another signer's update.
	unrelated := make(chan error, 1)
	go func() { unrelated <- trust.Allow("reference/unrelated", key) }()
	select {
	case err := <-unrelated:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("a running process blocked an unrelated signer update")
	}
	close(release)
	if err := <-invoked; err != nil {
		t.Fatalf("admitted invocation did not complete: %v", err)
	}
	select {
	case yes := <-revoked:
		if !yes {
			t.Fatal("present signer was not revoked")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Revoke never returned after the final invocation drained")
	}
	if trust.Revoke(ref) {
		t.Fatal("removed signer was revoked a second time")
	}
}

func TestLinuxSignerRotationDrainsOnlyOldGeneration(t *testing.T) {
	trust := NewLinuxHostSignerTrust()
	const ref = "reference/signer"
	if err := trust.Allow(ref, signerTestKey(t)); err != nil {
		t.Fatal(err)
	}
	_, oldEpoch, err := trust.resolve(ref)
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	oldDone := make(chan error, 1)
	go func() {
		_, err := trust.invokeCurrent(ref, oldEpoch, func() (Response, error) {
			close(entered)
			<-release
			return Response{}, nil
		})
		oldDone <- err
	}()
	<-entered
	rotated := make(chan error, 1)
	go func() { rotated <- trust.Allow(ref, signerTestKey(t)) }()
	var newEpoch uint64
	waitSignerCondition(t, "new signer generation", func() bool {
		_, epoch, err := trust.resolve(ref)
		if err != nil || epoch == oldEpoch {
			return false
		}
		newEpoch = epoch
		return true
	})
	if _, err := trust.invokeCurrent(ref, oldEpoch, func() (Response, error) {
		return Response{}, errors.New("stale signer was admitted")
	}); err == nil {
		t.Fatal("rotation accepted a stale signer")
	}
	// New approvals are not blocked by the old process draining. Only
	// completion of the Allow operation waits for the old invocation.
	if _, err := trust.invokeCurrent(ref, newEpoch, func() (Response, error) {
		return Response{}, nil
	}); err != nil {
		t.Fatalf("new signer was not available: %v", err)
	}
	select {
	case <-rotated:
		t.Fatal("Allow completed without draining its previous signer")
	default:
	}
	close(release)
	if err := <-oldDone; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-rotated:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Allow did not finish after the old generation drained")
	}
}
