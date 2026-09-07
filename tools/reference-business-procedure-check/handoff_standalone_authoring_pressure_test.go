package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"testing"
)

const (
	standaloneHandoffSchemaVersion = "business-handoff-source-v0-pressure"
	standaloneHandoffArtifactRole  = "reference-handoff-authoring-pressure"
)

type standaloneHandoffSourceProbe struct {
	SchemaVersion string                `json:"schemaVersion"`
	ArtifactRole  string                `json:"artifactRole"`
	Canonical     bool                  `json:"canonical"`
	Handoffs      []referenceSetHandoff `json:"handoffs"`
}

func TestStandaloneHandoffSourceNormalizesToSameMeaningAsReferenceSet(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, root)

	want, err := deriveCanonicalBusinessHandoffProbes(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	source := standaloneHandoffSourceProbe{
		SchemaVersion: standaloneHandoffSchemaVersion,
		ArtifactRole:  standaloneHandoffArtifactRole,
		Canonical:     false,
		Handoffs:      append([]referenceSetHandoff(nil), set.Handoffs...),
	}
	got, err := deriveStandaloneHandoffProbes(source, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("standalone Handoff sourceがReference Setと異なるnormalized meaningを生成しました: got=%+v want=%+v", got, want)
	}
}

func TestStandaloneHandoffSourceRoundTripsWithoutProcedureArtifactLocators(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, _ := currentReferenceSetAndFixturesD4O1Probe(t, root)
	source := standaloneHandoffSourceProbe{
		SchemaVersion: standaloneHandoffSchemaVersion,
		ArtifactRole:  standaloneHandoffArtifactRole,
		Canonical:     false,
		Handoffs:      append([]referenceSetHandoff(nil), set.Handoffs...),
	}

	encoded, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte(`"artifact"`)) || bytes.Contains(encoded, []byte(`order-entry-business-procedure.json`)) {
		t.Fatalf("standalone Handoff sourceへProcedure artifact locatorが混入しました: %s", encoded)
	}
	reloaded, err := decodeStandaloneHandoffSourceProbe(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(source, reloaded) {
		t.Fatalf("standalone Handoff source round-tripでmeaningを失いました: before=%+v after=%+v", source, reloaded)
	}
}

func TestStandaloneHandoffSourceFailsClosedOnStaleEndpoint(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, root)
	source := standaloneHandoffSourceProbe{
		SchemaVersion: standaloneHandoffSchemaVersion,
		ArtifactRole:  standaloneHandoffArtifactRole,
		Canonical:     false,
		Handoffs:      append([]referenceSetHandoff(nil), set.Handoffs...),
	}
	if len(source.Handoffs) == 0 {
		t.Fatal("test setup: Handoffがありません")
	}

	stale := source
	stale.Handoffs = append([]referenceSetHandoff(nil), source.Handoffs...)
	stale.Handoffs[0].To.Step = "FU-00-renamed"
	if _, err := deriveStandaloneHandoffProbes(stale, fixtures); err == nil {
		t.Fatal("standalone Handoff sourceがstale Step endpointを黙って受理しました")
	}
}

func TestStandaloneHandoffSourceRejectsUnknownWorkflowFields(t *testing.T) {
	raw := `{
		"schemaVersion":"business-handoff-source-v0-pressure",
		"artifactRole":"reference-handoff-authoring-pressure",
		"canonical":false,
		"handoffs":[{
			"from":{"procedure":"p1","step":"s1"},
			"to":{"procedure":"p2","step":"s2"},
			"condition":"continue",
			"retry":3
		}]
	}`
	if _, err := decodeStandaloneHandoffSourceProbe(bytes.NewBufferString(raw)); err == nil {
		t.Fatal("standalone Handoff sourceへWorkflow retry fieldを黙って受理しました")
	}
}

func decodeStandaloneHandoffSourceProbe(r io.Reader) (standaloneHandoffSourceProbe, error) {
	var source standaloneHandoffSourceProbe
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&source); err != nil {
		return standaloneHandoffSourceProbe{}, err
	}
	if err := requireEOF(decoder); err != nil {
		return standaloneHandoffSourceProbe{}, err
	}
	if source.SchemaVersion != standaloneHandoffSchemaVersion {
		return standaloneHandoffSourceProbe{}, fmt.Errorf("unsupported standalone Handoff schemaVersion %q", source.SchemaVersion)
	}
	if source.ArtifactRole != standaloneHandoffArtifactRole {
		return standaloneHandoffSourceProbe{}, fmt.Errorf("unsupported standalone Handoff artifactRole %q", source.ArtifactRole)
	}
	if source.Canonical {
		return standaloneHandoffSourceProbe{}, fmt.Errorf("standalone Handoff pressure source must not claim Canonical authority")
	}
	return source, nil
}

func deriveStandaloneHandoffProbes(source standaloneHandoffSourceProbe, fixtures map[string]fixture) ([]canonicalBusinessHandoffProbe, error) {
	set := referenceSet{Handoffs: append([]referenceSetHandoff(nil), source.Handoffs...)}
	return deriveCanonicalBusinessHandoffProbes(set, fixtures)
}
