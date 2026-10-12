package runtime_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/plugin/transform"
)

// realEngineReplayReceipt is evidence, not authority. A replay verifies the
// actual bytes and canonical Source before binding the separately declared
// target-specific Engine to the known Reference Plugin environment.
type realEngineReplayReceipt struct {
	Schema                             string `json:"schema"`
	EngineLogicalID                    string `json:"engineLogicalId"`
	EngineVersion                      string `json:"engineVersion"`
	EngineContract                     string `json:"engineContract"`
	PluginID                           string `json:"pluginId"`
	PluginVersion                      string `json:"pluginVersion"`
	PluginBuildPlanFingerprint         string `json:"pluginBuildPlanFingerprint"`
	NativeTarget                       string `json:"nativeTarget"`
	WasmTarget                         string `json:"wasmTarget"`
	NativeArtifactDigest               string `json:"nativeArtifactDigest"`
	WasmArtifactDigest                 string `json:"wasmArtifactDigest"`
	RealizationArtifactDigest          string `json:"realizationArtifactDigest"`
	NativeRuntimeEngineLockFingerprint string `json:"nativeRuntimeEngineLockFingerprint"`
	WasmRuntimeEngineLockFingerprint   string `json:"wasmRuntimeEngineLockFingerprint"`
}

// Strict JSON is required even for Reference evidence. Encoding/json alone
// permits duplicate keys and lossy Unicode decoding, which would make a
// receipt ambiguous before physical images can be verified.
func decodeRealEngineReplayReceipt(raw []byte) (realEngineReplayReceipt, error) {
	if len(raw) == 0 || len(raw) > 1<<20 {
		return realEngineReplayReceipt{}, fmt.Errorf("runtime Engine receipt size is invalid")
	}
	if _, err := transform.ParseJSON(raw); err != nil {
		return realEngineReplayReceipt{}, fmt.Errorf("ambiguous or invalid runtime Engine receipt: %w", err)
	}
	var receipt realEngineReplayReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return realEngineReplayReceipt{}, err
	}
	return receipt, nil
}

