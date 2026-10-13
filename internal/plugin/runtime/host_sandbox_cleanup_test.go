package runtime_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
)

// sandboxCleanupProbe simulates host adapters that either forget to return a
// teardown hook or report a partial failure with a valid rollback callback.
// It does not claim to enforce any operating-system policy.
type sandboxCleanupProbe struct {
	missing string
	fail    string
	applied []string
	cleaned []string
}

func (p *sandboxCleanupProbe) apply(label string) (pruntime.Cleanup, error) {
	p.applied = append(p.applied, label)
	if p.missing == label {
		return nil, nil
	}
	cleanup := func(context.Context) error {
		p.cleaned = append(p.cleaned, label)
		return nil
	}
	if p.fail == label {
		return cleanup, errors.New("partial host adapter failure")
	}
	return cleanup, nil
}

func (p *sandboxCleanupProbe) ApplyNetworkPolicy(context.Context, pruntime.NetworkPolicy) (pruntime.Cleanup, error) {
	return p.apply("network")
}
func (p *sandboxCleanupProbe) ApplyFilesystemPolicy(context.Context, pruntime.FilesystemPolicy) (pruntime.Cleanup, error) {
	return p.apply("filesystem")
}
func (p *sandboxCleanupProbe) InjectSecrets(context.Context, []pruntime.SecretInjection) (pruntime.Cleanup, error) {
	return p.apply("secret")
}

func TestHostSandboxRejectsMissingCleanupAndRollsBackEarlierAdapters(t *testing.T) {
	for _, tc := range []struct {
		name      string
		stage     string
		wantApply []string
		wantClean []string
	}{
		{name: "network", stage: "network", wantApply: []string{"network"}},
		{name: "filesystem", stage: "filesystem", wantApply: []string{"network", "filesystem"}, wantClean: []string{"network"}},
		{name: "secret", stage: "secret", wantApply: []string{"network", "filesystem", "secret"}, wantClean: []string{"filesystem", "network"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			probe := &sandboxCleanupProbe{missing: tc.stage}
			runner := &sandboxRunner{}
			host := pruntime.HostSandboxLauncher{
				Network: probe, Filesystem: probe, Secrets: probe, Runner: runner,
			}
			resp, err := host.Invoke(context.Background(), cleanupTestSpec(), pruntime.Request{})
			if err == nil || !strings.Contains(err.Error(), "provided no cleanup") {
				t.Fatalf("missing cleanup accepted: %v", err)
			}
			if strings.Contains(err.Error(), "secret-handle-canary") {
				t.Fatal("secret handle escaped into cleanup error")
			}
			if runner.called || resp.Status != "" {
				t.Fatal("runner was invoked despite missing cleanup")
			}
			if !reflect.DeepEqual(probe.applied, tc.wantApply) {
				t.Fatalf("applied=%v, want %v", probe.applied, tc.wantApply)
			}
			if !reflect.DeepEqual(probe.cleaned, tc.wantClean) {
				t.Fatalf("cleanup=%v, want %v", probe.cleaned, tc.wantClean)
			}
		})
	}
}

func TestHostSandboxCleansPartiallyAppliedAdapterOnError(t *testing.T) {
	for _, tc := range []struct {
		stage     string
		wantApply []string
		wantClean []string
	}{
		{stage: "network", wantApply: []string{"network"}, wantClean: []string{"network"}},
		{stage: "filesystem", wantApply: []string{"network", "filesystem"}, wantClean: []string{"filesystem", "network"}},
		{stage: "secret", wantApply: []string{"network", "filesystem", "secret"}, wantClean: []string{"secret", "filesystem", "network"}},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			probe := &sandboxCleanupProbe{fail: tc.stage}
			runner := &sandboxRunner{}
			host := pruntime.HostSandboxLauncher{
				Network: probe, Filesystem: probe, Secrets: probe, Runner: runner,
			}
			resp, err := host.Invoke(context.Background(), cleanupTestSpec(), pruntime.Request{})
			if err == nil || !strings.Contains(err.Error(), "partial host adapter failure") {
				t.Fatalf("partial adapter error was dropped: %v", err)
			}
			if strings.Contains(err.Error(), "secret-handle-canary") {
				t.Fatal("secret handle escaped into cleanup error")
			}
			if runner.called || resp.Status != "" {
				t.Fatal("runner was invoked following partial adapter failure")
			}
			if !reflect.DeepEqual(probe.applied, tc.wantApply) {
				t.Fatalf("applied=%v, want %v", probe.applied, tc.wantApply)
			}
			if !reflect.DeepEqual(probe.cleaned, tc.wantClean) {
				t.Fatalf("cleanup=%v, want %v", probe.cleaned, tc.wantClean)
			}
		})
	}
}

func cleanupTestSpec() pruntime.SandboxSpec {
	return pruntime.SandboxSpec{
		Command: pruntime.ProcessCommand{Executable: "worker"},
		Profile: pruntime.SandboxProfile{
			DenyNetworkByDefault:    true,
			DenyFilesystemByDefault: true,
			SecretInjections: []pruntime.SecretInjection{{
				Scope:      "credential",
				ResourceID: core.MustCanonicalID("prod.secret"),
				Handle:     "secret-handle-canary",
				Delivery:   resource.SecretDeliveryEnvironment,
				Target:     "DB_CREDENTIAL",
			}},
		},
	}
}
