package runtime_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/plugin/transform"
)

// This test is activated by the dedicated native/js-wasm CI after it builds
// both real artifacts. It does not confer production Host authority.
func TestActualTransformEngineArtifactsBindDistinctRuntimeEngineLocks(t *testing.T) {
	nativePath := os.Getenv("TEI_TARGET_BIND_NATIVE")
	wasmPath := os.Getenv("TEI_TARGET_BIND_WASM")
	nativeExpected := os.Getenv("TEI_TARGET_BIND_NATIVE_DIGEST")
	wasmExpected := os.Getenv("TEI_TARGET_BIND_WASM_DIGEST")
	if nativePath == "" && wasmPath == "" && nativeExpected == "" && wasmExpected == "" {
		t.Skip("requires real native and js/wasm artifacts from the target-conformance workflow")
	}
	if nativePath == "" || wasmPath == "" || nativeExpected == "" || wasmExpected == "" {
		t.Fatal("partial real-artifact binding configuration")
	}

	verifyBytes := func(path, expected string) error {
		if !strings.HasPrefix(expected, "sha256:") || len(expected) != 71 {
			return fmt.Errorf("expected exact sha256 digest")
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		if got := fmt.Sprintf("sha256:%x", sum); got != expected {
			return fmt.Errorf("actual Engine bytes %s disagree with lock %s", got, expected)
		}
		return nil
	}
	if err := verifyBytes(nativePath, nativeExpected); err != nil {
		t.Fatal(err)
	}
	if err := verifyBytes(wasmPath, wasmExpected); err != nil {
		t.Fatal(err)
	}
	if nativeExpected == wasmExpected {
		t.Fatal("different target Engine artifacts require different physical digests")
	}
	if err := verifyBytes(nativePath, wasmExpected); err == nil {
		t.Fatal("swapping physical target artifacts was accepted")
	}
	if err := verifyBytes(wasmPath, nativeExpected); err == nil {
		t.Fatal("reverse target artifact swap was accepted")
	}
	altered := append([]byte(nil), []byte("tampered-engine-image")...)
	alteredPath := t.TempDir() + "/tampered.wasm"
	if err := os.WriteFile(alteredPath, altered, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyBytes(alteredPath, wasmExpected); err == nil {
		t.Fatal("tampered physical Wasm image was accepted")
	}

	sourceRaw, err := os.ReadFile("../../../reference/plugin-self-build/builder-sources/transform-source-realizer-v1.transform.json")
	if err != nil {
		t.Fatal(err)
	}
	source, err := transform.ParseSourceJSON(sourceRaw)
	if err != nil {
		t.Fatal(err)
	}
	canonical, _, err := transform.MaterializeSource(source)
	if err != nil {
		t.Fatal(err)
	}
	realizationSum := sha256.Sum256(canonical)
	realizationDigest := fmt.Sprintf("sha256:%x", realizationSum)

	base := engineLockPlan(pruntime.BoundaryInProcess)
	base.Plugins[0].Artifact = plugin.Artifact{Digest: realizationDigest}
	pluginIdentity := base.Plugins[0].Plugin
	bind := func(target, digest string) pruntime.EnvironmentPlan {
		t.Helper()
		engine := pruntime.RuntimeEngineIdentity{
			ID:                   core.MustCanonicalID("tei.runtime.engine.deterministic_transform.reference"),
			Version:              "1.0.0",
			ContractVersion:      "tei.runtime-engine/deterministic-transform-v1",
			ImplementationDigest: digest,
			Target:               target,
		}
		lock := pruntime.RuntimeEngineLock{
			FormatVersion: pruntime.RuntimeEngineLockFormatVersion,
			Bindings: []pruntime.RuntimeEngineBinding{{
				Plugin: pluginIdentity,
				Engine: engine,
			}},
		}
		plan, err := pruntime.BindRuntimeEngineLock(base, lock)
		if err != nil {
			t.Fatal(err)
		}
		if plan.Plugins[0].Plugin != pluginIdentity ||
			plan.Plugins[0].Artifact.Digest != realizationDigest ||
			plan.Plugins[0].BuildPlanFingerprint != base.Plugins[0].BuildPlanFingerprint {
			t.Fatal("target binding modified logical Plugin identity or realization")
		}
		if plan.Plugins[0].RuntimeEngine == nil ||
			*plan.Plugins[0].RuntimeEngine != engine {
			t.Fatal("bound plan does not contain the exact real Engine artifact")
		}
		return plan
	}
	native := bind("host/linux/amd64", nativeExpected)
	wasm := bind("js/wasm", wasmExpected)
	if native.RuntimeEngineLockFingerprint == wasm.RuntimeEngineLockFingerprint ||
		native.RuntimePolicyFingerprint == wasm.RuntimePolicyFingerprint {
		t.Fatal("two physical targets collapsed to the same Runtime Engine Lock or policy")
	}
	if err := verifyBytes(nativePath, native.Plugins[0].RuntimeEngine.ImplementationDigest); err != nil {
		t.Fatal(err)
	}
	if err := verifyBytes(wasmPath, wasm.Plugins[0].RuntimeEngine.ImplementationDigest); err != nil {
		t.Fatal(err)
	}
	t.Logf("TARGET_NATIVE_LOCK_SHA256=%s", native.RuntimeEngineLockFingerprint)
	t.Logf("TARGET_WASM_LOCK_SHA256=%s", wasm.RuntimeEngineLockFingerprint)
	t.Logf("TARGET_REALIZATION_SHA256=%s", realizationDigest)
}
