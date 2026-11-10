package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// §4.21's claim is that one builder, run twice, produced the same bytes. Two of
// the three inputs to that claim were recorded but not checked: which builder ran
// each time. These tests pin the verdict, and the record, to the builder.

func TestReproducibilityBuilderVerdict(t *testing.T) {
	cases := []struct {
		name          string
		first, second string
		want          string
	}{
		{
			name:   "the same builder ran twice",
			first:  "sha256:aaaa",
			second: "sha256:aaaa",
			want:   "",
		},
		{
			name:   "a different builder ran the second time",
			first:  "sha256:aaaa",
			second: "sha256:bbbb",
			want:   "the two runs used different builder executables",
		},
		{
			// An unrecorded digest is not agreement: SC0-18's rule, applied here.
			name:   "the first digest was not recorded",
			first:  "",
			second: "sha256:bbbb",
			want:   "the builder executable digest was not recorded for both runs",
		},
		{
			name:   "the second digest was not recorded",
			first:  "sha256:aaaa",
			second: "",
			want:   "the builder executable digest was not recorded for both runs",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := reproducibilityBuilderVerdict(tc.first, tc.second); got != tc.want {
				t.Fatalf("verdict = %q, want %q", got, tc.want)
			}
		})
	}
}

// The record must carry the builder's digest on both runs, not the entry's. The
// two are different files, so a mix-up is visible in the report.
//
// The check needs a child that honours -artifact-out and -state, so this uses the
// same real child as the L2-b gate rather than the stub.
func TestReproducibilityRecordsTheBuildersDigestAndNotTheEntrys(t *testing.T) {
	cli := buildToolBinary(t, ".", "tei")
	builderChild := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-plugin-package-build-child"), "build-child")
	resolver := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-build-authoring-intent"), "resolver")

	root, intentPath, sourceRoot, _ := setupL2BFixture(t)
	workspace := outerWorkspace(t, root)
	catalogPath, catalogDigest := authorCatalog(t, root, sourceRoot, builderChild)

	planPath := filepath.Join(root, "plan.json")
	resolveCmd := exec.Command(resolver,
		"-intent", intentPath,
		"-catalog", catalogPath,
		"-source-root", sourceRoot,
	)
	var resolveStdout, resolveStderr bytes.Buffer
	resolveCmd.Stdout, resolveCmd.Stderr = &resolveStdout, &resolveStderr
	if err := resolveCmd.Run(); err != nil {
		t.Fatalf("reference resolver: %v\n%s", err, resolveStderr.String())
	}
	var receipt struct {
		Plan json.RawMessage `json:"plan"`
	}
	if err := json.Unmarshal(resolveStdout.Bytes(), &receipt); err != nil {
		t.Fatalf("decode the reference receipt: %v", err)
	}
	if err := os.WriteFile(planPath, append(receipt.Plan, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}

	childArg, err := json.Marshal([]string{
		"-plan", planPath,
		"-plugin", filepath.Join(sourceRoot, "plugin.tei"),
		"-intent", intentPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	childDigest, err := digestOfFile(builderChild)
	if err != nil {
		t.Fatal(err)
	}
	entryDigest, err := digestOfFile(cli)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(cli,
		"self-build",
		"-workspace", workspace,
		"-intent", "tei.plugin.l2b_e2e",
		"-intent-file", intentPath,
		"-build-catalog", catalogPath,
		"-build-catalog-digest", catalogDigest,
		"-build-source-root", sourceRoot,
		"-child", builderChild,
		"-child-arg", string(childArg),
		"-state", filepath.Join(root, "state"),
		"-output", filepath.Join(root, "package.json"),
	)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	_ = cmd.Run()

	var report struct {
		ArtifactReproducibility *struct {
			State             string `json:"state"`
			FirstChildDigest  string `json:"firstChildDigest"`
			SecondChildDigest string `json:"secondChildDigest"`
			MismatchReason    string `json:"mismatchReason"`
		} `json:"artifactReproducibility"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode the report: %v\n%s", err, stdout.String())
	}
	if report.ArtifactReproducibility == nil {
		t.Fatalf("no reproducibility record:\n%s", stdout.String())
	}
	record := report.ArtifactReproducibility
	if record.State != reproducibilityReproduced {
		t.Fatalf("state = %q (%s)\n%s", record.State, record.MismatchReason, stderr.String())
	}
	if record.FirstChildDigest != childDigest || record.SecondChildDigest != childDigest {
		t.Fatalf("builder digests = %q / %q, want %q for both",
			record.FirstChildDigest, record.SecondChildDigest, childDigest)
	}
	// The entry's own digest must not appear: that is the mix-up this test exists
	// to catch.
	if record.FirstChildDigest == entryDigest || record.SecondChildDigest == entryDigest {
		t.Fatal("the record carries the entry's digest where the builder's belongs")
	}
}
