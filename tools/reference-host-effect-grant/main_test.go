package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

func clock() time.Time { return t0 }

func tf(d time.Duration) string { return t0.Add(d).Format(time.RFC3339) }

type env struct {
	dir, key, trust, grant string
}

func setup(t *testing.T) env {
	t.Helper()
	dir := t.TempDir()
	e := env{dir: dir, key: filepath.Join(dir, "issuer.key"), trust: filepath.Join(dir, "trust.json"), grant: filepath.Join(dir, "grant.json")}
	var out bytes.Buffer
	if err := run([]string{"keygen", "-private-key", e.key, "-issuer", "dev", "-key-id", "k1", "-application", "order_entry"}, &out, clock); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(e.trust, out.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return e
}

func (e env) signArgs(extra ...string) []string {
	return append([]string{"sign", "-private-key", e.key, "-issuer", "dev", "-key-id", "k1", "-application", "order_entry",
		"-capability", "order_entry.submit_order", "-subject", "editor", "-resource", "order/42",
		"-epoch-source", "local-auth-store", "-subject-link-epoch", "1", "-membership-epoch", "2", "-policy-epoch", "3",
		"-not-before", tf(-time.Minute), "-not-after", tf(time.Hour), "-out", e.grant}, extra...)
}

func TestKeygenSignShowRoundTrip(t *testing.T) {
	e := setup(t)
	rawTrust, _ := os.ReadFile(e.trust)
	if !strings.Contains(string(rawTrust), "tei.host-effect-grant-provisional-trust/v0") {
		t.Fatalf("%s", rawTrust)
	}
	key, _ := os.ReadFile(e.key)
	if strings.Contains(string(rawTrust), strings.TrimSpace(string(key))) {
		t.Fatal("trust entry carries the private key")
	}
	if err := run(e.signArgs(), &bytes.Buffer{}, clock); err != nil {
		t.Fatal(err)
	}
	rawGrant, _ := os.ReadFile(e.grant)
	var doc map[string]any
	if err := json.Unmarshal(rawGrant, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["provisional"] != true || doc["schema"] != "tei.host-effect-grant-provisional/v0" {
		t.Fatalf("%v", doc)
	}
	if strings.Contains(string(rawGrant), strings.TrimSpace(string(key))) {
		t.Fatal("grant carries the private key")
	}
	var out bytes.Buffer
	if err := run([]string{"show", "-grant", e.grant, "-trust", e.trust}, &out, clock); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "PROVISIONAL") || !strings.Contains(out.String(), "not G2 evidence") || strings.Contains(out.String(), doc["signature"].(string)) {
		t.Fatalf("%s", out.String())
	}
	// Expired at show time.
	late := func() time.Time { return t0.Add(2 * time.Hour) }
	if err := run([]string{"show", "-grant", e.grant, "-trust", e.trust}, &bytes.Buffer{}, late); err == nil {
		t.Fatal("expired grant shown as valid")
	}
}

func TestSignRefusesBadInputs(t *testing.T) {
	e := setup(t)
	cases := map[string][]string{
		"over 24h":    e.signArgs("-not-after", tf(25*time.Hour)),
		"reversed":    e.signArgs("-not-before", tf(time.Hour), "-not-after", tf(0)),
		"zero epoch":  e.signArgs("-policy-epoch", "0"),
		"offset time": e.signArgs("-not-after", "2026-11-03T22:00:00+09:00"),
	}
	for name, args := range cases {
		_ = os.Remove(e.grant)
		if err := run(args, &bytes.Buffer{}, clock); err == nil {
			t.Fatalf("%s accepted", name)
		}
		if _, err := os.Stat(e.grant); err == nil {
			t.Fatalf("%s left a grant file", name)
		}
	}
	if err := run([]string{"sign"}, &bytes.Buffer{}, clock); err == nil {
		t.Fatal("missing flags accepted")
	}
	// A wrong private key yields a Grant the trust file rejects.
	other := filepath.Join(e.dir, "other.key")
	if err := run([]string{"keygen", "-private-key", other, "-issuer", "dev", "-key-id", "k1", "-application", "order_entry"}, &bytes.Buffer{}, clock); err != nil {
		t.Fatal(err)
	}
	args := e.signArgs()
	for i, a := range args {
		if a == e.key {
			args[i] = other
		}
	}
	if err := run(args, &bytes.Buffer{}, clock); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"show", "-grant", e.grant, "-trust", e.trust}, &bytes.Buffer{}, clock); err == nil {
		t.Fatal("grant signed by an untrusted key accepted")
	}
}

