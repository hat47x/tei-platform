package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runLockClassification builds the real CLI once and runs one command against a
// supplied lock, returning the exit code and the envelope's failure code.
func runLockClassification(t *testing.T, binary, command, workspace, environment, lockPath string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(binary, command, "-workspace", workspace, "-environment", environment, "-plugin-lock", lockPath)
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		exit, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run %s: %v (%s)", command, err, stderr.String())
		}
		code = exit.ExitCode()
	}
	var envelope struct {
		OK    bool `json:"ok"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout.String()), &envelope); err != nil {
		t.Fatalf("%s did not return one JSON envelope: %v (stdout=%q stderr=%q)", command, err, stdout.String(), stderr.String())
	}
	failure := ""
	message := ""
	if envelope.Error != nil {
		failure = envelope.Error.Code
		message = envelope.Error.Message
	}
	return code, failure, message
}

// A supplied lock that does not match the declaration is something only the
// caller can repair, so every command that reads -plugin-lock must classify it
// the same way. Before #3765 the same file answered exit 2 /
// PLUGIN_RESOLVE_FAILED from plugin-resolve and exit 1 /
// PLUGIN_RUNTIME_PLAN_FAILED from plugin-runtime-plan, so a caller testing
// `exit == 2` for "the input is wrong" silently missed the second command.
func TestAnInconsistentLockIsOneInputFailureInBothCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	binary := filepath.Join(t.TempDir(), "tei")
	if os.PathSeparator == '\\' {
		binary += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	dir := t.TempDir()

	// A valid lock for the development environment, then one broken copy per
	// defect the issue observed. Only the lock bytes change; the declaration
	// stays as it is, so each failure is about the supplied file.
	build := exec.Command(binary, "plugin-lock", "-workspace", reference, "-environment", "development")
	var lockOut, lockErrText strings.Builder
	build.Stdout, build.Stderr = &lockOut, &lockErrText
	if err := build.Run(); err != nil {
		t.Fatalf("build the development lock: %v (%s)", err, lockErrText.String())
	}
	var lockEnvelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal([]byte(lockOut.String()), &lockEnvelope); err != nil {
		t.Fatalf("plugin-lock did not return one envelope: %v (%s)", err, lockOut.String())
	}
	var lock struct {
		FormatVersion int `json:"formatVersion"`
		Environments  []struct {
			Environment string `json:"environment"`
			Plugins     []struct {
				ID          string `json:"id"`
				Version     string `json:"version"`
				Fingerprint string `json:"fingerprint"`
			} `json:"plugins"`
		} `json:"environments"`
	}
	if err := json.Unmarshal(lockEnvelope.Result, &lock); err != nil {
		t.Fatalf("plugin-lock result is not a lock: %v", err)
	}
	if len(lock.Environments) == 0 || len(lock.Environments[0].Plugins) == 0 {
		t.Fatalf("the development lock pins no plugin, so no defect can be expressed: %+v", lock)
	}

	write := func(name string, value any) string {
		t.Helper()
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, name+".json")
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	// decode into a fresh copy so each defect starts from the valid lock.
	copyLock := func() (map[string]any, []map[string]any) {
		t.Helper()
		var generic map[string]any
		if err := json.Unmarshal(lockEnvelope.Result, &generic); err != nil {
			t.Fatal(err)
		}
		environments, _ := generic["environments"].([]any)
		first, _ := environments[0].(map[string]any)
		plugins, _ := first["plugins"].([]any)
		out := make([]map[string]any, 0, len(plugins))
		for _, p := range plugins {
			out = append(out, p.(map[string]any))
		}
		return generic, out
	}

	defects := []struct {
		name  string
		build func() any
	}{
		{name: "fingerprint-rewritten", build: func() any {
			g, plugins := copyLock()
			plugins[0]["fingerprint"] = "sha256:" + strings.Repeat("0", 64)
			return g
		}},
		{name: "version-absent", build: func() any {
			g, plugins := copyLock()
			plugins[0]["version"] = "99.99.99"
			return g
		}},
		{name: "plugin-removed", build: func() any {
			g, _ := copyLock()
			environments := g["environments"].([]any)
			first := environments[0].(map[string]any)
			first["plugins"] = []any{}
			return g
		}},
		{name: "environment-unknown", build: func() any {
			g, _ := copyLock()
			environments := g["environments"].([]any)
			environments[0].(map[string]any)["environment"] = "absent_environment"
			return g
		}},
	}

	for _, defect := range defects {
		t.Run(defect.name, func(t *testing.T) {
			path := write(defect.name, defect.build())
			type outcome struct {
				code    int
				failure string
			}
			results := map[string]outcome{}
			for _, command := range []string{"plugin-resolve", "plugin-runtime-plan"} {
				code, failure, message := runLockClassification(t, binary, command, reference, "development", path)
				t.Logf("%s: exit=%d code=%q message=%q", command, code, failure, message)
				results[command] = outcome{code: code, failure: failure}
			}
			resolve, plan := results["plugin-resolve"], results["plugin-runtime-plan"]
			if resolve != plan {
				t.Fatalf("the same lock was classified differently: resolve=%+v plan=%+v", resolve, plan)
			}
			if resolve.code != 2 || resolve.failure != "INVALID_PLUGIN_LOCK" {
				t.Fatalf("an inconsistent lock is an input failure: got exit=%d code=%q", resolve.code, resolve.failure)
			}
		})
	}
}
