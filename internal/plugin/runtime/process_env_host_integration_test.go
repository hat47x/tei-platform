package runtime_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"tei/internal/core"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
)

// checkProcessEnvProbe compares only observations, never copies secret values
// into test failure messages or plugin protocol evidence.
func checkProcessEnvProbe(t *testing.T, resp pruntime.Response, wantExplicit string) {
	t.Helper()
	var observed struct {
		HostSecretPresent bool   `json:"hostSecretPresent"`
		Explicit          string `json:"explicit"`
	}
	if err := json.Unmarshal(resp.Payload, &observed); err != nil {
		t.Fatal(err)
	}
	if observed.HostSecretPresent || observed.Explicit != wantExplicit {
		t.Fatalf("child env boundary violation: hostSecretPresent=%t explicitMatches=%t", observed.HostSecretPresent, observed.Explicit == wantExplicit)
	}
}

// This is a transport/invocation integration test, not a test of real OS
// network or filesystem enforcement: the host adapters are test doubles.
func TestHostSandboxProcessDoesNotInheritHostEnvironment(t *testing.T) {
	t.Setenv("TEI_PROCESS_HOST_SECRET_CANARY", "parent-secret-only")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := pruntime.ProcessCommand{Executable: executable, Args: []string{"--tei-process-env-probe"}}
	for _, tt := range []struct {
		name         string
		env          []string
		wantExplicit string
	}{
		{name: "missing environment"},
		{name: "allowlisted environment", env: []string{"TEI_PROCESS_EXPLICIT_ALLOWLIST=approved"}, wantExplicit: "approved"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			net, fs, secrets := &netEnforcer{}, &fsEnforcer{}, &secretInjector{}
			host := pruntime.HostSandboxLauncher{
				Network: net, Filesystem: fs, Secrets: secrets,
				Runner: pruntime.StdioProcessLauncher{Env: tt.env},
			}
			spec := pruntime.SandboxSpec{
				Command: command,
				Profile: pruntime.SandboxProfile{
					DenyNetworkByDefault: true, DenyFilesystemByDefault: true,
					SecretInjections: []pruntime.SecretInjection{{
						Scope: "credential", ResourceID: core.MustCanonicalID("prod.secret"),
						Handle: "vault://opaque", Delivery: resource.SecretDeliveryEnvironment,
						Target: "TEI_DATABASE_CREDENTIAL",
					}},
				},
			}
			resp, err := host.Invoke(context.Background(), spec, processRequest())
			if err != nil {
				t.Fatal(err)
			}
			checkProcessEnvProbe(t, resp, tt.wantExplicit)
			if !net.called || !net.cleaned || !fs.called || !fs.cleaned || !secrets.called || !secrets.cleaned {
				t.Fatalf("host enforcement lifecycle incomplete")
			}
		})
	}
}
