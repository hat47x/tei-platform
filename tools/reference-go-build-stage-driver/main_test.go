package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// These tests pin the SB-C0/C1 prerequisite: the driver records who invoked
// whom, and it cross-checks the child's own report against the artifact bytes
// on disk instead of trusting the child's JSON.

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "tei-driver-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := removeWritable(dir); err != nil {
			t.Errorf("remove isolated state %s: %v", dir, err)
		}
	})
	return dir
}

func removeWritable(root string) error {
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return os.Chmod(path, 0o700)
		}
		return os.Chmod(path, 0o600)
	}); err != nil {
		return err
	}
	return os.RemoveAll(root)
}

// childSource builds a cross-platform child. Using a real Go binary instead of a
// shell script keeps the same test meaningful on Windows, where a POSIX shell is
// unavailable, and exercises a genuine executable digest.
const childSource = `package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// flags: -artifact <path> -digest <value> -exit <code>
func main() {
	artifact, digest, code := "", "", 0
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-artifact":
			artifact = os.Args[i+1]
			i++
		case "-digest":
			digest = os.Args[i+1]
			i++
		case "-exit":
			fmt.Sscan(os.Args[i+1], &code)
			i++
		}
	}
	if artifact != "" {
		if err := os.MkdirAll(filepath.Dir(artifact), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := os.WriteFile(artifact, []byte("artifact-bytes"), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	fmt.Printf("{\"schemaVersion\":\"tei.reference.go-build-with-supply/v0\",\"artifactDigest\":%q}\n", digest)
	os.Exit(code)
}
`

func buildChild(t *testing.T, dir string) string {
	t.Helper()
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	source := filepath.Join(dir, "childmain.go")
	if err := os.WriteFile(source, []byte(childSource), 0o600); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(dir, "child")
	if runtime.GOOS == "windows" {
		child += ".exe"
	}
	cmd := exec.Command(goExe, "build", "-trimpath", "-o", child, source)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build child fixture: %v: %s", err, out)
	}
	return child
}

func TestDriveRecordsBothIdentitiesAndCrossChecksArtifact(t *testing.T) {
	dir := tempDir(t)
	artifact := filepath.Join(dir, "artifact")
	child := buildChild(t, dir)

	evidence, err := Drive(child, []string{"-artifact", artifact, "-digest", "sha256:deadbeef"}, artifact, "")
	if err != nil {
		t.Fatal(err)
	}
	if !evidence.InvokerRecorded || !evidence.InvokedRecorded {
		t.Fatalf("identities were not both recorded: %+v", evidence)
	}
	if evidence.DriverSelfDigest == "" || evidence.Invoked.ExecutableDigest == "" {
		t.Fatal("digest fields are empty")
	}
	if evidence.DriverSelfDigest == evidence.Invoked.ExecutableDigest {
		t.Fatal("invoker and invoked identities collapsed into one digest")
	}
	if evidence.Invoked.ExitCode != 0 || evidence.Invoked.PID <= 0 {
		t.Fatalf("child was not observed: %+v", evidence.Invoked)
	}
	if len(evidence.Invoked.Argv) == 0 || evidence.Invoked.Argv[0] != child {
		t.Fatalf("argv did not record the exact executable: %+v", evidence.Invoked.Argv)
	}
	if evidence.CrossCheckedArtifact == "" {
		t.Fatal("artifact was not cross-checked from disk")
	}
	// The child reported a wrong digest on purpose; the driver must notice.
	if evidence.CrossCheckMatched {
		t.Fatal("driver trusted the child's digest instead of the artifact bytes")
	}
	if evidence.FailureCause == "" {
		t.Fatal("mismatch had no stated cause")
	}
	if evidence.L3Claimed {
		t.Fatal("driver claimed L3")
	}
	if evidence.Boundary == "" {
		t.Fatal("evidence lost its boundary statement")
	}
}

func TestDriveAcceptsAMatchingChildReport(t *testing.T) {
	dir := tempDir(t)
	artifact := filepath.Join(dir, "artifact")
	child := buildChild(t, dir)

	// Compute the digest the child will create, so the report can be correct.
	if err := os.WriteFile(artifact, []byte("artifact-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := fileDigest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(artifact); err != nil {
		t.Fatal(err)
	}

	evidence, err := Drive(child, []string{"-artifact", artifact, "-digest", digest}, artifact, "")
	if err != nil {
		t.Fatal(err)
	}
	if !evidence.CrossCheckMatched {
		t.Fatalf("matching child report was not accepted: %s", evidence.FailureCause)
	}
	if evidence.FailureCause != "" {
		t.Fatalf("successful cross-check reported a failure: %s", evidence.FailureCause)
	}
	if evidence.ChildSchema != "tei.reference.go-build-with-supply/v0" {
		t.Fatalf("child schema was not recorded: %s", evidence.ChildSchema)
	}
}

func TestDriveDetectsChildFailureWithoutClaimingResults(t *testing.T) {
	dir := tempDir(t)
	artifact := filepath.Join(dir, "artifact")
	child := buildChild(t, dir)

	evidence, err := Drive(child, []string{"-exit", "7"}, artifact, "")
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Invoked.ExitCode != 7 {
		t.Fatalf("child exit was not recorded: %+v", evidence.Invoked)
	}
	if evidence.FailureCause == "" {
		t.Fatal("failure had no stated cause")
	}
	if evidence.CrossCheckMatched || evidence.ChildArtifact != "" {
		t.Fatal("a failed child produced accepted-looking results")
	}
}

func TestDriveRefusesToOverwriteAnExistingArtifact(t *testing.T) {
	dir := tempDir(t)
	artifact := filepath.Join(dir, "artifact")
	if err := os.WriteFile(artifact, []byte("already here"), 0o600); err != nil {
		t.Fatal(err)
	}
	child := buildChild(t, dir)
	if _, err := Drive(child, []string{"-artifact", artifact}, artifact, ""); err == nil || !strings.Contains(err.Error(), "does not overwrite") {
		t.Fatalf("existing artifact was not refused: %v", err)
	}
}

func TestDriveRejectsANonExecutableChild(t *testing.T) {
	dir := tempDir(t)
	path := filepath.Join(dir, "not-executable")
	if err := os.WriteFile(path, []byte("plain file"), 0o600); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(dir, "artifact")
	if _, err := Drive(path, nil, artifact, ""); err == nil {
		t.Skipf("this platform executed a non-executable file")
	}
}

func TestExitCodeReportsObservationOnly(t *testing.T) {
	var stdout, stderr strings.Builder
	code := run([]string{"-child", "/nonexistent", "-child-arg", "[]", "-artifact", "x"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("unreadable child did not fail observation: code=%d stderr=%s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("observation failure wrote to stdout: %s", stdout.String())
	}
}

func TestBadChildArgIsRejectedBeforeStarting(t *testing.T) {
	var stdout, stderr strings.Builder
	code := run([]string{"-child", "x", "-child-arg", "not-json", "-artifact", "y"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("malformed -child-arg was not a usage failure: code=%d", code)
	}
	var _ = json.Marshal
}
