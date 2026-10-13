package gobuilder

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveReproducibilityPreservesReferenceDecision(t *testing.T) {
	first := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	second := "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if reproducible, artifact := ResolveReproducibility(first, second); reproducible || artifact != "" {
		t.Fatalf("divergent attempts promoted to accepted candidate: %v %q", reproducible, artifact)
	}
	if reproducible, artifact := ResolveReproducibility(first, first); !reproducible || artifact != first {
		t.Fatalf("matching attempts did not retain candidate digest: %v %q", reproducible, artifact)
	}
	// The decision does not validate the digest. Evidence formatting and
	// artifact acceptance remain responsibilities of each Reference.
	if reproducible, artifact := ResolveReproducibility("", ""); !reproducible || artifact != "" {
		t.Fatalf("comparison contract unexpectedly added acceptance policy: %v %q", reproducible, artifact)
	}
}

func TestBuildPackageUsesExactInvocationAndIsolatedEnvironment(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("fake executable is a Linux shell fixture")
	}
	root := t.TempDir()
	builder := filepath.Join(root, "go-fixture")
	// Confirm actual subprocess argv and environment rather than simply
	// inspecting command construction or the Environment helper in isolation.
	script := `#!/bin/sh
set -eu
if [ "$#" -ne 7 ] || [ "$1" != build ] || [ "$2" != -trimpath ] || [ "$3" != -buildvcs=false ] || [ "$4" != -mod=readonly ] || [ "$5" != -o ] || [ "$7" != ./cmd/demo ]; then
  echo "unexpected go build argv" >&2
  exit 21
fi
if [ "${TEI_GOBUILDER_SENTINEL+present}" = present ] || [ "${GOFLAGS-unset}" != "" ] || [ "${GOPROXY-unset}" != off ] || [ "${GOTOOLCHAIN-unset}" != local ] || [ "${CGO_ENABLED-unset}" != 0 ]; then
  echo "Go build inherited ambient settings" >&2
  exit 22
fi
if [ ! -d "$HOME" ] || [ ! -d "$TMPDIR" ] || [ -z "${GOCACHE-unset}" ] || [ -z "${GOMODCACHE-unset}" ]; then
  echo "Go build state is not isolated" >&2
  exit 23
fi
printf "compiled artifact" > "$6"
`
	if err := os.WriteFile(builder, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEI_GOBUILDER_SENTINEL", "do-not-inherit")
	var outputs [2]string
	for i, name := range []string{"first", "second"} {
		outputs[i] = filepath.Join(root, name+"-artifact")
		if err := BuildPackage(root, builder, "./cmd/demo", outputs[i], filepath.Join(root, name+"-state")); err != nil {
			t.Fatalf("%s actual build failed: %v", name, err)
		}
	}
	first, err := os.ReadFile(outputs[0])
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(outputs[1])
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != "compiled artifact" || string(second) != string(first) {
		t.Fatalf("two isolated executions did not create matching output: %q / %q", first, second)
	}
}

func TestBuildPackageReportsSubprocessFailureWithoutExportingArtifact(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("fake executable is a Linux shell fixture")
	}
	root := t.TempDir()
	builder := filepath.Join(root, "go-failure-fixture")
	if err := os.WriteFile(builder, []byte("#!/bin/sh\necho 'synthetic compiler failure' >&2\nexit 17\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "not-created")
	err := BuildPackage(root, builder, "./cmd/broken", output, filepath.Join(root, "build-state"))
	if err == nil || !strings.Contains(err.Error(), "go build ./cmd/broken") || !strings.Contains(err.Error(), "synthetic compiler failure") {
		t.Fatalf("build execution error lost exact package and stderr: %v", err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("failed subprocess unexpectedly created an artifact: %v", err)
	}
}

func TestBuildPackageRejectsInvalidStateWithoutRunningBuilder(t *testing.T) {
	dir := t.TempDir()
	state := filepath.Join(dir, "occupied")
	if err := os.WriteFile(state, []byte("file, not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := BuildPackage(dir, filepath.Join(dir, "missing-go"), "./cmd/demo", filepath.Join(dir, "out"), state)
	if err == nil || !strings.Contains(err.Error(), "create isolated build state") {
		t.Fatalf("invalid isolation state was not rejected before build: %v", err)
	}
}
