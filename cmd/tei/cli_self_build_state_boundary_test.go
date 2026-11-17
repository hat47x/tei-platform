package main

import (
	"os"
	"path/filepath"
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
	_, err := stateBoundaryLauncher("/tmp/state", "/tmp/art", "/tmp/first", "sha256:proof")
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
	boundary, err := stateBoundaryLauncher("/tmp/second-state", "/tmp/art", "/tmp/first-state", "sha256:proof")
	if err != nil {
		t.Skipf("no boundary in this environment: %v", err)
	}
	if boundary.Name != "linux-mount-namespace" {
		t.Fatalf("name = %q", boundary.Name)
	}
	if len(boundary.Launcher) < 6 || !strings.HasSuffix(boundary.Launcher[0], "unshare") {
		t.Fatalf("launcher = %v, want unshare -rm sh -c <script> sh", boundary.Launcher)
	}
	// unshare -rm --propagation private sh -c <script> sh: the propagation flag must be
	// present, because a shared / on another host would let the read-only remount leak
	// out of the namespace and make that host read-only.
	wantPrefix := []string{"-rm", "--propagation", "private", "sh", "-c"}
	if len(boundary.Launcher) < len(wantPrefix)+2 {
		t.Fatalf("launcher too short: %v", boundary.Launcher)
	}
	for i, want := range wantPrefix {
		if boundary.Launcher[1+i] != want {
			t.Fatalf("launcher[%d] = %q, want %q (full: %v)", 1+i, boundary.Launcher[1+i], want, boundary.Launcher[:len(wantPrefix)+1])
		}
	}
	if boundary.Launcher[len(boundary.Launcher)-1] != "sh" {
		t.Fatalf("launcher must end with the sh argv0 placeholder: %v", boundary.Launcher)
	}

	// The script must prove the remount took effect, not merely report that mount
	// returned success: a launcher that skips the script would otherwise look the
	// same as one that confined the build.
	if !strings.Contains(strings.Join(boundary.Launcher, " "), "refusing to run unconfined") {
		t.Fatalf("script does not verify the remount took effect: %v", boundary.Launcher)
	}

	script := boundary.Launcher[1+len(wantPrefix)]
	for _, want := range []string{
		"remount,bind,ro /",
		"remount,bind,rw",
		"mount -t tmpfs none",
		`exec "$@"`,
		"exit 96",
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
		"ALLOW_PROOF":       "ALLOW_PROOF=sha256:proof",
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

// The scope must not claim that nothing else was writable. Measured on this host:
// /tmp, /proc, /dev, /sys, /run and /home are refused, but /dev/shm and
// /run/user/<uid> still accept writes because they are separate mounts. Reads were
// never confined. The wording has to say all three things.
func TestStateBoundaryScopeNamesWhatIsNotCovered(t *testing.T) {
	lower := strings.ToLower(stateSeparationHostBoundaryScope)
	for _, want := range []string{"reads are not confined", "remain writable", "/dev/shm"} {
		if !strings.Contains(lower, want) {
			t.Fatalf("scope does not mention %q: %q", want, stateSeparationHostBoundaryScope)
		}
	}
	for _, forbidden := range []string{
		"only the passed state directory and the artifact directory writable",
		"nothing else was writable",
		"no other writes",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("scope overstates the boundary: %q", forbidden)
		}
	}
}

// The launcher script reserves 96-99. A second build that exits with one of those was
// stopped by the boundary, so the boundary cannot be recorded as having confined it.
// Before this verdict existed, a non-confining launcher produced claimed=true.
func TestStateBoundaryRanVerdict(t *testing.T) {
	for _, code := range []int{96, 97, 98, 99} {
		if got := stateBoundaryRanVerdict(code); got == "" {
			t.Fatalf("exit %d should be read as the boundary refusing the build", code)
		}
	}
	for _, code := range []int{0, 1, 2, 95, 100} {
		if got := stateBoundaryRanVerdict(code); got != "" {
			t.Fatalf("exit %d should not be attributed to the boundary: %q", code, got)
		}
	}
}

// The entry must not take the boundary as having run merely because it built one. The
// script writes this nonce; a launcher that skips the script leaves it absent, which a
// non-confining launcher was measured to produce.
func TestStateBoundaryProofVerdict(t *testing.T) {
	dir := t.TempDir()
	proof := "sha256:abc"
	if got := stateBoundaryProofVerdict(dir, proof); got == "" {
		t.Fatal("a missing proof should be refused")
	}

	if err := os.WriteFile(filepath.Join(dir, stateBoundaryProofFile), []byte(proof), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := stateBoundaryProofVerdict(dir, proof); got != "" {
		t.Fatalf("matching proof verdict = %q, want none", got)
	}
	if got := stateBoundaryProofVerdict(dir, "sha256:other"); got == "" {
		t.Fatal("a mismatched proof should be refused")
	}
	if got := stateBoundaryProofVerdict(dir, "  "); got == "" {
		t.Fatal("an empty expectation should be refused")
	}
}

func TestStateBoundaryProofValueIsStable(t *testing.T) {
	a := stateBoundaryProofValue("/tmp/s1", "/tmp/f1")
	b := stateBoundaryProofValue("/tmp/s1", "/tmp/f1")
	c := stateBoundaryProofValue("/tmp/s2", "/tmp/f1")
	if a != b {
		t.Fatalf("same inputs gave %q and %q", a, b)
	}
	if a == c {
		t.Fatal("different state directories must not share a proof")
	}
	if !strings.HasPrefix(a, "sha256:") {
		t.Fatalf("proof = %q", a)
	}
}
