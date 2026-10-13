//go:build linux

package runtime

import (
	"context"
	"crypto/ed25519"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The callback represents the point at which the OS accepts a child start.
// A simultaneous Host revocation or rotation must never return while the
// previously authorized Start call is still within that critical section.
func TestLinuxSignerStartGateSerializesRevocationAndRotation(t *testing.T) {
	const ref = "host-signer/start-test"
	for _, tc := range []struct {
		name   string
		change func(*LinuxHostSignerTrust, string, ed25519.PublicKey) error
	}{
		{name: "revoke", change: func(trust *LinuxHostSignerTrust, ref string, _ ed25519.PublicKey) error {
			if !trust.Revoke(ref) {
				return errors.New("trusted signer did not revoke")
			}
			return nil
		}},
		{name: "rotate", change: func(trust *LinuxHostSignerTrust, ref string, key ed25519.PublicKey) error {
			other := append(ed25519.PublicKey(nil), key...)
			other[0] ^= 1
			return trust.Allow(ref, other)
		}},
		{name: "re-register-same-key", change: func(trust *LinuxHostSignerTrust, ref string, key ed25519.PublicKey) error {
			return trust.Allow(ref, key)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trust := NewLinuxHostSignerTrust()
			key := make(ed25519.PublicKey, ed25519.PublicKeySize)
			if err := trust.Allow(ref, key); err != nil {
				t.Fatal(err)
			}
			_, epoch, err := trust.resolve(ref)
			if err != nil {
				t.Fatal(err)
			}

			inStart := make(chan struct{})
			releaseStart := make(chan struct{})
			startResult := make(chan error, 1)
			go func() {
				startResult <- trust.startIfCurrent(ref, epoch, func() error {
					close(inStart)
					<-releaseStart
					return nil
				})
			}()
			select {
			case <-inStart:
			case <-time.After(3 * time.Second):
				t.Fatal("start callback was never reached")
			}

			changeResult := make(chan error, 1)
			go func() { changeResult <- tc.change(trust, ref, key) }()
			select {
			case err := <-changeResult:
				close(releaseStart)
				t.Fatalf("trust change completed while Start was protected: %v", err)
			case <-time.After(75 * time.Millisecond):
				// Trust writer must still be waiting for the Start callback.
			}
			close(releaseStart)
			select {
			case err := <-startResult:
				if err != nil {
					t.Fatalf("an authorized Start was rejected: %v", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("protected Start failed to return")
			}
			select {
			case err := <-changeResult:
				if err != nil {
					t.Fatalf("trust change failed after Start: %v", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("trust change could not complete after Start")
			}

			startedAgain := false
			if err := trust.startIfCurrent(ref, epoch, func() error {
				startedAgain = true
				return nil
			}); err == nil || startedAgain {
				t.Fatal("stale signed image restarted after Host trust changed")
			}
		})
	}
}

func TestLinuxSignerStartGateDoesNotBlockRevokeDuringChildLifetime(t *testing.T) {
	const ref = "host-signer/nonblocking-wait"
	trust := NewLinuxHostSignerTrust()
	if err := trust.Allow(ref, make(ed25519.PublicKey, ed25519.PublicKeySize)); err != nil {
		t.Fatal(err)
	}
	_, epoch, err := trust.resolve(ref)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	finishChild := make(chan struct{})
	childDone := make(chan error, 1)
	go func() {
		err := trust.startIfCurrent(ref, epoch, func() error {
			close(started)
			return nil
		})
		if err == nil {
			<-finishChild // A running child waits WITHOUT holding trust lock.
		}
		childDone <- err
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("start was not reached")
	}
	revoked := make(chan bool, 1)
	go func() { revoked <- trust.Revoke(ref) }()
	select {
	case ok := <-revoked:
		if !ok {
			t.Fatal("revoke did not remove the trusted signer")
		}
	case <-time.After(3 * time.Second):
		close(finishChild)
		t.Fatal("running child blocked Host trust revocation")
	}
	close(finishChild)
	if err := <-childDone; err != nil {
		t.Fatal(err)
	}
}

func TestStdioStartGateDeniesBeforeAnyProcessSideEffect(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "must-not-run")
	marker := filepath.Join(dir, "side-effect")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n: > \""+marker+"\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	gateCalls := 0
	_, err := (StdioProcessLauncher{}).invokeExecutableWithStartGate(
		context.Background(), ProcessCommand{Executable: script}, Request{}, script, nil,
		func(start func() error) error {
			gateCalls++
			return errors.New("Host signer trust was revoked or changed")
		},
	)
	if err == nil || !strings.Contains(err.Error(), "Host signer trust was revoked or changed") || gateCalls != 1 {
		t.Fatalf("untrusted process was not rejected by start gate: calls=%d err=%v", gateCalls, err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("process ran before Host start permission: %v", err)
	}
}
