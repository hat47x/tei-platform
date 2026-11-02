package hoststore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

func ts(t time.Time) string { return t.Format(time.RFC3339) }

func ptr(s string) *string { return &s }

type fakeClock struct {
	now   time.Time
	calls int
}

func (c *fakeClock) read() time.Time { c.calls++; return c.now }

func verifierAt(t *testing.T, k issuerKey, extra map[string]any, clock func() time.Time) DecisionVerifier {
	t.Helper()
	policy, err := ParseTrustPolicy(trustDoc(t, k, extra))
	if err != nil {
		t.Fatal(err)
	}
	return policy.WithClock(clock).Verifier()
}

func TestKeyWindowBoundaries(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	d := signed(t, k, "d1", "application order_entry {}\n")
	both := map[string]any{"notBefore": ts(t0), "notAfter": ts(t0.Add(time.Hour))}
	cases := []struct {
		name  string
		extra map[string]any
		now   time.Time
		want  string
	}{
		{"now = notBefore accepts", both, t0, ""},
		{"notBefore - 1ns refuses", both, t0.Add(-time.Nanosecond), "HOST_STORE_KEY_NOT_YET_VALID"},
		{"notAfter - 1ns accepts", both, t0.Add(time.Hour - time.Nanosecond), ""},
		{"now = notAfter refuses", both, t0.Add(time.Hour), "HOST_STORE_KEY_EXPIRED"},
		{"only notBefore, after", map[string]any{"notBefore": ts(t0)}, t0.Add(1000 * time.Hour), ""},
		{"only notBefore, before", map[string]any{"notBefore": ts(t0)}, t0.Add(-time.Second), "HOST_STORE_KEY_NOT_YET_VALID"},
		{"only notAfter, long before", map[string]any{"notAfter": ts(t0)}, t0.Add(-1000 * time.Hour), ""},
		{"only notAfter, at", map[string]any{"notAfter": ts(t0)}, t0, "HOST_STORE_KEY_EXPIRED"},
		{"revoked beats a valid window", map[string]any{"notBefore": ts(t0), "notAfter": ts(t0.Add(time.Hour)), "revoked": true}, t0, "HOST_STORE_KEY_REVOKED"},
		{"fractional seconds", map[string]any{"notAfter": "2030-01-01T00:00:00.5Z"}, t0.Add(400 * time.Millisecond), ""},
	}
	for _, c := range cases {
		clock := &fakeClock{now: c.now}
		err := verifierAt(t, k, c.extra, clock.read)(context.Background(), d)
		if c.want == "" && err != nil || c.want != "" && (err == nil || err.Error() != c.want) {
			t.Errorf("%s: got %v want %q", c.name, err, c.want)
		}
	}
}

func TestRotationHasNoGapAndNoOverlap(t *testing.T) {
	k1, k2 := newIssuer(t, "issuer-a", "k1"), newIssuer(t, "issuer-a", "k2")
	policy := trustOf(t, []string{"order_entry"}, k1, k2)
	policy.Issuers[0].NotAfter = ptr(ts(t0))
	policy.Issuers[1].NotBefore = ptr(ts(t0))
	d1, d2 := signed(t, k1, "d1", "application order_entry {}\n"), signed(t, k2, "d2", "application order_entry {}\n")
	for _, c := range []struct {
		now      time.Time
		ok1, ok2 bool
	}{
		{t0.Add(-time.Nanosecond), true, false},
		{t0, false, true},
	} {
		now := c.now
		v := policy.WithClock(func() time.Time { return now }).Verifier()
		if got := v(context.Background(), d1) == nil; got != c.ok1 {
			t.Errorf("k1 at %v: accepted=%v", c.now, got)
		}
		if got := v(context.Background(), d2) == nil; got != c.ok2 {
			t.Errorf("k2 at %v: accepted=%v", c.now, got)
		}
	}
}

func TestClockIsReadPerCallNotAtVerifierCreation(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	d := signed(t, k, "d1", "application order_entry {}\n")
	clock := &fakeClock{now: t0.Add(-time.Hour)}
	v := verifierAt(t, k, map[string]any{"notAfter": ts(t0)}, clock.read)
	if clock.calls != 0 {
		t.Fatalf("clock read at Verifier(): %d", clock.calls)
	}
	if err := v(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	clock.now = t0.Add(time.Second)
	wantErr(t, v(context.Background(), d), "HOST_STORE_KEY_EXPIRED")
}

func TestNilClockDefaultsToTimeNow(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	d := signed(t, k, "d1", "application order_entry {}\n")
	past, err := ParseTrustPolicy(trustDoc(t, k, map[string]any{"notAfter": rfc(-time.Hour)}))
	if err != nil {
		t.Fatal(err)
	}
	wantErr(t, past.WithClock(nil).Verifier()(context.Background(), d), "HOST_STORE_KEY_EXPIRED")
	future, err := ParseTrustPolicy(trustDoc(t, k, map[string]any{"notBefore": rfc(time.Hour)}))
	if err != nil {
		t.Fatal(err)
	}
	wantErr(t, future.WithClock(nil).Verifier()(context.Background(), d), "HOST_STORE_KEY_NOT_YET_VALID")
}

func TestUnusableClockFailsClosedOnlyForRestrictedKeys(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	d := signed(t, k, "d1", "application order_entry {}\n")
	bad := map[string]func() time.Time{
		"zero":       func() time.Time { return time.Time{} },
		"year 10000": func() time.Time { return time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC) },
		"year 0":     func() time.Time { return time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC) },
		"panic":      func() time.Time { panic("clock broke") },
	}
	for name, clock := range bad {
		wantErr(t, verifierAt(t, k, map[string]any{"notAfter": ts(t0)}, clock)(context.Background(), d), "HOST_STORE_CLOCK_INVALID")
		if err := verifierAt(t, k, nil, clock)(context.Background(), d); err != nil {
			t.Errorf("%s: unrestricted key must not consult the clock: %v", name, err)
		}
	}
}

