package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestPluginLockEnvelopeIsAcceptedAsPluginLockInput pins #3183: the file that
// `tei plugin-lock` writes must go straight into -plugin-lock. A pipeline that
// creates a lock and then plans a runtime should not need an external jq or
// python3 step between them.
func TestPluginLockEnvelopeIsAcceptedAsPluginLockInput(t *testing.T) {
	binary := buildCLIForLifecycle(t)
	reference := filepath.Join("..", "..", "reference", "workspace")
	dir := t.TempDir()

	lockBody := discoveryRun(t, binary, "plugin-lock", "-workspace", reference)
	var envelope struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(lockBody, &envelope); err != nil || !envelope.OK || len(envelope.Result) == 0 {
		t.Fatalf("plugin-lock is not one envelope with a result: %v (%s)", err, lockBody)
	}
	// The envelope file, exactly as the CLI wrote it.
	envelopePath := filepath.Join(dir, "lock-envelope.json")
	if err := os.WriteFile(envelopePath, lockBody, 0o644); err != nil {
		t.Fatal(err)
	}
	// The same lock without the envelope, which the CLI already accepted.
	innerPath := filepath.Join(dir, "lock-inner.json")
	if err := os.WriteFile(innerPath, envelope.Result, 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		args []string
	}{
		{"plugin-runtime-plan envelope", []string{"plugin-runtime-plan", "-workspace", reference,
			"-environment", "development", "-plugin-lock", envelopePath}},
		{"plugin-runtime-plan inner", []string{"plugin-runtime-plan", "-workspace", reference,
			"-environment", "development", "-plugin-lock", innerPath}},
		{"plugin-resolve envelope", []string{"plugin-resolve", "-workspace", reference,
			"-environment", "development", "-plugin-lock", envelopePath}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := discoveryRun(t, binary, tc.args...)
			var response struct {
				OK     bool            `json:"ok"`
				Result json.RawMessage `json:"result"`
				Error  *cliFailure     `json:"error"`
			}
			if err := json.Unmarshal(body, &response); err != nil {
				t.Fatalf("not one envelope: %v (%s)", err, body)
			}
			if !response.OK || response.Error != nil || len(response.Result) == 0 {
				t.Fatalf("lock input was not accepted: %s", body)
			}
		})
	}

	// The two inputs must produce the same fingerprints: unwrapping the
	// envelope must not change what was pinned.
	fingerprint := func(args ...string) string {
		t.Helper()
		body := discoveryRun(t, binary, args...)
		var response struct {
			Result struct {
				PluginLockFingerprint    string `json:"pluginLockFingerprint"`
				RuntimePolicyFingerprint string `json:"runtimePolicyFingerprint"`
			} `json:"result"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("not one envelope: %v (%s)", err, body)
		}
		if response.Result.PluginLockFingerprint == "" {
			t.Fatalf("no pluginLockFingerprint in %s", body)
		}
		return response.Result.PluginLockFingerprint + "/" + response.Result.RuntimePolicyFingerprint
	}
	fromEnvelope := fingerprint("plugin-runtime-plan", "-workspace", reference,
		"-environment", "development", "-plugin-lock", envelopePath)
	fromInner := fingerprint("plugin-runtime-plan", "-workspace", reference,
		"-environment", "development", "-plugin-lock", innerPath)
	if fromEnvelope != fromInner {
		t.Fatalf("envelope and inner lock give different fingerprints: %q vs %q", fromEnvelope, fromInner)
	}

	// A file that is neither a lock nor an envelope still fails, and the
	// message names the file rather than a JSON offset inside it.
	bogusPath := filepath.Join(dir, "bogus.json")
	if err := os.WriteFile(bogusPath, []byte(`{"notALock": true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	body := discoveryRun(t, binary, "plugin-runtime-plan", "-workspace", reference,
		"-environment", "development", "-plugin-lock", bogusPath)
	var response struct {
		OK    bool        `json:"ok"`
		Error *cliFailure `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("not one envelope: %v (%s)", err, body)
	}
	if response.OK || response.Error == nil || response.Error.Code != "PLUGIN_RUNTIME_PLAN_FAILED" {
		t.Fatalf("a non-lock file was accepted: %s", body)
	}
	if !bytes.Contains([]byte(response.Error.Message), []byte("unsupported plugin lock format")) {
		t.Fatalf("unexpected message for a non-lock file: %q", response.Error.Message)
	}
}
