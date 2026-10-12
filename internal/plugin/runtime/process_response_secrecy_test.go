package runtime_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

const processResponseSecretCanary = "TEI_PROCESS_SECRET_CANARY_DO_NOT_LOG"

// A child plugin's failure text is not a trusted host diagnostic. Exercise the
// actual JSON/stdio process transport, not a Response created only in memory.
func TestRegistryDoesNotPublishUntrustedProcessErrorContents(t *testing.T) {
	for _, tc := range []struct {
		name            string
		mode            string
		wantError       string
		wantSuccessData bool
	}{
		{name: "reported failure", mode: "secret-error", wantError: "untrusted process error details suppressed"},
		{name: "unexpected response status", mode: "secret-unknown-status", wantError: "invalid status"},
		{name: "identity mismatch", mode: "secret-forged-identity", wantError: "identity mismatch"},
		{name: "success with extraneous error field", mode: "secret-ok", wantSuccessData: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			launcher, command := helperLauncher(t, tc.mode)
			req := processRequest()
			plan := pruntime.EnvironmentPlan{
				Environment:           core.MustCanonicalID("reference"),
				PluginLockFingerprint: req.PluginLockFingerprint,
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
						Kind: req.CapabilityKind, ID: req.CapabilityID,
					},
					Boundary: pruntime.BoundaryProcess,
				}},
			}
			inv, err := (pruntime.Registry{Processes: launcher}).InvokeWithProvenance(
				context.Background(), plan, req.CapabilityKind, req.CapabilityID, req.Operation,
				req.RequestID, map[string]string{"value": "ok"},
			)
			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("success reply rejected: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("unexpected error category: %v", err)
			}
			if err != nil && strings.Contains(err.Error(), processResponseSecretCanary) {
				t.Fatal("process secret leaked into host diagnostic")
			}
			if inv.Response.Error != "" || strings.Contains(string(inv.Response.Error), processResponseSecretCanary) {
				t.Fatal("untrusted process error carrier escaped into Invocation")
			}
			if tc.wantSuccessData {
				if got := string(inv.Response.Payload); got != `{"value":"ok"}` {
					t.Fatalf("valid business payload was changed: %q", got)
				}
			} else if len(inv.Response.Payload) != 0 {
				t.Fatal("rejected response payload escaped into Invocation")
			}
		})
	}
}

type processErrorBuiltin struct {
	identity    plugin.Identity
	fingerprint string
}

func (b processErrorBuiltin) Identity() plugin.Identity     { return b.identity }
func (b processErrorBuiltin) DescriptorFingerprint() string { return b.fingerprint }
func (b processErrorBuiltin) Invoke(context.Context, pruntime.Request) (pruntime.Response, error) {
	return pruntime.Response{Status: pruntime.ResponseError, Error: "trusted-business-error"}, nil
}

// The fix is scoped to external Process responses; do not reinterpret an
// in-process builtin's established business error semantics.
func TestRegistryKeepsBuiltinBusinessError(t *testing.T) {
	req := processRequest()
	p := pruntime.EnvironmentPlan{
		Environment:           core.MustCanonicalID("reference"),
		PluginLockFingerprint: req.PluginLockFingerprint,
		Plugins: []pruntime.PluginRuntime{{
			Plugin: req.Plugin, DescriptorFingerprint: req.DescriptorFingerprint,
			Boundary: pruntime.BoundaryInProcess,
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: req.Plugin,
			Capability: plugin.Capability{Kind: req.CapabilityKind, ID: req.CapabilityID},
			Boundary: pruntime.BoundaryInProcess,
		}},
	}
	reg := pruntime.Registry{Builtins: map[string]pruntime.Builtin{
		string(req.Plugin.ID) + "@" + req.Plugin.Version: processErrorBuiltin{
			identity: req.Plugin, fingerprint: req.DescriptorFingerprint,
		},
	}}
	inv, err := reg.InvokeWithProvenance(context.Background(), p, req.CapabilityKind, req.CapabilityID, req.Operation, req.RequestID, nil)
	if err == nil || !strings.Contains(err.Error(), "trusted-business-error") {
		t.Fatalf("builtin business error was changed: %v", err)
	}
	if inv.Response.Error != "trusted-business-error" {
		t.Fatalf("builtin error response was changed: %q", inv.Response.Error)
	}
}
