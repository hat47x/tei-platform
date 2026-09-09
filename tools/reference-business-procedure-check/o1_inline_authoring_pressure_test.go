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

const inlineO1FixtureSchemaVersion = "business-procedure-reference-v0-inline-o1-pressure"

type inlineO1FixtureProbe struct {
	SchemaVersion string              `json:"schemaVersion"`
	ArtifactRole  string              `json:"artifactRole"`
	Canonical     bool                `json:"canonical"`
	Application   string              `json:"application,omitempty"`
	Procedure     procedure           `json:"procedure"`
	Steps         []inlineO1StepProbe `json:"steps"`
}

type inlineO1StepProbe struct {
	ID                          string   `json:"id"`
	BusinessAction              string   `json:"businessAction"`
	Realization                 string   `json:"realization"`
	ApplicationPatterns         []string `json:"applicationPatterns,omitempty"`
	Data                        []string `json:"data,omitempty"`
	Relations                   []string `json:"relations,omitempty"`
	Validations                 []string `json:"validations,omitempty"`
	Capabilities                []string `json:"capabilities,omitempty"`
	ProtectedApplicationPattern string   `json:"protectedApplicationPattern,omitempty"`
}

func TestInlineO1AuthoringCandidateRoundTripsSingularTargetLosslessly(t *testing.T) {
	base, err := readFixture(filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	candidate := inlineO1CandidateFromFixtureProbe(base, map[string]string{"BP-06": "order_entry.submit_order"})

	encoded, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := decodeInlineO1FixtureProbe(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(candidate, reloaded) {
		t.Fatalf("inline O1 round-tripでmeaningを失いました: before=%+v after=%+v", candidate, reloaded)
	}

	bp06, ok := inlineO1StepByIDProbe(reloaded, "BP-06")
	if !ok || bp06.ProtectedApplicationPattern != "order_entry.submit_order" {
		t.Fatalf("BP-06 singular protected targetを保持できません: %+v", bp06)
	}
	if len(bp06.ApplicationPatterns) != 0 || !reflect.DeepEqual(bp06.Capabilities, []string{"security.authorize"}) {
		t.Fatalf("inline O1がtarget / realization / capability separationを壊しました: %+v", bp06)
	}
}

func TestInlineO1AuthoringCandidateNormalizesToSameMeaningAsCurrentTargetPressure(t *testing.T) {
	base, err := readFixture(filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	candidate := inlineO1CandidateFromFixtureProbe(base, map[string]string{"BP-06": "order_entry.submit_order"})
	inlineFixture, inlineTargets, err := inlineO1CandidateToFixtureProbe(candidate)
	if err != nil {
		t.Fatal(err)
	}
	inlineMeaning, err := deriveD4O1ProcedureProbe(inlineFixture, inlineTargets)
	if err != nil {
		t.Fatal(err)
	}

	currentMeaning, err := deriveD4O1ProcedureProbe(base, map[string][]string{"BP-06": {"order_entry.submit_order"}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(inlineMeaning, currentMeaning) {
		t.Fatalf("inline narrow O1とcurrent target pressureが異なるnormalized meaningを生成しました: inline=%+v current=%+v", inlineMeaning, currentMeaning)
	}
}

func TestInlineO1AuthoringCandidateAllowsNoTargetWithoutInventingRoleMeaning(t *testing.T) {
	base, err := readFixture(filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	candidate := inlineO1CandidateFromFixtureProbe(base, nil)
	fixture, targets, err := inlineO1CandidateToFixtureProbe(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 0 {
		t.Fatalf("targetなしsourceからprotected targetを発明しました: %v", targets)
	}
	if _, err := deriveD4O1ProcedureProbe(fixture, targets); err != nil {
		t.Fatalf("protected targetなしのProcedureを受理できません: %v", err)
	}
}

func TestInlineO1AuthoringCandidateRejectsMultiTargetAndGenericRoleFields(t *testing.T) {
	multiTarget := `{
		"schemaVersion":"business-procedure-reference-v0-inline-o1-pressure",
		"artifactRole":"reference-authoring-fixture",
		"canonical":false,
		"procedure":{"id":"p","identityStatus":"reference-local-stable","purpose":"p","trigger":{"status":"candidate","text":"t"},"actor":{"status":"candidate","label":"a"}},
		"steps":[{"id":"S","businessAction":"x","realization":"manual","protectedApplicationPattern":["a","b"]}]
	}`
	if _, err := decodeInlineO1FixtureProbe(bytes.NewBufferString(multiTarget)); err == nil {
		t.Fatal("current Evidenceなしにmulti-target O1を黙って受理しました")
	}

	genericRole := `{
		"schemaVersion":"business-procedure-reference-v0-inline-o1-pressure",
		"artifactRole":"reference-authoring-fixture",
		"canonical":false,
		"procedure":{"id":"p","identityStatus":"reference-local-stable","purpose":"p","trigger":{"status":"candidate","text":"t"},"actor":{"status":"candidate","label":"a"}},
		"steps":[{"id":"S","businessAction":"x","realization":"manual","protectedApplicationPattern":"a","role":"protects"}]
	}`
	if _, err := decodeInlineO1FixtureProbe(bytes.NewBufferString(genericRole)); err == nil {
		t.Fatal("generic role taxonomyをinline O1 candidateへ黙って受理しました")
	}
}

func decodeInlineO1FixtureProbe(r io.Reader) (inlineO1FixtureProbe, error) {
	var value inlineO1FixtureProbe
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return inlineO1FixtureProbe{}, err
	}
	if err := requireEOF(decoder); err != nil {
		return inlineO1FixtureProbe{}, err
	}
	if value.SchemaVersion != inlineO1FixtureSchemaVersion {
		return inlineO1FixtureProbe{}, fmt.Errorf("unsupported inline O1 schemaVersion %q", value.SchemaVersion)
	}
	if value.ArtifactRole != fixtureArtifactRole {
		return inlineO1FixtureProbe{}, fmt.Errorf("unsupported inline O1 artifactRole %q", value.ArtifactRole)
	}
	if value.Canonical {
		return inlineO1FixtureProbe{}, fmt.Errorf("inline O1 pressure fixture must not claim Canonical authority")
	}
	return value, nil
}

func inlineO1CandidateFromFixtureProbe(base fixture, targets map[string]string) inlineO1FixtureProbe {
	result := inlineO1FixtureProbe{
		SchemaVersion: inlineO1FixtureSchemaVersion,
		ArtifactRole:  base.ArtifactRole,
		Canonical:     base.Canonical,
		Application:   base.Application,
		Procedure:     base.Procedure,
		Steps:         make([]inlineO1StepProbe, 0, len(base.Steps)),
	}
	for _, item := range base.Steps {
		result.Steps = append(result.Steps, inlineO1StepProbe{
			ID:                          item.ID,
			BusinessAction:              item.BusinessAction,
			Realization:                 item.Realization,
			ApplicationPatterns:         append([]string(nil), item.ApplicationPatterns...),
			Data:                        append([]string(nil), item.Data...),
			Relations:                   append([]string(nil), item.Relations...),
			Validations:                 append([]string(nil), item.Validations...),
			Capabilities:                append([]string(nil), item.Capabilities...),
			ProtectedApplicationPattern: targets[item.ID],
		})
	}
	return result
}

func inlineO1CandidateToFixtureProbe(value inlineO1FixtureProbe) (fixture, map[string][]string, error) {
	base := fixture{
		SchemaVersion: fixtureSchemaVersion,
		ArtifactRole:  value.ArtifactRole,
		Canonical:     value.Canonical,
		Application:   value.Application,
		Procedure:     value.Procedure,
		Steps:         make([]step, 0, len(value.Steps)),
	}
	targets := map[string][]string{}
	seen := map[string]struct{}{}
	for _, item := range value.Steps {
		if item.ID == "" {
			return fixture{}, nil, fmt.Errorf("inline O1 Step address is required")
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return fixture{}, nil, fmt.Errorf("duplicate inline O1 Step %q", item.ID)
		}
		seen[item.ID] = struct{}{}
		base.Steps = append(base.Steps, step{
			ID:                  item.ID,
			BusinessAction:      item.BusinessAction,
			Realization:         item.Realization,
			ApplicationPatterns: append([]string(nil), item.ApplicationPatterns...),
			Data:                append([]string(nil), item.Data...),
			Relations:           append([]string(nil), item.Relations...),
			Validations:         append([]string(nil), item.Validations...),
			Capabilities:        append([]string(nil), item.Capabilities...),
		})
		if item.ProtectedApplicationPattern != "" {
			targets[item.ID] = []string{item.ProtectedApplicationPattern}
		}
	}
	return base, targets, nil
}

func inlineO1StepByIDProbe(value inlineO1FixtureProbe, id string) (inlineO1StepProbe, bool) {
	for _, item := range value.Steps {
		if item.ID == id {
			return item, true
		}
	}
	return inlineO1StepProbe{}, false
}
