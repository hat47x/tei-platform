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
	if mode == "stderr-success" {
		_, _ = fmt.Fprintln(os.Stderr, "diagnostic on stderr")
	}
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		return 4
	}
	return 0
}

func helperLauncher(t *testing.T, mode string) (pruntime.StdioProcessLauncher, string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	env := append([]string(nil), os.Environ()...)
	env = append(env, processHelperModeEnv+"="+mode)
	return pruntime.StdioProcessLauncher{Env: env}, executable
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
			Command:               command,
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
