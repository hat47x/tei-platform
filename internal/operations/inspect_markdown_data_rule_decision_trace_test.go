package operations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// decisionInspectionSource is an explicitly supplied, state-aware reading.
// It does not claim a Provider or database read; it records the actual
// Attribute ID asked for by the existing evaluator.
type decisionInspectionSource struct {
	reading validationeval.NumberReading
	calls   []core.CanonicalID
}

func (s *decisionInspectionSource) ReadNumber(id core.CanonicalID) (validationeval.NumberReading, error) {
	s.calls = append(s.calls, id)
	return s.reading, nil
}

func (s *decisionInspectionSource) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, fmt.Errorf("unexpected relation read %s/%s", relation, attribute)
}

// The real Markdown Reader and the existing read-only Inspector select the
// exact typed Attribute/Rule and their Source byte ranges from one Load.
// Decide then consumes that selected Rule, distinguishes NULL from an unread
// Attribute, and reports the actually requested Read ID. The fixed expected
// judgments below do not come from the reader or from Decide.
//
// This is a limited Source -> Inspection -> state-aware Rule witness, not a
// Provider invocation, generated Go parity, R2-P Builder, or public grant.
func TestSelectedMarkdownDataRuleInspectionCarriesDecisionReasons(t *testing.T) {
	const sourcePath = "semantic/organization/transfer-request.tei.md"
	const attributeID = "organization.transfer_request.allocation"
	const ruleID = "organization.transfer_request.allocation_positive"

	fixture := filepath.Join("..", "workspace", "testdata", "tei-md-data-table")
	loaded, err := workspace.Load(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics := validation.Validate(loaded.World); len(diagnostics) != 0 {
		t.Fatalf("selected World is invalid: %+v", diagnostics)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Path != sourcePath ||
		loaded.AuthoringSources[0].Language.LanguageID != "tei.authoring" ||
		loaded.AuthoringSources[0].Language.ContractVersion != "v1" {
		t.Fatalf("unexpected selected Reader: %+v", loaded.AuthoringSources)
	}
	index, err := workspace.BuildSourceIndexFromLoad(fixture, loaded)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	if err := index.Validate(func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join(fixture, filepath.FromSlash(path)))
	}); err != nil {
		t.Fatalf("selected Source has stale provenance: %v", err)
	}

	read := func(kind, id string) SubjectInspection {
		t.Helper()
		subject := sourceindex.Subject{Kind: kind, ID: id}
		inspected, err := InspectionFromLoad(loaded, index, subject)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateInspectionProjection(subject, inspected); err != nil {
			t.Fatal(err)
		}
		if inspected.Effect != "read-only" || len(inspected.Sources.References) != 1 ||
			len(inspected.Sources.Artifacts) != 1 ||
			inspected.Sources.References[0].ArtifactPath != sourcePath ||
			inspected.Sources.References[0].Role != sourceindex.RoleDefinition ||
			inspected.Sources.Artifacts[0].Fingerprint != sourceindex.Fingerprint(raw) {
			t.Fatalf("Inspection lost its selected Source: %+v", inspected)
		}
		if err := inspected.Sources.References[0].Range.Validate(raw); err != nil {
			t.Fatal(err)
		}
		return inspected
	}
	attributeInspection := read("semantic-attribute", attributeID)
	ruleInspection := read("semantic-validation", ruleID)
	if attributeInspection.WorldFingerprint == "" ||
		attributeInspection.WorldFingerprint != ruleInspection.WorldFingerprint ||
		attributeInspection.SourceIndexFingerprint == "" ||
		attributeInspection.SourceIndexFingerprint != ruleInspection.SourceIndexFingerprint {
		t.Fatal("Attribute and Rule did not share one selected World and Source Index")
	}
	var attribute semantic.Attribute
	if err := json.Unmarshal(attributeInspection.Canonical, &attribute); err != nil {
		t.Fatal(err)
	}
	var rule semantic.Validation
	if err := json.Unmarshal(ruleInspection.Canonical, &rule); err != nil {
		t.Fatal(err)
	}
	if attribute.ID != attributeID || attribute.TypeRef != "std.decimal" ||
		!attribute.Required || rule.ID != ruleID ||
		rule.Kind != semantic.ValidationPositive || rule.Attribute != attribute.ID ||
		rule.Relation != "" {
		t.Fatalf("selected Data/Rule cannot satisfy the expected typed Read: %+v / %+v", attribute, rule)
	}
	// The Attribute definition is an item table cell, not an example value.
	cell := []byte("allocation | std.decimal |")
	if bytes.Count(raw, cell) != 1 ||
		attributeInspection.Sources.References[0].Range.Start.Offset != bytes.Index(raw, cell) {
		t.Fatal("the selected Attribute has no exact definition byte range")
	}

	tests := []struct {
		name    string
		reading validationeval.NumberReading
		result  validationeval.Result
		reason  validationeval.UndecidedReason
	}{
		{
			name: "present positive",
			reading: validationeval.NumberReading{
				State: validationeval.ValuePresent,
				Value: new(big.Rat).SetFrac64(5, 2),
			},
			result: validationeval.ResultSatisfied,
		},
		{
			name: "present zero",
			reading: validationeval.NumberReading{
				State: validationeval.ValuePresent,
				Value: new(big.Rat).SetInt64(0),
			},
			result: validationeval.ResultNotSatisfied,
		},
		{
			name:    "explicit null",
			reading: validationeval.NumberReading{State: validationeval.ValueNull},
			reason:  validationeval.UndecidedValueNull,
		},
		{
			name:    "not fetched",
			reading: validationeval.NumberReading{State: validationeval.ValueNotFetched},
			reason:  validationeval.UndecidedValueNotFetched,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source := &decisionInspectionSource{reading: tc.reading}
			decision, err := validationeval.Decide(rule, source, validationeval.Operators{})
			if err != nil {
				t.Fatal(err)
			}
			if len(source.calls) != 1 || source.calls[0] != attribute.ID {
				t.Fatalf("actual Rule Read differs from the typed Source: %v", source.calls)
			}
			if decision.Result != tc.result {
				t.Fatalf("result=%q, independent expected %q", decision.Result, tc.result)
			}
			if tc.reason == "" {
				if decision.Undecided != nil {
					t.Fatalf("a decided result carried an undecided reason: %+v", decision.Undecided)
				}
			} else {
				if decision.Result != "" || decision.Undecided == nil ||
					decision.Undecided.Reason != tc.reason ||
					decision.Undecided.Attribute != attribute.ID ||
					decision.Undecided.Relation != "" {
					t.Fatalf("undecided=%+v, expected reason=%s for %s",
						decision, tc.reason, attribute.ID)
				}
			}
		})
	}
}
