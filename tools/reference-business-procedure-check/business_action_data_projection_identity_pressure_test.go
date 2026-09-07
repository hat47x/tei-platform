package main

import (
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

func TestBusinessActionDataProjectionFailsClosedWhenCapabilityPayloadIdentityIsMissing(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !setCapabilityOutputForProjectionPressure(&loaded.World, "security.authorize", "missing.projection.payload") {
		t.Fatal("security.authorize Capabilityを見つけられません")
	}

	_, err = deriveBusinessActionDataProjectionProbe(loaded.World, f)
	if err == nil {
		t.Fatal("存在しないCapability payload identityをnon-Dataとして黙って受理しました")
	}
	if !strings.Contains(err.Error(), "payload identityをexactに解決できません") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBusinessActionDataProjectionFailsClosedWhenCapabilityPayloadTypeIdentityIsAmbiguous(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.World.Semantics) == 0 {
		t.Fatal("Semantic namespaceがありません")
	}

	duplicateID := core.CanonicalID("projection.duplicate_type")
	loaded.World.Semantics[0].Types = append(loaded.World.Semantics[0].Types,
		semantic.Type{ID: duplicateID, Symbol: "duplicate type 1", Kind: semantic.TypePrimitive},
		semantic.Type{ID: duplicateID, Symbol: "duplicate type 2", Kind: semantic.TypePrimitive},
	)
	if !setCapabilityOutputForProjectionPressure(&loaded.World, "security.authorize", string(duplicateID)) {
		t.Fatal("security.authorize Capabilityを見つけられません")
	}

	_, err = deriveBusinessActionDataProjectionProbe(loaded.World, f)
	if err == nil {
		t.Fatal("複数のSemantic Typeへ解決するCapability payload identityを黙って受理しました")
	}
	if !strings.Contains(err.Error(), "payload identity ambiguity") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func setCapabilityOutputForProjectionPressure(world *ir.World, capabilityID, output string) bool {
	for namespaceIndex := range world.Semantics {
		for capabilityIndex := range world.Semantics[namespaceIndex].Capabilities {
			capability := &world.Semantics[namespaceIndex].Capabilities[capabilityIndex]
			if string(capability.ID) != capabilityID {
				continue
			}
			capability.Output = core.CanonicalID(output)
			return true
		}
	}
	return false
}
