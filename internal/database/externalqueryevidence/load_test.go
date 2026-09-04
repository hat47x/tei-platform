package externalqueryevidence

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLoadAcceptsKnownEvidenceFields(t *testing.T) {
	realizationDef := testRealization()
	encoded, err := json.Marshal(testEvidence(realizationDef))
	if err != nil {
		t.Fatalf("test EvidenceをJSON化できません: %v", err)
	}

	loaded, err := Load(encoded)
	if err != nil {
		t.Fatalf("既知fieldだけのEvidence JSONを読み込めません: %v", err)
	}
	if loaded.Target != realizationDef.Target || loaded.NativeSource.Fingerprint == "" {
		t.Fatalf("Evidence JSONのidentityまたはfingerprintが失われています: %#v", loaded)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	_, err := Load([]byte(`{"target":{},"unexpected":true}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("未知fieldを黙って受け入れています: %v", err)
	}
}

func TestLoadRejectsTrailingJSONValue(t *testing.T) {
	_, err := Load([]byte(`{} {}`))
	if err == nil || !strings.Contains(err.Error(), "複数の値") {
		t.Fatalf("後続JSON値を黙って無視しています: %v", err)
	}
}
