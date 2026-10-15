package runtime_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
)

func TestHostProcessEnvironmentProfileIsExplicitAndDeterministic(t *testing.T) {
	empty, err := (pruntime.HostProcessEnvironmentProfile{}).Environment()
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty profile must yield non-nil, empty environment: count=%d error=%v", len(empty), err)
	}

	root := t.TempDir()
	search := filepath.Join(root, "bin")
	home := filepath.Join(root, "home")
	tmp := filepath.Join(root, "tmp")
	profile := pruntime.HostProcessEnvironmentProfile{
		SearchDirectories:  []string{search},
		HomeDirectory:      home,
		TemporaryDirectory: tmp,
		Additional:         map[string]string{
			"TEI_PROFILE_Z": "last=with-equals",
			"TEI_PROFILE_A": "first",
		},
	}
	first, err := profile.Environment()
	if err != nil {
		t.Fatal(err)
	}
	second, err := profile.Environment()
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("profile environment is nondeterministic: %v", err)
	}
	want := []string{"PATH=" + search}
	if runtime.GOOS == "windows" {
		want = append(want, "USERPROFILE="+home, "TEMP="+tmp, "TMP="+tmp)
	} else {
		want = append(want, "HOME="+home, "TMPDIR="+tmp)
	}
	want = append(want, "TEI_PROFILE_A=first", "TEI_PROFILE_Z=last=with-equals")
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("unexpected host-approved profile: got=%q want=%q", first, want)
	}

	launcher, err := pruntime.NewStdioProcessLauncherForHostProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	profile.Additional["TEI_PROFILE_A"] = "changed-after-launcher-creation"
	profile.SearchDirectories[0] = filepath.Join(root, "other")
	if !reflect.DeepEqual(launcher.Env, want) {
		t.Fatal("launcher environment must be a snapshot of Host configuration")
	}
}

func TestHostProcessEnvironmentProfileRejectsInvalidDirectoriesAndOverrides(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name    string
		profile pruntime.HostProcessEnvironmentProfile
	}{
		{"relative search", pruntime.HostProcessEnvironmentProfile{SearchDirectories: []string{"bin"}}},
		{"empty search", pruntime.HostProcessEnvironmentProfile{SearchDirectories: []string{""}}},
		{"path list inside one directory", pruntime.HostProcessEnvironmentProfile{SearchDirectories: []string{root + string(os.PathListSeparator) + root}}},
		{"relative home", pruntime.HostProcessEnvironmentProfile{HomeDirectory: "home"}},
		{"relative temp", pruntime.HostProcessEnvironmentProfile{TemporaryDirectory: "tmp"}},
		{"NUL search", pruntime.HostProcessEnvironmentProfile{SearchDirectories: []string{root + "\x00secret"}}},
		{"reserved PATH", pruntime.HostProcessEnvironmentProfile{Additional: map[string]string{"PATH": "/secret"}}},
		{"reserved HOME", pruntime.HostProcessEnvironmentProfile{Additional: map[string]string{"HOME": "/secret"}}},
		{"reserved TMPDIR", pruntime.HostProcessEnvironmentProfile{Additional: map[string]string{"TMPDIR": "/secret"}}},
		{"reserved TEMP", pruntime.HostProcessEnvironmentProfile{Additional: map[string]string{"TEMP": "/secret"}}},
		{"malformed variable", pruntime.HostProcessEnvironmentProfile{Additional: map[string]string{"BAD NAME": "secret"}}},
		{"duplicate case Windows", pruntime.HostProcessEnvironmentProfile{Additional: map[string]string{"TEI_A": "secret", "tei_a": "other"}}},
		{"NUL variable", pruntime.HostProcessEnvironmentProfile{Additional: map[string]string{"TEI_A": "secret\x00other"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.profile.Environment()
			if tc.name == "duplicate case Windows" && runtime.GOOS != "windows" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("invalid environment profile was accepted")
			}
			if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), root) {
				t.Fatal("environment profile diagnostic included a supplied value")
			}
			if _, err := pruntime.NewStdioProcessLauncherForHostProfile(tc.profile); err == nil {
				t.Fatal("launcher constructor accepted invalid environment profile")
			}
		})
	}
}

