package runtime_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

const processHelperModeEnv = "TEI_PROCESS_HELPER_MODE"

func TestMain(m *testing.M) {
	// The dedicated probe uses argv so it works even with an empty child env.
	if len(os.Args) == 2 && os.Args[1] == "--tei-process-env-probe" {
		os.Exit(runProcessHelper("env-probe"))
	}
	if mode := os.Getenv(processHelperModeEnv); mode != "" {
		os.Exit(runProcessHelper(mode))
	}
	os.Exit(m.Run())
}

func runProcessHelper(mode string) int {
	if mode == "wait" {
		time.Sleep(30 * time.Second)
		return 0
	}
	if mode == "malformed" {
		_, _ = fmt.Fprint(os.Stdout, "{")
		return 0
	}
	if mode == "large" {
		_, _ = fmt.Fprint(os.Stdout, strings.Repeat("x", 4096))
		return 0
	}
	if mode == "exit" {
		_, _ = fmt.Fprintln(os.Stderr, "TOP-SECRET-SHOULD-NOT-LEAK")
		return 7
	}

	var req pruntime.Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		return 3
	}
	resp := pruntime.Response{
		ProtocolVersion:       req.ProtocolVersion,
		RequestID:             req.RequestID,
		Plugin:                req.Plugin,
		CapabilityID:          req.CapabilityID,
		DescriptorFingerprint: req.DescriptorFingerprint,
		Status:                pruntime.ResponseOK,
		Payload:               req.Payload,
	}
	if mode == "forged" {
		resp.Plugin.ID = core.MustCanonicalID("forged.plugin")
	}
	// An external process may put sensitive data in the error carrier, even
	// when it is reporting success or a malformed/mismatched response.
	switch mode {
	case "secret-error":
		resp.Status = pruntime.ResponseError
	case "secret-ok":
		// Deliberately attach an error field to an otherwise successful reply.
	case "secret-unknown-status":
		resp.Status = pruntime.ResponseStatus("unrecognized")
	case "secret-forged-identity":
		resp.Plugin.ID = core.MustCanonicalID("forged.plugin")
	}
	if strings.HasPrefix(mode, "secret-") {
		resp.Error = "TEI_PROCESS_SECRET_CANARY_DO_NOT_LOG"
		if mode != "secret-ok" {
			resp.Payload = json.RawMessage(`{"secret":"TEI_PROCESS_SECRET_CANARY_DO_NOT_LOG"}`)
		}
	}
	if mode == "env-probe" {
		payload, err := json.Marshal(struct {
			HostSecretPresent bool   `json:"hostSecretPresent"`
			Explicit          string `json:"explicit"`
		}{
			HostSecretPresent: os.Getenv("TEI_PROCESS_HOST_SECRET_CANARY") != "",
			Explicit:          os.Getenv("TEI_PROCESS_EXPLICIT_ALLOWLIST"),
		})
		if err != nil {
			return 4
		}
		resp.Payload = payload
	}
	if mode == "stderr-success" {
		_, _ = fmt.Fprintln(os.Stderr, "diagnostic on stderr")
	}
	if strings.HasPrefix(mode, "wire-") {
		// Send deliberately ambiguous wire bytes from a REAL child process;
		// encoding a typed Response normally makes these attacks impossible.
		resp.Payload = json.RawMessage(`{"quantity":1,"quantity":2}`)
		raw, err := json.Marshal(resp)
		if err != nil {
			return 4
		}
		wire := string(raw)
		var before, after string
		switch mode {
		case "wire-duplicate-status":
			before, after = `"status":"ok"`, `"status":"ok","status":"error"`
		case "wire-duplicate-request":
			before, after = `"requestId":"`, `"requestId":"forged","requestId":"`
		case "wire-escaped-duplicate-status":
			before, after = `"status":"ok"`, `"status":"ok","\u0073tatus":"error"`
		case "wire-casefold-status":
			before, after = `"status":"ok"`, `"status":"ok","Status":"error"`
		case "wire-unknown-sensitive":
			before, after = `"status":"ok"`, `"status":"ok","TOP-SECRET-DO-NOT-LOG":true`
		case "wire-duplicate-plugin-id":
			before, after = `"plugin":{"id":"`, `"plugin":{"id":"forged","id":"`
		case "wire-casefold-plugin-id":
			before, after = `"plugin":{"id":"`, `"plugin":{"ID":"forged","id":"`
		case "wire-unknown-plugin-field":
			before, after = `"plugin":{"id":"`, `"plugin":{"hostAdmin":true,"id":"`
		case "wire-opaque-payload-duplicates":
			// Carrier does not parse application-specific payload members.
			_, _ = fmt.Fprintln(os.Stdout, wire)
			return 0
		default:
			return 4
		}
		if !strings.Contains(wire, before) {
			return 4
		}
		_, _ = fmt.Fprintln(os.Stdout, strings.Replace(wire, before, after, 1))
		return 0
	}
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		return 4
	}
	return 0
}

