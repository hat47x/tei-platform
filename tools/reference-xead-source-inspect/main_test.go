package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/xeadsource"
)

func TestBuildReportOmitsRawBusinessIdentifiersAndDetectsDerivedRelationshipKey(t *testing.T) {
	contract := []byte(`{
	  "constructMappings": {
	    "xead": {
	      "System": {"classification":["source-metadata"],"extension":"reference.compat.xead.source"},
	      "Table": {"classification":["canonical-candidate"],"extension":"reference.xead.relational-semantics"},
	      "TableField": {"classification":["canonical-candidate"],"extension":"reference.xead.relational-semantics"},
	      "TableKey": {
	        "classification":["canonical-candidate","unresolved"],
	        "extension":"reference.xead.relational-semantics",
	        "authorityLimits":["legacy-key-is-not-business-identity"]
	      },
	      "TableKeyField": {"classification":["canonical-candidate"],"extension":"reference.xead.relational-semantics"},
	      "Relationship": {"classification":["canonical-candidate","unresolved"],"extension":"reference.xead.relational-semantics"}
	    }
	  }
	}`)
	source := []byte(`<System><Table ID="SECRET_TABLE"><TableField ID="SECRET_DERIVED" AttributeType="DERIVABLE"/><TableKey ID="SECRET_KEY" Type="PK"><TableKeyField FieldID="SECRET_DERIVED"/></TableKey></Table><Table ID="OTHER_TABLE"><TableField ID="OTHER_FIELD" AttributeType="NATIVE"/><TableKey ID="OTHER_KEY" Type="PK"><TableKeyField FieldID="OTHER_FIELD"/></TableKey></Table><Relationship ID="SECRET_REL" Table1ID="SECRET_TABLE" TableKey1ID="SECRET_KEY" Table2ID="OTHER_TABLE" TableKey2ID="OTHER_KEY"/><PRIVATE_EXTENSION/></System>`)
	inspection, err := xeadsource.InspectWithContract(xeadsource.FormatXead, source, contract)
	if err != nil {
		t.Fatal(err)
	}
	got := buildReport(inspection)
	if got.SchemaVersion != "reference-xead-source-inspection-v2" {
		t.Fatalf("schema version=%q", got.SchemaVersion)
	}
	if got.RelationalFacts.DerivedLegacyKeyMemberCount != 1 ||
		got.RelationalFacts.RelationshipsUsingDerivedKeyCount != 1 {
		t.Fatalf("derived relation pressure was not detected: %#v", got.RelationalFacts)
	}
	if got.RelationalResolutionStatus != "resolved" {
		t.Fatalf("valid relational source was marked unresolved: %#v", got)
	}
	if got.RelationalFactResolutions.Total != 3 ||
		got.RelationalFactResolutions.Resolved != 3 ||
		got.RelationalFactResolutions.Unresolved != 0 ||
		got.RelationalFactResolutions.Unvalidated != 0 {
		t.Fatalf("valid relational facts were not reported as locally resolved: %#v", got.RelationalFactResolutions)
	}
	if got.UnknownConstructCount != 1 {
		t.Fatalf("unknown source construct count = %d", got.UnknownConstructCount)
	}
	if got.Source.SHA256 == "" ||
		got.ClassificationPolicyFingerprint == "" ||
		got.TranscriptionContractFingerprint == "" {
		t.Fatalf("provenance fingerprints are incomplete: %#v", got)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"SECRET_TABLE", "SECRET_DERIVED", "SECRET_KEY", "SECRET_REL", "OTHER_TABLE", "OTHER_FIELD", "OTHER_KEY", "PRIVATE_EXTENSION"} {
		if bytes.Contains(encoded, []byte(secret)) {
			t.Fatalf("public report leaked raw source identifier %q: %s", secret, encoded)
		}
	}
}

