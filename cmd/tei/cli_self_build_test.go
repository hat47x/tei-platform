package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// These tests pin the thin `self-build` entry (SB-C0, #2831): this TEI binary
// starts a child, records the invoker and the invoked separately, and re-reads
// the artifact from disk instead of trusting the child's own digest. The entry
// is dispatched but deliberately not advertised in the read-only discovery
// catalog (#3414 §4.5).

const selfBuildChildSource = `package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	artifact, digest, code, state, definition := "", "", 0, "", ""
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
		case "-state":
			state = os.Args[i+1]
			i++
		case "-report-definition":
			definition = os.Args[i+1]
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
	report := fmt.Sprintf("{\"schemaVersion\":\"tei.reference.go-build-with-supply/v0\",\"artifactDigest\":%q", digest)
	if state != "" {
		// A child that reports a state directory is claiming it used it. Creating it
		// is part of that claim: the entry now checks the directory exists, so a
		// fixture that only prints the path would be modelling a child that does
		// not honour -state (#4486 S1).
		if err := os.MkdirAll(state, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		report += fmt.Sprintf(",\"stateDir\":%q", state)
	}
	if definition != "" {
		report += fmt.Sprintf(",\"definitionBytesDigest\":%q", definition)
	}
	report += "}"
	fmt.Println(report)
	os.Exit(code)
}
`

func buildSelfBuildChild(t *testing.T, dir string) string {
	t.Helper()
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	source := filepath.Join(dir, "selfbuildchild.go")
	if err := os.WriteFile(source, []byte(selfBuildChildSource), 0o600); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(dir, "selfbuildchild")
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

func TestSelfBuildIsDispatchedButNotAdvertised(t *testing.T) {
	advertised := false
	for _, record := range commandCatalog().Commands {
		if record.Name == "self-build" {
			advertised = true
		}
	}
	if advertised {
		t.Fatal("self-build was added to the read-only discovery catalog; that needs the D-1 contract decision (#3414 §4.5)")
	}
	dispatched := false
	for _, name := range acceptedCommands {
		if name == "self-build" {
			dispatched = true
		}
	}
	if !dispatched {
		t.Fatal("self-build is not in acceptedCommands, so the CLI rejects it as unknown")
	}
}

func TestSelfBuildRecordsBothIdentitiesAndCrossChecksArtifact(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "artifact")
	child := buildSelfBuildChild(t, dir)

	var stdout bytes.Buffer
	code, err := runSelfBuildCommand([]string{
		"-child", child,
		"-child-arg", childArgs(t, "-artifact", artifact, "-digest", "sha256:deadbeef"),
		"-artifact", artifact,
	}, &stdout)
	if code != 0 || err != nil {
		t.Fatalf("self-build did not complete an observation: code=%d err=%v", code, err)
	}
	var evidence selfBuildEntryEvidence
	if err := json.Unmarshal(stdout.Bytes(), &evidence); err != nil {
		t.Fatalf("self-build did not emit one JSON record: %v", err)
	}
	// The entry advertises its own schema; this is the assertion the catalog
	// coverage check counts for self-build-entry (#4032).
	// The literal is written here as well as the constant, so the repository-wide
	// schema coverage check sees the identifier (#4032).
	if evidence.SchemaVersion != "tei.cli.self-build-entry/v0" {
		t.Fatalf("schemaVersion=%q, want the literal and the constant %q", evidence.SchemaVersion, selfBuildEntrySchema)
	}
	if evidence.SchemaVersion != selfBuildEntrySchema {
		t.Fatalf("schemaVersion=%q, want %q", evidence.SchemaVersion, selfBuildEntrySchema)
	}
	if evidence.InvokerDigest == "" || evidence.Invoked.ExecutableDigest == "" {
		t.Fatalf("identities were not both recorded: %+v", evidence)
	}
	if evidence.InvokerDigest == evidence.Invoked.ExecutableDigest {
		t.Fatal("invoker and invoked identities collapsed into one digest")
	}
	if len(evidence.Invoked.Argv) == 0 || evidence.Invoked.Argv[0] != child {
		t.Fatalf("argv did not record the exact executable: %+v", evidence.Invoked.Argv)
	}
	if evidence.CrossCheckOK {
		t.Fatal("self-build trusted the child's digest instead of the artifact bytes")
	}
	if evidence.FailureCause == "" {
		t.Fatal("mismatch had no stated cause")
	}
	if evidence.L3Claimed {
		t.Fatal("self-build claimed L3")
	}
	if evidence.Boundary == "" {
		t.Fatal("evidence lost its boundary statement")
	}
}

