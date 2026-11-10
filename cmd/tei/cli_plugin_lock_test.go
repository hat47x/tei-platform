package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// writeLock writes a lock fixture and returns its path.
func writeLock(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "lock.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// runLockedCommand runs one plugin command against a lock fixture.
func runLockedCommand(t *testing.T, binary, workspace, command, environment, lock string) (int, bool, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(binary, command, "-workspace", workspace,
		"-environment", environment, "-plugin-lock", lock)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		exit, isExit := err.(*exec.ExitError)
		if !isExit {
			t.Fatalf("%s: %v (%s)", command, err, stderr.String())
		}
		code = exit.ExitCode()
	}
	var doc struct {
		OK    bool `json:"ok"`
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("%s did not return one JSON document: %v (%s)", command, err, stdout.String())
	}
	failure := ""
	if doc.Error != nil {
		failure = doc.Error.Code
	}
	return code, doc.OK, failure
}

// The same unusable -plugin-lock must be classified the same way whichever
// command reads it. Before this, plugin-resolve answered a diverged pin with
// exit 2 while plugin-runtime-plan answered the same bytes with exit 1, so a
// caller that treats exit 2 as "fix your input" silently accepted the second
// answer as a finding (#3765).
func TestBrokenPluginLockIsClassifiedTheSameByBothCommands(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")

	cases := []struct {
		name string
		lock string
	}{
		{"malformed json", `{not json`},
		{"wrong shape", `{"foo":1}`},
		{"unsupported format", `{"formatVersion":0,"environments":[]}`},
		{"missing environment", `{"formatVersion":2,"environments":[{"environment":"other","plugins":[]}]}`},
		{"diverged pin", `{"formatVersion":2,"environments":[{"environment":"development","plugins":[` +
			`{"id":"tei.database.postgres","version":"1.0.0","fingerprint":"` + repeatHex(64) + `"}]}]}`},
		{"unavailable version", `{"formatVersion":2,"environments":[{"environment":"development","plugins":[` +
			`{"id":"tei.database.postgres","version":"9.9.9","fingerprint":"` + repeatHex(64) + `"}]}]}`},
	}

	binary := filepath.Join(t.TempDir(), "tei")
	if os.PathSeparator == '\\' {
		binary += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lock := writeLock(t, tc.lock)
			resolveCode, resolveOK, resolveFailure :=
				runLockedCommand(t, binary, reference, "plugin-resolve", "development", lock)
			planCode, planOK, planFailure :=
				runLockedCommand(t, binary, reference, "plugin-runtime-plan", "development", lock)

			if resolveOK || planOK {
				t.Fatalf("a broken lock answered ok: resolve=%v plan=%v", resolveOK, planOK)
			}
			if resolveCode != planCode {
				t.Fatalf("exit codes differ: plugin-resolve=%d plugin-runtime-plan=%d",
					resolveCode, planCode)
			}
			if resolveFailure != planFailure {
				t.Fatalf("codes differ: plugin-resolve=%q plugin-runtime-plan=%q",
					resolveFailure, planFailure)
			}
			if resolveCode != 2 {
				t.Fatalf("exit=%d, want 2 (input error)", resolveCode)
			}
			if resolveFailure != "INVALID_PLUGIN_LOCK" {
				t.Fatalf("code=%q, want INVALID_PLUGIN_LOCK", resolveFailure)
			}
		})
	}
}

func repeatHex(n int) string {
	var b bytes.Buffer
	for i := 0; i < n; i++ {
		b.WriteByte('a')
	}
	return b.String()
}
