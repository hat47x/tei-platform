package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"

	"tei/internal/core"
	pruntime "tei/internal/plugin/runtime"
)

// The independently measured wasm image must be explicitly accepted by
// signed Stage2 Host authority before the CI driver executes that very file.
// This does not itself implement a production browser Host.
func TestReferenceWasmImageRequiresSignedHostAcceptance(t *testing.T) {
	path := os.Getenv("TEI_WASM_ACCEPT_IMAGE")
	digest := os.Getenv("TEI_WASM_ACCEPT_DIGEST")
	if path == "" && digest == "" {
		t.Skip("requires a real js/wasm artifact and its independently measured digest")
	}
	if path == "" || !strings.HasPrefix(digest, "sha256:") || len(digest) != 71 {
		t.Fatal("partial or invalid Wasm artifact acceptance configuration")
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	actual := fmt.Sprintf("sha256:%x", sha256.Sum256(bytes))
	if actual != digest {
		t.Fatalf("Wasm physical image digest mismatch: %s != %s", actual, digest)
	}
	engine := pruntime.RuntimeEngineIdentity{
		ID:                   core.MustCanonicalID("tei.runtime.engine.deterministic_transform.reference"),
		Version:              "1.0.0",
		ContractVersion:      "tei.runtime-engine/deterministic-transform-v1",
		ImplementationDigest: actual,
		Target:               "js/wasm",
	}
	options := writeReferenceStage2Acceptance(t, t.TempDir(), engine)
	result, err := acceptReferenceStage2Engine(options, engine)
	if err != nil {
		t.Fatal(err)
	}
	if result.Engine != engine || result.LockFingerprint == "" ||
		result.HostRegistrationFingerprint == "" || result.HostApprovalFingerprint == "" {
		t.Fatalf("accepted signed Wasm registration is incomplete: %#v", result)
	}
	for _, tc := range []struct {
		name string
		edit func(*pruntime.RuntimeEngineIdentity)
	}{
		{"wrong digest", func(e *pruntime.RuntimeEngineIdentity) {
			e.ImplementationDigest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
		}},
		{"wrong target", func(e *pruntime.RuntimeEngineIdentity) {
			e.Target = "host/linux/amd64"
		}},
		{"wrong contract", func(e *pruntime.RuntimeEngineIdentity) {
			e.ContractVersion = "tei.runtime-engine/other-v1"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			modified := engine
			tc.edit(&modified)
			if _, err := acceptReferenceStage2Engine(options, modified); err == nil {
				t.Fatal("stale signed Host approval authorized a different image or contract")
			}
		})
	}
	noSignature := options
	noSignature.HostSignature = ""
	if _, err := acceptReferenceStage2Engine(noSignature, engine); err == nil {
		t.Fatal("unsigned Host self-declaration was accepted as authority")
	}
	t.Logf("WASM_HOST_APPROVED_DIGEST=%s", actual)
}