func TestKeyMustStayOutsideRepositoryAndIsNeverOverwritten(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	inRepo := filepath.Join(repo, "sub", "k")
	if err := os.MkdirAll(filepath.Dir(inRepo), 0o700); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"keygen", "-private-key", inRepo, "-issuer", "dev", "-key-id", "k1", "-application", "order_entry"}, &bytes.Buffer{}, clock)
	if err == nil || !strings.HasPrefix(err.Error(), "PRIVATE_KEY_IN_REPOSITORY") {
		t.Fatalf("%v", err)
	}
	if _, err := os.Stat(inRepo); err == nil {
		t.Fatal("key written inside the repository")
	}
	// An existing key inside a repository is not usable for signing either.
	seed := base64.StdEncoding.EncodeToString(make([]byte, 32))
	if err := os.WriteFile(inRepo, []byte(seed+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	e := setup(t)
	args := e.signArgs()
	for i, a := range args {
		if a == e.key {
			args[i] = inRepo
		}
	}
	if err := run(args, &bytes.Buffer{}, clock); err == nil || !strings.HasPrefix(err.Error(), "PRIVATE_KEY_IN_REPOSITORY") {
		t.Fatalf("%v", err)
	}
	// Never overwrite.
	if err := run([]string{"keygen", "-private-key", e.key, "-issuer", "dev", "-key-id", "k1", "-application", "order_entry"}, &bytes.Buffer{}, clock); err == nil {
		t.Fatal("existing key overwritten")
	}
	// Bad flags leave no key behind.
	bad := filepath.Join(e.dir, "bad.key")
	if err := run([]string{"keygen", "-private-key", bad, "-issuer", "dev", "-key-id", "k1", "-application", "order_entry", "-not-before", "2026-01-01T00:00:00+09:00"}, &bytes.Buffer{}, clock); err == nil {
		t.Fatal("offset time accepted")
	}
	if _, err := os.Stat(bad); err == nil {
		t.Fatal("bad flags left a key file")
	}
}

func TestErrorsNeverEchoKeyMaterial(t *testing.T) {
	e := setup(t)
	if err := os.WriteFile(e.key, []byte("SECRET-NOT-A-SEED\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := run(e.signArgs(), &bytes.Buffer{}, clock)
	if err == nil || strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("%v", err)
	}
	if err := run([]string{"bogus"}, &bytes.Buffer{}, clock); err == nil {
		t.Fatal("unknown command accepted")
	}
}

// linkDir makes link point at target: a directory junction on Windows (no
// privilege needed), a symbolic link elsewhere. It skips when neither works.
func linkDir(t *testing.T, link, target string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
			t.Skipf("mklink /J unavailable: %v %s", err, out)
		}
		return
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
}

func repoWithSub(t *testing.T) (repo, sub string) {
	t.Helper()
	repo = t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	sub = filepath.Join(repo, "sub")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	return repo, sub
}

func TestKeygenRefusesJunctionOrLinkIntoRepository(t *testing.T) {
	_, sub := repoWithSub(t)
	outside := t.TempDir()
	jn := filepath.Join(outside, "jn")
	linkDir(t, jn, sub)
	err := run([]string{"keygen", "-private-key", filepath.Join(jn, "k5"), "-issuer", "dev", "-key-id", "k1", "-application", "order_entry"}, &bytes.Buffer{}, clock)
	if err == nil || !strings.HasPrefix(err.Error(), "PRIVATE_KEY_IN_REPOSITORY") {
		t.Fatalf("junction into the repository accepted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sub, "k5")); err == nil {
		t.Fatal("a key file was left inside the repository")
	}
}

func TestSignRefusesKeyReachedThroughJunctionOrLink(t *testing.T) {
	_, sub := repoWithSub(t)
	seed := base64.StdEncoding.EncodeToString(make([]byte, 32))
	if err := os.WriteFile(filepath.Join(sub, "k"), []byte(seed+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	jn := filepath.Join(outside, "jn")
	linkDir(t, jn, sub)
	e := setup(t)
	args := e.signArgs()
	for i, a := range args {
		if a == e.key {
			args[i] = filepath.Join(jn, "k")
		}
	}
	if err := run(args, &bytes.Buffer{}, clock); err == nil || !strings.HasPrefix(err.Error(), "PRIVATE_KEY_IN_REPOSITORY") {
		t.Fatalf("key behind a junction into the repository was used: %v", err)
	}
	if _, err := os.Stat(e.grant); err == nil {
		t.Fatal("a Grant was signed with a repository key")
	}
}

func TestSignNeverOverwritesAnything(t *testing.T) {
	e := setup(t)
	keyBefore, _ := os.ReadFile(e.key)
	trustBefore, _ := os.ReadFile(e.trust)
	for name, out := range map[string]string{
		"private key":     e.key,
		"trust file":      e.trust,
		"key via ..":      filepath.Join(e.dir, "sub", "..", "issuer.key"),
		"key, other case": strings.ToUpper(e.key[:1]) + e.key[1:],
		"existing grant":  e.grant,
	} {
		if name == "existing grant" {
			if err := os.WriteFile(e.grant, []byte("earlier"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		args := e.signArgs()
		args[len(args)-1] = out
		if name == "key via .." {
			_ = os.MkdirAll(filepath.Join(e.dir, "sub"), 0o700)
		}
		if err := run(args, &bytes.Buffer{}, clock); err == nil {
			t.Fatalf("%s: -out accepted", name)
		}
	}
	if got, _ := os.ReadFile(e.key); !bytes.Equal(got, keyBefore) {
		t.Fatal("private key was modified")
	}
	if got, _ := os.ReadFile(e.trust); !bytes.Equal(got, trustBefore) {
		t.Fatal("trust file was modified")
	}
	if got, _ := os.ReadFile(e.grant); string(got) != "earlier" {
		t.Fatal("an existing Grant file was overwritten")
	}
}
