package main

import (
	"runtime"
	"strings"
	"testing"
)

// The boundary is a Linux facility. Off Linux the launcher must refuse rather than
// return a boundary that would silently run unconfined.
func TestStateBoundaryLauncherRefusesOffLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("this test pins the non-Linux answer")
	}
	_, err := stateBoundaryLauncher("/tmp/state", "/tmp/art", "/tmp/first")
	if err == nil {
		t.Fatal("a boundary was returned on a platform without mount namespaces")
	}
	if err.Error() != "state boundary needs linux mount namespaces" {
		t.Fatalf("error = %q", err.Error())
	}
}

// The launcher must name the three allow paths in the environment, must run under
// unshare -rm, and must exec its arguments. Anything less and the second build
// either cannot write where it needs to or is not confined at all.
func TestStateBoundaryLauncherShape(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the launcher only builds on linux")
	}
	boundary, err := stateBoundaryLauncher("/tmp/second-state", "/tmp/art", "/tmp/first-state")
	if err != nil {
		t.Skipf("no boundary in this environment: %v", err)
	}
	if boundary.Name != "linux-mount-namespace" {
		t.Fatalf("name = %q", boundary.Name)
	}
	if len(boundary.Launcher) < 6 || !strings.HasSuffix(boundary.Launcher[0], "unshare") {
		t.Fatalf("launcher = %v, want unshare -rm sh -c <script> sh", boundary.Launcher)
	}
	if boundary.Launcher[1] != "-rm" || boundary.Launcher[2] != "sh" || boundary.Launcher[3] != "-c" {
		t.Fatalf("launcher prefix = %v", boundary.Launcher[:4])
	}

	script := boundary.Launcher[4]
	for _, want := range []string{
		"remount,bind,ro /",
		"remount,bind,rw",
		"mount -t tmpfs none",
		`exec "$@"`,
		"exit 97",
		"exit 98",
		"exit 99",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script is missing %q", want)
		}
	}

	wantEnv := map[string]string{
		"ALLOW_STATE":       "ALLOW_STATE=/tmp/second-state",
		"ALLOW_ART":         "ALLOW_ART=/tmp/art",
		"ALLOW_FIRST_STATE": "ALLOW_FIRST_STATE=/tmp/first-state",
	}
	for _, entry := range boundary.Env {
		name, _, _ := strings.Cut(entry, "=")
		want, ok := wantEnv[name]
		if !ok {
			t.Fatalf("unexpected env entry %q", entry)
		}
		if entry != want {
			t.Fatalf("env %q, want %q", entry, want)
		}
		delete(wantEnv, name)
	}
	if len(wantEnv) != 0 {
		t.Fatalf("env is missing %v", wantEnv)
	}
}

// The scope must not claim that nothing else was writable: the read-only remount
// covers the root tree, and separately mounted filesystems are outside it.
func TestStateBoundaryScopeNamesWhatIsNotCovered(t *testing.T) {
	lower := strings.ToLower(stateSeparationHostBoundaryScope)
	if !strings.Contains(lower, "not covered") {
		t.Fatalf("scope does not say what the boundary leaves out: %q", stateSeparationHostBoundaryScope)
	}
	for _, forbidden := range []string{"only the passed state directory and the artifact directory writable", "nothing else was writable"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("scope overstates the boundary: %q", forbidden)
		}
	}
}