func TestRunEmitsAggregateInspectionWithoutRawSource(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "fixture.xeaf")
	contractPath := filepath.Join(dir, "contract.json")
	source := []byte(`<System><Table ID="PRIVATE_TABLE"><Field ID="PRIVATE_FIELD"/><Key Type="PK" Fields="PRIVATE_FIELD;"/></Table></System>`)
	contract := []byte(`{
	  "constructMappings": {
	    "xeaf": {
	      "System": {"classification":["realization"],"extension":"reference.xead.platform-service-binding"},
	      "Table": {"classification":["realization"],"extension":"reference.xead.relational-semantics"},
	      "Field": {"classification":["realization"],"extension":"reference.xead.presentation"},
	      "Key": {
	        "classification":["realization","unresolved"],
	        "extension":"reference.xead.relational-semantics",
	        "authorityLimits":["runtime-key-is-realization-not-business-identity"]
	      }
	    }
	  }
	}`)
	if err := os.WriteFile(sourcePath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contractPath, contract, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"-format", "xeaf",
		"-source", sourcePath,
		"-contract", contractPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run code=%d stderr=%s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "PRIVATE_TABLE") ||
		strings.Contains(stdout.String(), "PRIVATE_FIELD") {
		t.Fatalf("aggregate output leaked source identifiers: %s", stdout.String())
	}
	var got report
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != "reference-xead-source-inspection-v2" {
		t.Fatalf("schema version=%q", got.SchemaVersion)
	}
	if got.RelationalFacts.RuntimeKeyCount != 1 ||
		got.Source.Format != xeadsource.FormatXeaf ||
		got.Source.SHA256 == "" ||
		got.RelationalResolutionStatus != "resolved" ||
		got.RelationalFactResolutions.Total != 1 ||
		got.RelationalFactResolutions.Resolved != 1 {
		t.Fatalf("unexpected report: %#v", got)
	}
}

func TestRunRejectsUnsupportedFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-format", "other", "-source", "unused"}, &stdout, &stderr); code != 2 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
}


func TestBuildReportMarksRelationalDiagnosticsAsUnresolved(t *testing.T) {
	inspection, err := xeadsource.Inspect(
		xeadsource.FormatXead,
		[]byte(`<System><Table ID="T1"><TableKey ID="K1" Type="PK"><TableKeyField FieldID="MISSING"/></TableKey></Table></System>`),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	got := buildReport(inspection)
	if got.RelationalResolutionStatus != "unresolved" {
		t.Fatalf("relational diagnostic did not block lowering readiness: %#v", got)
	}
	if got.RelationalDiagnostics.Total == 0 {
		t.Fatalf("unresolved report lost diagnostic Evidence: %#v", got)
	}
	if got.RelationalFactResolutions.Total != 1 ||
		got.RelationalFactResolutions.Unresolved != 1 ||
		got.RelationalFactResolutions.Resolved != 0 {
		t.Fatalf("unresolved fact readiness was not summarized: %#v", got.RelationalFactResolutions)
	}
}


func TestBuildReportPreservesLocalReadinessWithoutPublishingLocators(t *testing.T) {
	inspection, err := xeadsource.Inspect(
		xeadsource.FormatXead,
		[]byte(`<System><Table ID="SECRET_TABLE"><TableField ID="SECRET_A"/><TableKey ID="SECRET_GOOD" Type="PK"><TableKeyField FieldID="SECRET_A"/></TableKey><TableKey ID="SECRET_BAD" Type="SK"><TableKeyField FieldID="SECRET_MISSING"/></TableKey></Table></System>`),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	got := buildReport(inspection)
	if got.RelationalResolutionStatus != "unresolved" ||
		got.RelationalFactResolutions.Total != 2 ||
		got.RelationalFactResolutions.Resolved != 1 ||
		got.RelationalFactResolutions.Unresolved != 1 ||
		got.RelationalFactResolutions.Unvalidated != 0 {
		t.Fatalf("local readiness summary drifted: %#v", got.RelationalFactResolutions)
	}
	legacyKeys := got.RelationalFactResolutions.ByKind["legacy-key"]
	if legacyKeys.Total != 2 || legacyKeys.Resolved != 1 || legacyKeys.Unresolved != 1 {
		t.Fatalf("legacy-key readiness summary drifted: %#v", legacyKeys)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"SECRET_TABLE", "SECRET_A", "SECRET_GOOD", "SECRET_BAD", "SECRET_MISSING", "/System[1]/Table[1]"} {
		if bytes.Contains(encoded, []byte(secret)) {
			t.Fatalf("public local readiness leaked source locator or identifier %q: %s", secret, encoded)
		}
	}
}

func TestBuildReportMarksUnvalidatedDeclaredKeyFieldsAsPartial(t *testing.T) {
	inspection, err := xeadsource.Inspect(
		xeadsource.FormatXeaf,
		[]byte(`<System><Function ID="PRIVATE_FUNCTION" KeyFields="A;B;"/></System>`),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	got := buildReport(inspection)
	if got.RelationalResolutionStatus != "partial" ||
		got.RelationalFactResolutions.Total != 1 ||
		got.RelationalFactResolutions.Unvalidated != 1 ||
		got.RelationalFactResolutions.ByKind["declared-key-fields"].Unvalidated != 1 {
		t.Fatalf("unvalidated declared key fields were reported as lowerable: %#v", got)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("PRIVATE_FUNCTION")) {
		t.Fatalf("public partial readiness leaked function identifier: %s", encoded)
	}
}
