package main

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/xeadsource"
)

func commandContractFixture() []byte {
	return []byte(`{
  "constructMappings": {
    "xead": {
      "System": {
        "classification": ["source-metadata"],
        "extension": "reference.compat.xead.source"
      }
    }
  },
  "sourceProfiles": {
    "xead": {
      "rootKind": "System",
      "supportedFormatVersions": ["1.2"]
    }
  }
}`)
}

func TestCommandOutputDefaultsToPublicAccounting(t *testing.T) {
	source := []byte(`<System FormatVersion="1.2">
  <Table ID="private-table"/>
  <Function ID="private-function"><IOTable ID="private-io" TableID="private-table" OpR="+"/></Function>
  <Task ID="private-task"><TaskAction ID=""><TaskFunctionIO FunctionID="private-function" IOID="0"/></TaskAction></Task>
</System>`)
	output, err := buildCommandOutput(xeadsource.FormatXead, source, commandContractFixture(), false)
	if err != nil {
		t.Fatal(err)
	}
	if output.LocalProjection != nil || output.LocalOutputNotice != "" {
		t.Fatalf("default command leaked local projection: %+v", output)
	}
	encoded, err := json.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private-table", "private-function", "private-task", "private-io", "fromToken", "sourceTokens"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("default public output leaked %q: %s", forbidden, encoded)
		}
	}
	if output.Accounting.TaskActionMissingIDCount != 1 ||
		output.Accounting.PotentialActionFunctionTablePathCount != 1 {
		t.Fatalf("unexpected accounting: %+v", output.Accounting)
	}
}

func TestCommandOutputRequiresExplicitLocalProjection(t *testing.T) {
	source := []byte(`<System FormatVersion="1.2"><Task ID="T1"><TaskAction ID=""/></Task></System>`)
	output, err := buildCommandOutput(xeadsource.FormatXead, source, commandContractFixture(), true)
	if err != nil {
		t.Fatal(err)
	}
	if output.LocalProjection == nil || output.LocalOutputNotice == "" {
		t.Fatalf("explicit local output missing: %+v", output)
	}
}