func TestVerifyTimeRecheckCatchesPoliciesThatSkippedTheParser(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	d := signed(t, k, "d1", "application order_entry {}\n")
	for name, mutate := range map[string]func(*TrustedIssuer){
		"malformed":    func(i *TrustedIssuer) { i.NotAfter = ptr("tomorrow") },
		"empty window": func(i *TrustedIssuer) { i.NotBefore, i.NotAfter = ptr(ts(t0)), ptr(ts(t0)) },
		"offset":       func(i *TrustedIssuer) { i.NotAfter = ptr("2030-01-01T09:00:00+09:00") },
	} {
		policy := trustOf(t, []string{"order_entry"}, k)
		mutate(&policy.Issuers[0]) // mutated after parsing, as a struct-literal policy would be
		clock := &fakeClock{now: t0}
		wantErr(t, policy.WithClock(clock.read).Verifier()(context.Background(), d), "HOST_STORE_KEY_WINDOW_INVALID")
		if clock.calls != 0 {
			t.Errorf("%s: clock read before the window was validated", name)
		}
	}
}

func TestSignatureIsCheckedBeforeKeyState(t *testing.T) {
	k, other := newIssuer(t, "issuer-a", "k1"), newIssuer(t, "issuer-a", "k1")
	forged := signed(t, other, "d1", "application order_entry {}\n")
	for _, extra := range []map[string]any{{"notAfter": ts(t0)}, {"revoked": true}, {"notBefore": ts(t0.Add(time.Hour))}} {
		clock := &fakeClock{now: t0.Add(24 * time.Hour)}
		wantErr(t, verifierAt(t, k, extra, clock.read)(context.Background(), forged), "HOST_STORE_SIGNATURE_INVALID")
	}
}

func TestDuplicateIssuerKeyStillRejectedWithKeyState(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	policy, err := ParseTrustPolicy(trustDoc(t, k, map[string]any{"notAfter": ts(t0)}))
	if err != nil {
		t.Fatal(err)
	}
	policy.Issuers = append(policy.Issuers, policy.Issuers[0])
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseTrustPolicy(raw); err == nil {
		t.Fatal("duplicate issuer+keyId accepted")
	}
}

func TestStoreEnforcesExpiryAtReadTimeWithoutTouchingFiles(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	clock := &fakeClock{now: t0.Add(-time.Hour)}
	policy, err := ParseTrustPolicy(trustDoc(t, k, map[string]any{"notBefore": ts(t0.Add(-2 * time.Hour)), "notAfter": ts(t0)}))
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "store")
	store, err := Open(dir, policy.WithClock(clock.read).Verifier())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Ingest(context.Background(), signed(t, k, "d1", "application order_entry {}\n")); err != nil {
		t.Fatalf("ingest in window: %v", err)
	}
	if err := verify(store, members(t, "application order_entry {}\n")); err != nil {
		t.Fatalf("read in window: %v", err)
	}
	before := snapshot(t, dir)

	clock.now = t0
	_, err = store.ReadCurrentAcceptedSource(context.Background(), "order_entry")
	if !errors.Is(err, ErrNoCurrentAcceptance) {
		t.Fatalf("after notAfter: %v", err)
	}
	if strings.Contains(err.Error(), "HOST_STORE_KEY") {
		t.Fatalf("store error leaks the specific code: %s", err)
	}
	if verify(store, members(t, "application order_entry {}\n")) == nil {
		t.Fatal("expired key still verifies")
	}
	if after := snapshot(t, dir); !reflect.DeepEqual(before, after) {
		t.Fatal("enforcement modified Store files")
	}

	clock.now = t0.Add(-time.Hour)
	if err := verify(store, members(t, "application order_entry {}\n")); err != nil {
		t.Fatalf("clock back in window should read again: %v", err)
	}
	if err := store.Revoke("order_entry", "d1"); err != nil {
		t.Fatalf("Revoke must be unaffected: %v", err)
	}
}

func TestStoreIngestOutsideTheWindowIsRefused(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	clock := &fakeClock{now: t0.Add(time.Hour)}
	store := openStore(t, verifierAt(t, k, map[string]any{"notAfter": ts(t0)}, clock.read))
	err := store.Ingest(context.Background(), signed(t, k, "d1", "application order_entry {}\n"))
	if !errors.Is(err, ErrIngestRefused) || strings.Contains(err.Error(), "HOST_STORE_KEY") {
		t.Fatalf("got %v", err)
	}
	if _, err := store.ReadCurrentAcceptedSource(context.Background(), "order_entry"); !errors.Is(err, ErrNoCurrentAcceptance) {
		t.Fatalf("got %v", err)
	}
}

func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := os.ReadFile(path)
		out[path] = string(raw)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