func helperLauncher(t *testing.T, mode string) (pruntime.StdioProcessLauncher, pruntime.ProcessCommand) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	env := append([]string(nil), os.Environ()...)
	env = append(env, processHelperModeEnv+"="+mode)
	return pruntime.StdioProcessLauncher{Env: env}, pruntime.ProcessCommand{Executable: executable}
}

func processRequest() pruntime.Request {
	return pruntime.Request{
		ProtocolVersion:       pruntime.ProtocolVersion,
		RequestID:             "process-request-1",
		Plugin:                plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"},
		CapabilityKind:        plugin.CapabilityDeploymentExecutor,
		CapabilityID:          core.MustCanonicalID("process.reference"),
		DescriptorFingerprint: "descriptor-fingerprint",
		PluginLockFingerprint: "lock-fingerprint",
		Operation:             "execute",
		Payload:               json.RawMessage(`{"value":"ok"}`),
	}
}

func TestStdioProcessLauncherRoundTripsRealChildProcess(t *testing.T) {
	launcher, command := helperLauncher(t, "echo")
	req := processRequest()
	resp, err := launcher.Invoke(context.Background(), command, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.ProtocolVersion != req.ProtocolVersion || resp.RequestID != req.RequestID || resp.Plugin != req.Plugin || resp.CapabilityID != req.CapabilityID || resp.DescriptorFingerprint != req.DescriptorFingerprint {
		t.Fatalf("response identity=%#v", resp)
	}
	if string(resp.Payload) != string(req.Payload) {
		t.Fatalf("payload=%s want=%s", resp.Payload, req.Payload)
	}
}

// A missing Env must not mean "inherit everything" as it does in os/exec.
func TestStdioProcessLauncherEnvironmentIsHostExplicit(t *testing.T) {
	t.Setenv("TEI_PROCESS_HOST_SECRET_CANARY", "parent-only-secret")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name         string
		env          []string
		wantExplicit string
	}{
		{name: "nil env does not inherit"},
		{name: "empty env does not inherit", env: []string{}},
		{name: "explicit allowlist is passed", env: []string{"TEI_PROCESS_EXPLICIT_ALLOWLIST=allowed"}, wantExplicit: "allowed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			launcher := pruntime.StdioProcessLauncher{Env: tt.env}
			command := pruntime.ProcessCommand{
				Executable: executable, Args: []string{"--tei-process-env-probe"},
			}
			resp, err := launcher.Invoke(context.Background(), command, processRequest())
			if err != nil {
				t.Fatal(err)
			}
			var observed struct {
				HostSecretPresent bool   `json:"hostSecretPresent"`
				Explicit          string `json:"explicit"`
			}
			if err := json.Unmarshal(resp.Payload, &observed); err != nil {
				t.Fatal(err)
			}
			if observed.HostSecretPresent || observed.Explicit != tt.wantExplicit {
				t.Fatalf("unexpected child environment: %#v", observed)
			}
		})
	}
}

func TestStdioProcessLauncherKeepsSuccessStderrOutOfProtocol(t *testing.T) {
	launcher, command := helperLauncher(t, "stderr-success")
	resp, err := launcher.Invoke(context.Background(), command, processRequest())
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != pruntime.ResponseOK {
		t.Fatalf("response=%#v", resp)
	}
}

func TestStdioProcessLauncherRejectsMalformedResponse(t *testing.T) {
	launcher, command := helperLauncher(t, "malformed")
	if _, err := launcher.Invoke(context.Background(), command, processRequest()); err == nil || !strings.Contains(err.Error(), "decode process plugin response") {
		t.Fatalf("error=%v", err)
	}
}

