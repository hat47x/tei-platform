package compositionprocessbridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tei/internal/composition"
)

// This is a synthetic read-only contract comparison, not a TEI wire ABI,
// formal conformance authority, production Runtime Engine, or OS sandbox.
func TestNeutralContractSwitchesGoAndRealJVMWithoutKernelChanges(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows JVM process behavior has not been qualified")
	}
	javac, err := exec.LookPath("javac")
	if err != nil {
		t.Skip("JDK unavailable: actual JVM execution not verified")
	}
	java, err := exec.LookPath("java")
	if err != nil {
		t.Skip("JVM unavailable: actual JVM execution not verified")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	dir := t.TempDir()
	compile := exec.CommandContext(ctx, javac, "-encoding", "UTF-8", "-d", dir, filepath.Join("testdata", "java", "NeutralAsciiUpperProvider.java"))
	compile.Env = []string{}
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile JVM fixture: %v; output=%s", err, output)
	}
	classPath := filepath.Join(dir, "NeutralAsciiUpperProvider.class")
	javaDigest, err := neutralReferenceFileDigest(classPath)
	if err != nil {
		t.Fatal(err)
	}
	goBinary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	goDigest, err := neutralReferenceFileDigest(goBinary)
	if err != nil {
		t.Fatal(err)
	}

	// The shape is a synthetic fixture fingerprint, not an accepted public
	// Contract or an independent issuer's conformance evidence.
	shape := sha256.Sum256([]byte("ascii-lower-1..32 -> ascii-upper-1..32; read-only; reject invalid input"))
	contract := composition.Contract{ID: "reference.text.uppercase", Version: "1.0.0", Shape: hex.EncodeToString(shape[:])}
	const scope = "reference-tenant"
	registry := composition.NewRegistry()
	_, err = registry.Apply(0, []composition.Provider{
		{ID: "go-inprocess", Scope: scope, Offers: contract},
		{ID: "java-process", Scope: scope, Offers: contract},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selectedGo, err := registry.Select("go-inprocess", scope)
	if err != nil || selectedGo.Contract != contract {
		t.Fatalf("wrong initial selection: %+v %v", selectedGo, err)
	}
	if _, err := registry.Select("java-process", "other-scope"); !errors.Is(err, composition.ErrSelectionMismatch) {
		t.Fatalf("cross-scope JVM selection accepted: %v", err)
	}

	digests := map[string]string{"go-inprocess": goDigest, "java-process": javaDigest}
	allowed := true // Host-owned test flag; not a production authority.
	hostGuard := func(b composition.Binding) error {
		if !allowed || b.ContractID != contract.ID || b.ContractVersion != contract.Version || digests[b.ProviderID] != b.ArtifactDigest {
			return errors.New("test host rejected runtime binding")
		}
		return nil
	}
	goBinding := composition.Binding{ContractID: contract.ID, ContractVersion: contract.Version, ProviderID: "go-inprocess", ArtifactDigest: goDigest, Generation: 1}
	oldScope := composition.NewScope()
	oldCleanups := 0
	if err := oldScope.Track(func() error { oldCleanups++; return nil }); err != nil {
		t.Fatal(err)
	}
	coordinator, err := composition.NewSelectedCoordinator(registry, selectedGo, goBinding, oldScope, hostGuard)
	if err != nil {
		t.Fatal(err)
	}
	launches := 0

	// This physical dispatch is deliberately test-local. Only the existing
	// Composition Coordinator selects the Provider and holds the Generation.
	invoke := func(expected composition.Binding, input string) (string, error) {
		lease, err := coordinator.Acquire()
		if err != nil {
			return "", err
		}
		defer lease.Release()
		if lease.Binding != expected {
			return "", errors.New("pinned generation differs from Host-accepted binding")
		}
		if len(input) == 0 || len(input) > 32 {
			return "", errors.New("invalid fixture input")
		}
		for i := 0; i < len(input); i++ {
			if input[i] < 'a' || input[i] > 'z' {
				return "", errors.New("invalid fixture input")
			}
		}
		artifact := goBinary
		if lease.Binding.ProviderID == "java-process" {
			artifact = classPath
		}
		actual, err := neutralReferenceFileDigest(artifact)
		if err != nil || actual != lease.Binding.ArtifactDigest {
			return "", errors.New("test artifact bytes do not match the binding")
		}
		switch lease.Binding.ProviderID {
		case "go-inprocess":
			return strings.ToUpper(input), nil
		case "java-process":
			cmd := exec.CommandContext(ctx, java, "-cp", dir, "NeutralAsciiUpperProvider")
			cmd.Env = []string{} // Do not inherit ambient secrets.
			cmd.Stdin = strings.NewReader(input + "\n")
			cmd.Stderr = io.Discard
			launches++
			output, err := cmd.Output()
			if err != nil {
				return "", fmt.Errorf("test JVM invocation failed: %w", err)
			}
			if string(output) != strings.ToUpper(input)+"\n" {
				return "", errors.New("JVM response differs from synthetic contract")
			}
			return strings.TrimSuffix(string(output), "\n"), nil
		default:
			return "", errors.New("unrecognized test realization")
		}
	}
	first, err := invoke(goBinding, "tei")
	if err != nil || first != "TEI" || launches != 0 {
		t.Fatalf("Go provider: %q %v launches=%d", first, err, launches)
	}
	if _, err := invoke(goBinding, "invalid input"); err == nil || launches != 0 {
		t.Fatal("invalid input was not rejected before runtime execution")
	}

	inFlight, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	selectedJava, err := registry.Select("java-process", scope)
	if err != nil || selectedJava.Contract != contract {
		inFlight.Release()
		t.Fatalf("unaccepted JVM contract: %+v %v", selectedJava, err)
	}
	javaBinding := composition.Binding{ContractID: contract.ID, ContractVersion: contract.Version, ProviderID: "java-process", ArtifactDigest: javaDigest, Generation: 2}
	err = coordinator.ReplaceSelected(ctx, registry, selectedJava, 1, javaBinding, composition.Lifecycle{
		Prepare: func(_ context.Context, scope *composition.Scope) error {
			return scope.Track(func() error { return nil })
		},
		Ready: func(_ context.Context, binding composition.Binding) error {
			digest, err := neutralReferenceFileDigest(classPath)
			if err != nil || digest != javaDigest || binding != javaBinding {
				return errors.New("JVM fixture not ready")
			}
			return nil
		},
	})
	if err != nil {
		inFlight.Release()
		t.Fatal(err)
	}
	if oldCleanups != 0 || inFlight.Binding != goBinding {
		inFlight.Release()
		t.Fatal("in-flight native generation was not retained")
	}
	second, err := invoke(javaBinding, "tei")
	if err != nil || second != first || launches != 1 {
		inFlight.Release()
		t.Fatalf("JVM result: %q %v launches=%d", second, err, launches)
	}
	inFlight.Release()
	if err := coordinator.Reap(ctx); err != nil || oldCleanups != 1 {
		t.Fatalf("old generation cleanup: %d %v", oldCleanups, err)
	}

	allowed = false
	if _, err := invoke(javaBinding, "tei"); err == nil || launches != 1 {
		t.Fatalf("Host refusal launched JVM: %v; count=%d", err, launches)
	}
	allowed = true
	file, err := os.OpenFile(classPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.Write([]byte("altered fixture bytes"))
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatalf("modify fixture: %v %v", writeErr, closeErr)
	}
	if _, err := invoke(javaBinding, "tei"); err == nil || launches != 1 {
		t.Fatalf("modified JVM class launched: %v; count=%d", err, launches)
	}
	if err := coordinator.Withdraw(2); err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Reap(ctx); err != nil {
		t.Fatal(err)
	}
	t.Log("real JVM process invoked once; the ASCII line carrier is test-only, not the TEI process ABI")
}

func neutralReferenceFileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil)), nil
}
