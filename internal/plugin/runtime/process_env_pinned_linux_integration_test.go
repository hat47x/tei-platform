//go:build linux

package runtime_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	"tei/internal/core"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
)

// The Linux test binary is copied into the pinned, unlinked image and executed
// via /proc/self/fd/3. Its probe mode uses argv, not an inherited environment.
func TestPinnedLinuxProcessEnvironmentIsHostExplicit(t *testing.T) {
	t.Setenv("TEI_PROCESS_HOST_SECRET_CANARY", "parent-only-pinned-secret")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(binary)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	req := processRequest()
	command := pruntime.ProcessCommand{Executable: executable, Args: []string{"--tei-process-env-probe"}}

	for _, tt := range []struct {
		name         string
		env          []string
		wantExplicit string
	}{
		{name: "no inherited variables"},
		{name: "explicit host variables", env: []string{"TEI_PROCESS_EXPLICIT_ALLOWLIST=approved"}, wantExplicit: "approved"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pinned, err := pruntime.PinLinuxProcess(
				context.Background(), req.Plugin, executable, digest,
				pruntime.StdioProcessLauncher{Env: tt.env},
			)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := pinned.Close(); err != nil {
					t.Error(err)
				}
			})

			// Pinned content alone must not restore os/exec's ambient
			// environment inheritance.
			resp, err := pinned.Invoke(context.Background(), command, req)
			if err != nil {
				t.Fatal(err)
			}
			checkProcessEnvProbe(t, resp, tt.wantExplicit)

			// Also exercise HostSandboxLauncher -> PinnedLinuxProcess ->
			// StdioProcessLauncher as one actual child-process path.
			// The test-double adapters observe lifecycle, not OS isolation.
			net, fs, secrets := &netEnforcer{}, &fsEnforcer{}, &secretInjector{}
			host := pruntime.HostSandboxLauncher{
				Network: net, Filesystem: fs, Secrets: secrets, Runner: pinned,
			}
			spec := pruntime.SandboxSpec{
				Command: command,
				Profile: pruntime.SandboxProfile{
					DenyNetworkByDefault: true,
					DenyFilesystemByDefault: true,
					SecretInjections: []pruntime.SecretInjection{{
						Scope: "credential", ResourceID: core.MustCanonicalID("prod.secret"),
						Handle: "vault://opaque", Delivery: resource.SecretDeliveryEnvironment,
						Target: "TEI_DATABASE_CREDENTIAL",
					}},
				},
			}
			resp, err = host.Invoke(context.Background(), spec, req)
			if err != nil {
				t.Fatal(err)
			}
			checkProcessEnvProbe(t, resp, tt.wantExplicit)
			if !net.called || !net.cleaned || !fs.called || !fs.cleaned || !secrets.called || !secrets.cleaned {
				t.Fatal("host sandbox adapters were not invoked and cleaned")
			}
		})
	}
}