func TestSelfBuildRejectsAClaimedDigestThatDiffersFromTheImage(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "artifact")
	child := buildSelfBuildChild(t, dir)

	var stdout bytes.Buffer
	code, err := runSelfBuildCommand([]string{
		"-child", child,
		"-child-arg", childArgs(t, "-artifact", artifact, "-digest", "sha256:deadbeef"),
		"-artifact", artifact,
		"-claimed-self", "sha256:not-this-process",
	}, &stdout)
	if code != 0 || err != nil {
		t.Fatalf("self-build did not complete an observation: code=%d err=%v", code, err)
	}
	var evidence selfBuildEntryEvidence
	if err := json.Unmarshal(stdout.Bytes(), &evidence); err != nil {
		t.Fatal(err)
	}
	if !evidence.ClaimChecked {
		t.Fatal("a supplied claim was not compared")
	}
	if evidence.ClaimMatched {
		t.Fatal("a wrong claim was reported as matching")
	}
	if !strings.Contains(evidence.FailureCause, "claimed invoker digest differs") {
		t.Fatalf("claim mismatch had no stated cause: %q", evidence.FailureCause)
	}
	if evidence.L3Claimed {
		t.Fatal("a failed claim still claimed L3")
	}
}

// A claim that matches the running image is recorded as checked and matched,
// but it still does not make the attempt L3.
func TestSelfBuildRecordsAMatchingClaimWithoutClaimingL3(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "artifact")
	child := buildSelfBuildChild(t, dir)

	selfPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	selfDigest, err := digestOfFile(selfPath)
	if err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	code, err := runSelfBuildCommand([]string{
		"-child", child,
		"-child-arg", childArgs(t, "-artifact", artifact, "-digest", "sha256:deadbeef"),
		"-artifact", artifact,
		"-claimed-self", selfDigest,
	}, &stdout)
	if code != 0 || err != nil {
		t.Fatalf("self-build did not complete an observation: code=%d err=%v", code, err)
	}
	var evidence selfBuildEntryEvidence
	if err := json.Unmarshal(stdout.Bytes(), &evidence); err != nil {
		t.Fatal(err)
	}
	if !evidence.ClaimChecked || !evidence.ClaimMatched {
		t.Fatalf("a matching claim was not recorded as matched: %+v", evidence)
	}
	if evidence.L3Claimed {
		t.Fatal("matching claim was read as L3")
	}
}

func TestSelfBuildRefusesAnExistingArtifactAndMissingFlags(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "artifact")
	if err := os.WriteFile(artifact, []byte("already here"), 0o600); err != nil {
		t.Fatal(err)
	}
	child := buildSelfBuildChild(t, dir)
	if _, err := driveSelfBuild(child, []string{"-artifact", artifact}, artifact, ""); err == nil ||
		!strings.Contains(err.Error(), "does not overwrite") {
		t.Fatalf("existing artifact was not refused: %v", err)
	}

	for _, args := range [][]string{
		{},
		{"-child", child},
		{"-artifact", artifact},
		{"-child", child, "-artifact", "x", "positional"},
	} {
		code, err := runSelfBuildCommand(args, &bytes.Buffer{})
		if code != 2 || err == nil || !strings.Contains(err.Error(), "INVALID_ARGUMENT") {
			t.Fatalf("args=%q: code=%d err=%v, want INVALID_ARGUMENT and exit 2", args, code, err)
		}
	}
}