func replayBoundEngineReceipt(receipt realEngineReplayReceipt, nativePath, wasmPath, sourcePath string) error {
	if receipt.Schema != "tei.reference.transform-native-wasm-conformance/v1" ||
		receipt.EngineLogicalID != "tei.runtime.engine.deterministic_transform.reference" ||
		receipt.EngineVersion != "1.0.0" ||
		receipt.EngineContract != "tei.runtime-engine/deterministic-transform-v1" ||
		receipt.NativeTarget != "host/linux/amd64" || receipt.WasmTarget != "js/wasm" {
		return fmt.Errorf("unsupported Reference Engine replay contract or target")
	}
	base := engineLockPlan(pruntime.BoundaryInProcess)
	expected := base.Plugins[0]
	if receipt.PluginID != string(expected.Plugin.ID) ||
		receipt.PluginVersion != expected.Plugin.Version ||
		receipt.PluginBuildPlanFingerprint != expected.BuildPlanFingerprint {
		return fmt.Errorf("Reference Plugin realization identity is not the receipt identity")
	}
	for _, pair := range []struct{ path, digest string }{
		{nativePath, receipt.NativeArtifactDigest},
		{wasmPath, receipt.WasmArtifactDigest},
	} {
		if !strings.HasPrefix(pair.digest, "sha256:") || len(pair.digest) != 71 {
			return fmt.Errorf("artifact digest is not exact sha256")
		}
		data, err := os.ReadFile(pair.path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		if fmt.Sprintf("sha256:%x", sum) != pair.digest {
			return fmt.Errorf("physical Runtime Engine bytes disagree with Evidence")
		}
	}
	if receipt.NativeArtifactDigest == receipt.WasmArtifactDigest {
		return fmt.Errorf("different targets require independently identified physical Engine images")
	}
	sourceRaw, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	source, err := transform.ParseSourceJSON(sourceRaw)
	if err != nil {
		return err
	}
	canonical, _, err := transform.MaterializeSource(source)
	if err != nil {
		return err
	}
	realizationSum := sha256.Sum256(canonical)
	if receipt.RealizationArtifactDigest != fmt.Sprintf("sha256:%x", realizationSum) {
		return fmt.Errorf("canonical Source realization disagrees with Evidence")
	}

	base.Plugins[0].Artifact = plugin.Artifact{Digest: receipt.RealizationArtifactDigest}
	bind := func(target, physicalDigest, expectedFingerprint string) error {
		engine := pruntime.RuntimeEngineIdentity{
			ID                  : core.MustCanonicalID(receipt.EngineLogicalID),
			Version             : receipt.EngineVersion,
			ContractVersion     : receipt.EngineContract,
			ImplementationDigest: physicalDigest,
			Target              : target,
		}
		lock := pruntime.RuntimeEngineLock{
			FormatVersion: pruntime.RuntimeEngineLockFormatVersion,
			Bindings: []pruntime.RuntimeEngineBinding{{
				Plugin: base.Plugins[0].Plugin,
				Engine: engine,
			}},
		}
		bound, err := pruntime.BindRuntimeEngineLock(base, lock)
		if err != nil {
			return err
		}
		if expectedFingerprint == "" || bound.RuntimeEngineLockFingerprint != expectedFingerprint {
			return fmt.Errorf("reconstructed Runtime Engine Lock fingerprint disagrees with Evidence")
		}
		if len(bound.Plugins) != 1 || bound.Plugins[0].RuntimeEngine == nil ||
			*bound.Plugins[0].RuntimeEngine != engine ||
			bound.Plugins[0].Artifact.Digest != receipt.RealizationArtifactDigest {
			return fmt.Errorf("replayed plan lost physical Engine or exact realization")
		}
		return nil
	}
	if err := bind(receipt.NativeTarget, receipt.NativeArtifactDigest, receipt.NativeRuntimeEngineLockFingerprint); err != nil {
		return fmt.Errorf("native replay: %w", err)
	}
	if err := bind(receipt.WasmTarget, receipt.WasmArtifactDigest, receipt.WasmRuntimeEngineLockFingerprint); err != nil {
		return fmt.Errorf("js/wasm replay: %w", err)
	}
	if receipt.NativeRuntimeEngineLockFingerprint == receipt.WasmRuntimeEngineLockFingerprint {
		return fmt.Errorf("different targets collapsed to the same Engine Lock")
	}
	return nil
}

func TestActualEngineLockReceiptReplaysWithExactFiles(t *testing.T) {
	evidencePath := os.Getenv("TEI_ENGINE_REPLAY_EVIDENCE")
	nativePath := os.Getenv("TEI_ENGINE_REPLAY_NATIVE")
	wasmPath := os.Getenv("TEI_ENGINE_REPLAY_WASM")
	if evidencePath == "" && nativePath == "" && wasmPath == "" {
		t.Skip("requires actual two-target Runtime Engine evidence from the dedicated CI")
	}
	if evidencePath == "" || nativePath == "" || wasmPath == "" {
		t.Fatal("partial exact Runtime Engine replay inputs")
	}
	raw, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := decodeRealEngineReplayReceipt(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []struct {
		name string
		raw  string
	}{
		{name: "duplicate JSON member", raw: `{"schema":"a","schema":"b"}`},
		{name: "unpaired Unicode surrogate", raw: `{"schema":"\uD800"}`},
		{name: "trailing second root", raw: `{"schema":"a"} {"schema":"b"}`},
	} {
		t.Run(bad.name, func(t *testing.T) {
			if _, err := decodeRealEngineReplayReceipt([]byte(bad.raw)); err == nil {
				t.Fatal("ambiguous evidence JSON was accepted")
			}
		})
	}
	sourcePath := filepath.Join("..", "..", "..", "reference", "plugin-self-build",
		"builder-sources", "transform-source-realizer-v1.transform.json")
	assertReplay := func(name string, receipt realEngineReplayReceipt, native, wasm, source string, valid bool) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			err := replayBoundEngineReceipt(receipt, native, wasm, source)
			if valid && err != nil {
				t.Fatalf("exact verified replay was rejected: %v", err)
			}
			if !valid && err == nil {
				t.Fatal("stale or falsified replay evidence was accepted")
			}
		})
	}
	assertReplay("exact native and wasm images", receipt, nativePath, wasmPath, sourcePath, true)
	assertReplay("swapped native and wasm images", receipt, wasmPath, nativePath, sourcePath, false)

	wrongEngine := receipt
	wrongEngine.WasmArtifactDigest = receipt.NativeArtifactDigest
	assertReplay("swapped physical digest", wrongEngine, nativePath, wasmPath, sourcePath, false)

	wrongLock := receipt
	wrongLock.WasmRuntimeEngineLockFingerprint = receipt.NativeRuntimeEngineLockFingerprint
	assertReplay("stale wasm Engine Lock", wrongLock, nativePath, wasmPath, sourcePath, false)

	wrongTarget := receipt
	wrongTarget.WasmTarget = "host/linux/amd64"
	assertReplay("wrong wasm target", wrongTarget, nativePath, wasmPath, sourcePath, false)

	wrongSource := receipt
	wrongSource.RealizationArtifactDigest = "sha256:" + strings.Repeat("0", 64)
	assertReplay("changed Source realization", wrongSource, nativePath, wasmPath, sourcePath, false)

	wrongPlugin := receipt
	wrongPlugin.PluginVersion = "2.0.0"
	assertReplay("wrong logical Plugin version", wrongPlugin, nativePath, wasmPath, sourcePath, false)

	wrongPlan := receipt
	wrongPlan.PluginBuildPlanFingerprint = strings.Repeat("0", 64)
	assertReplay("wrong Plugin Build Plan", wrongPlan, nativePath, wasmPath, sourcePath, false)

	wrongEngineID := receipt
	wrongEngineID.EngineLogicalID = "tei.runtime.engine.untrusted"
	assertReplay("untrusted Engine identity", wrongEngineID, nativePath, wasmPath, sourcePath, false)

	altered := filepath.Join(t.TempDir(), "altered.wasm")
	if err := os.WriteFile(altered, []byte("altered physical wasm bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	assertReplay("tampered wasm physical image", receipt, nativePath, altered, sourcePath, false)
}