func TestStdioProcessLauncherRejectsAmbiguousResponseHeaders(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode string
		want string
	}{
		{"duplicate status", "wire-duplicate-status", "duplicate process response field"},
		{"duplicate request id", "wire-duplicate-request", "duplicate process response field"},
		{"unicode-escaped duplicate status", "wire-escaped-duplicate-status", "duplicate process response field"},
		{"case-folded status alias", "wire-casefold-status", "undeclared process response envelope field"},
		{"sensitive unknown field", "wire-unknown-sensitive", "undeclared process response envelope field"},
		{"duplicate nested plugin id", "wire-duplicate-plugin-id", "duplicate process response field"},
		{"case-folded nested plugin id", "wire-casefold-plugin-id", "undeclared process response plugin identity field"},
		{"unknown nested plugin field", "wire-unknown-plugin-field", "undeclared process response plugin identity field"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			launcher, command := helperLauncher(t, tc.mode)
			_, err := launcher.Invoke(context.Background(), command, processRequest())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("untrusted process response was accepted: %v", err)
			}
			if strings.Contains(err.Error(), "TOP-SECRET-DO-NOT-LOG") {
				t.Fatalf("untrusted process response field leaked into diagnostics: %v", err)
			}
		})
	}
}

func TestStdioProcessLauncherDoesNotValidateOpaqueBusinessPayload(t *testing.T) {
	launcher, command := helperLauncher(t, "wire-opaque-payload-duplicates")
	response, err := launcher.Invoke(context.Background(), command, processRequest())
	if err != nil {
		t.Fatal(err)
	}
	// The selected Service Contract validates this raw payload later. The
	// carrier must not imply domain conformance or silently rewrite bytes.
	if got := string(response.Payload); got != `{"quantity":1,"quantity":2}` {
		t.Fatalf("carrier modified opaque JSON payload: %s", got)
	}
}

func TestStdioProcessLauncherBoundsUntrustedStdout(t *testing.T) {
	launcher, command := helperLauncher(t, "large")
	launcher.MaxResponseBytes = 128
	if _, err := launcher.Invoke(context.Background(), command, processRequest()); err == nil || !strings.Contains(err.Error(), "response exceeds 128 bytes") {
		t.Fatalf("oversized response was not rejected: %v", err)
	}
}

func TestStdioProcessLauncherDoesNotCopyFailureStderr(t *testing.T) {
	launcher, command := helperLauncher(t, "exit")
	_, err := launcher.Invoke(context.Background(), command, processRequest())
	if err == nil {
		t.Fatal("expected process failure")
	}
	if strings.Contains(err.Error(), "TOP-SECRET-SHOULD-NOT-LEAK") {
		t.Fatalf("stderr leaked into host error: %v", err)
	}
	if !strings.Contains(err.Error(), "stderrBytes=") || !strings.Contains(err.Error(), "stderrSHA256=") {
		t.Fatalf("failure lacks bounded stderr diagnostics: %v", err)
	}
}

func TestStdioProcessLauncherCancellationTerminatesChild(t *testing.T) {
	launcher, command := helperLauncher(t, "wait")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := launcher.Invoke(ctx, command, processRequest())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v", err)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("canceled child did not terminate promptly: %s", elapsed)
	}
}

func TestRegistryRejectsForgedIdentityFromRealProcess(t *testing.T) {
	launcher, command := helperLauncher(t, "forged")
	req := processRequest()
	plan := pruntime.EnvironmentPlan{
		Environment:              core.MustCanonicalID("production"),
		PluginLockFingerprint:    req.PluginLockFingerprint,
		RuntimePolicyFingerprint: "runtime-policy",
		Plugins: []pruntime.PluginRuntime{{
			Plugin:                req.Plugin,
			DescriptorFingerprint: req.DescriptorFingerprint,
			Boundary:              pruntime.BoundaryProcess,
			Trust:                 pruntime.TrustExternalProcess,
			Command:               command.Executable,
			Args:                  append([]string(nil), command.Args...),
			SandboxPolicy:         plugin.SandboxNone,
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: req.Plugin,
			Capability: plugin.Capability{
				Kind: req.CapabilityKind,
				ID:   req.CapabilityID,
			},
			Boundary: pruntime.BoundaryProcess,
		}},
	}
	registry := pruntime.Registry{Processes: launcher}
	if _, err := registry.Invoke(context.Background(), plan, req.CapabilityKind, req.CapabilityID, req.Operation, req.RequestID, map[string]string{"value": "ok"}); err == nil || !strings.Contains(err.Error(), "response identity mismatch") {
		t.Fatalf("forged process identity was accepted: %v", err)
	}
}