func TestHostProcessEnvironmentProfileUsesRealChildWithoutHostCanary(t *testing.T) {
	t.Setenv("TEI_PROCESS_HOST_SECRET_CANARY", "host-only-canary")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	launcher, err := pruntime.NewStdioProcessLauncherForHostProfile(pruntime.HostProcessEnvironmentProfile{
		Additional: map[string]string{"TEI_PROCESS_EXPLICIT_ALLOWLIST": "approved"},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := launcher.Invoke(context.Background(), pruntime.ProcessCommand{
		Executable: executable, Args: []string{"--tei-process-env-probe"},
	}, processRequest())
	if err != nil {
		t.Fatal(err)
	}
	checkProcessEnvProbe(t, resp, "approved")
}

func TestHostProcessEnvironmentProfileControlsBareExecutableResolution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows transport requires an explicit executable path")
	}
	t.Setenv("TEI_PROCESS_HOST_SECRET_CANARY", "host-only-canary")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	if err := os.Symlink(executable, filepath.Join(bin, "tei-profile-probe")); err != nil {
		t.Skipf("test filesystem does not support symlinks: %v", err)
	}
	launcher, err := pruntime.NewStdioProcessLauncherForHostProfile(pruntime.HostProcessEnvironmentProfile{
		SearchDirectories: []string{bin},
		Additional:        map[string]string{"TEI_PROCESS_EXPLICIT_ALLOWLIST": "allowed-through-path"},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := launcher.Invoke(context.Background(), pruntime.ProcessCommand{
		Executable: "tei-profile-probe", Args: []string{"--tei-process-env-probe"},
	}, processRequest())
	if err != nil {
		t.Fatal(err)
	}
	var actual struct {
		HostSecretPresent bool   `json:"hostSecretPresent"`
		Explicit          string `json:"explicit"`
	}
	if err := json.Unmarshal(resp.Payload, &actual); err != nil {
		t.Fatal(err)
	}
	if actual.HostSecretPresent || actual.Explicit != "allowed-through-path" {
		t.Fatal("process profile did not constrain the actual child process")
	}
}

func TestHostProcessEnvironmentProfileThroughSandboxRunner(t *testing.T) {
	t.Setenv("TEI_PROCESS_HOST_SECRET_CANARY", "parent-only-profile-secret")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	launcher, err := pruntime.NewStdioProcessLauncherForHostProfile(pruntime.HostProcessEnvironmentProfile{
		Additional: map[string]string{"TEI_PROCESS_EXPLICIT_ALLOWLIST": "sandbox-approved"},
	})
	if err != nil {
		t.Fatal(err)
	}
	network, filesystem, secrets := &netEnforcer{}, &fsEnforcer{}, &secretInjector{}
	host := pruntime.HostSandboxLauncher{
		Network: network, Filesystem: filesystem, Secrets: secrets, Runner: launcher,
	}
	resp, err := host.Invoke(context.Background(), pruntime.SandboxSpec{
		Command: pruntime.ProcessCommand{Executable: executable, Args: []string{"--tei-process-env-probe"}},
		Profile: pruntime.SandboxProfile{
			DenyNetworkByDefault: true, DenyFilesystemByDefault: true,
			SecretInjections: []pruntime.SecretInjection{{
				Scope: "credential", ResourceID: core.MustCanonicalID("prod.secret"),
				Handle: "vault://opaque", Delivery: resource.SecretDeliveryEnvironment,
				Target: "TEI_DATABASE_CREDENTIAL",
			}},
		},
	}, processRequest())
	if err != nil {
		t.Fatal(err)
	}
	checkProcessEnvProbe(t, resp, "sandbox-approved")
	if !network.called || !network.cleaned || !filesystem.called || !filesystem.cleaned || !secrets.called || !secrets.cleaned {
		t.Fatal("Host sandbox lifecycle did not complete")
	}
	// The adapters above are test doubles. No OS sandbox or secret delivery
	// is proven by this test; it verifies profile + Host transport wiring.
}
