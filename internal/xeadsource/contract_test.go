package xeadsource

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

type referenceSourceInventory struct {
	Xead struct {
		Counts map[string]int `json:"counts"`
	} `json:"xead"`
	Xeaf struct {
		Counts map[string]int `json:"counts"`
	} `json:"xeaf"`
}

type referenceTranscriptionContract struct {
	ConstructMappings map[string]map[string]json.RawMessage `json:"constructMappings"`
}

func TestObservedReferenceInventoryConstructsAreKnownAndClassified(t *testing.T) {
	root := repositoryRootFromSource(t)
	inventoryBytes, err := os.ReadFile(filepath.Join(root, "reference", "reference-model-transfer", "source-inventory.json"))
	if err != nil {
		t.Fatal(err)
	}
	contractBytes, err := os.ReadFile(filepath.Join(root, "reference", "reference-model-transfer", "transcription-contract-v0.json"))
	if err != nil {
		t.Fatal(err)
	}

	var inventory referenceSourceInventory
	if err := json.Unmarshal(inventoryBytes, &inventory); err != nil {
		t.Fatal(err)
	}
	var contract referenceTranscriptionContract
	if err := json.Unmarshal(contractBytes, &contract); err != nil {
		t.Fatal(err)
	}

	xeadNames := map[string]string{
		"subsystem":             "Subsystem",
		"table":                 "Table",
		"tableField":            "TableField",
		"relationship":          "Relationship",
		"function":              "Function",
		"task":                  "Task",
		"taskAction":            "TaskAction",
		"taskFunctionIO":        "TaskFunctionIO",
		"ioTable":               "IOTable",
		"ioPanel":               "IOPanel",
		"ioSpool":               "IOSpool",
		"functionUsedByThis":    "FunctionUsedByThis",
		"instance":              "Instance",
		"subjectArea":           "SubjectArea",
		"dataflowNode":          "DataflowNode",
		"dataflowLine":          "DataflowLine",
		"role":                  "Role",
		"department":            "Department",
		"dataType":              "DataType",
		"functionType":          "FunctionType",
		"tableKey":              "TableKey",
		"tableKeyField":         "TableKeyField",
		"tableType":             "TableType",
		"subsystemTable":        "SubsystemTable",
		"subsystemRelationship": "SubsystemRelationship",
		"ioPanelField":          "IOPanelField",
		"ioPanelStyle":          "IOPanelStyle",
		"ioSpoolStyle":          "IOSpoolStyle",
		"maintenanceLog":        "MaintenanceLog",
	}
	xeafNames := map[string]string{
		"subsystem":        "Subsystem",
		"table":            "Table",
		"function":         "Function",
		"script":           "Script",
		"field":            "Field",
		"column":           "Column",
		"button":           "Button",
		"filter":           "Filter",
		"refer":            "Refer",
		"detail":           "Detail",
		"key":              "Key",
		"menu":             "Menu",
		"headerPhrase":     "HeaderPhrase",
		"phrase":           "Phrase",
		"option":           "Option",
		"printFont":        "PrintFont",
		"batchField":       "BatchField",
		"addRowListColumn": "AddRowListColumn",
		"addRowListButton": "AddRowListButton",
		"maintenanceLog":   "MaintenanceLog",
	}

	assertInventoryCoverage(t, FormatXead, inventory.Xead.Counts, xeadNames, contract.ConstructMappings["xead"])
	assertInventoryCoverage(t, FormatXeaf, inventory.Xeaf.Counts, xeafNames, contract.ConstructMappings["xeaf"])

	for _, format := range []Format{FormatXead, FormatXeaf} {
		policy, err := ClassificationPolicyFromContract(format, contractBytes)
		if err != nil {
			t.Fatal(err)
		}
		if len(policy) != len(contract.ConstructMappings[string(format)]) {
			t.Fatalf("%s policy count = %d, contract mappings = %d", format, len(policy), len(contract.ConstructMappings[string(format)]))
		}
	}
}

func TestClassificationPolicyFromContractCarriesMachineReadableAuthorityLimits(t *testing.T) {
	root := repositoryRootFromSource(t)
	contractBytes, err := os.ReadFile(filepath.Join(root, "reference", "reference-model-transfer", "transcription-contract-v0.json"))
	if err != nil {
		t.Fatal(err)
	}
	policy, err := ClassificationPolicyFromContract(FormatXead, contractBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(policy["TableKey"].AuthorityLimits, []string{
		"legacy-key-is-not-business-identity",
		"legacy-key-is-not-candidate-key",
		"legacy-key-is-not-functional-dependency",
		"legacy-key-is-not-physical-key-authority",
	}) {
		t.Fatalf("TableKey authority limits drifted: %#v", policy["TableKey"])
	}
}

func TestInspectWithContractBindsExactContractAndNormalizedPolicy(t *testing.T) {
	root := repositoryRootFromSource(t)
	contractBytes, err := os.ReadFile(filepath.Join(root, "reference", "reference-model-transfer", "transcription-contract-v0.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := InspectWithContract(
		FormatXead,
		[]byte(`<System FormatVersion="1.2"><Table><TableKey Type="PK"><TableKeyField FieldID="F1"/></TableKey></Table></System>`),
		contractBytes,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.TranscriptionContractFingerprint == "" {
		t.Fatal("inspection has no exact contract fingerprint")
	}
	if got.ClassificationPolicyFingerprint == "" {
		t.Fatal("inspection has no normalized policy fingerprint")
	}
	var keyCandidate *Candidate
	for i := range got.Candidates {
		if got.Candidates[i].SourceName == "TableKey" {
			keyCandidate = &got.Candidates[i]
			break
		}
	}
	if keyCandidate == nil {
		t.Fatal("TableKey candidate is missing")
	}
	if !reflect.DeepEqual(keyCandidate.AuthorityLimits, []string{
		"legacy-key-is-not-business-identity",
		"legacy-key-is-not-candidate-key",
		"legacy-key-is-not-functional-dependency",
		"legacy-key-is-not-physical-key-authority",
	}) {
		t.Fatalf("contract authority limits did not reach candidate: %#v", keyCandidate)
	}
}

func assertInventoryCoverage(
	t *testing.T,
	format Format,
	counts map[string]int,
	names map[string]string,
	contract map[string]json.RawMessage,
) {
	t.Helper()
	for inventoryName, count := range counts {
		if count <= 0 {
			continue
		}
		sourceName, ok := names[inventoryName]
		if !ok {
			t.Fatalf("%s observed inventory %q has no XML construct mapping", format, inventoryName)
		}
		if _, ok := known[format][sourceName]; !ok {
			t.Fatalf("%s observed construct %q is not in reader known set", format, sourceName)
		}
		if _, ok := contract[sourceName]; !ok {
			t.Fatalf("%s observed construct %q has no transcription classification", format, sourceName)
		}
	}
}

func repositoryRootFromSource(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}


func TestInspectWithContractRejectsUnsupportedSourceProfile(t *testing.T) {
	root := repositoryRootFromSource(t)
	contractBytes, err := os.ReadFile(filepath.Join(root, "reference", "reference-model-transfer", "transcription-contract-v0.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{
		`<System FormatVersion="9.9"><Table/></System>`,
		`<System><Table/></System>`,
		`<OtherRoot FormatVersion="1.2"><Table/></OtherRoot>`,
	} {
		if _, err := InspectWithContract(FormatXead, []byte(source), contractBytes); err == nil {
			t.Fatalf("unsupported source profile was classified: %s", source)
		}
	}
}

func TestInspectWithContractRequiresSourceProfileDeclaration(t *testing.T) {
	contractBytes := []byte(`{
		"constructMappings": {
			"xead": {
				"System": {"classification": ["source-metadata"]}
			}
		}
	}`)
	_, err := InspectWithContract(
		FormatXead,
		[]byte(`<System FormatVersion="1.2"/>`),
		contractBytes,
	)
	if err == nil || !strings.Contains(err.Error(), "no source profile") {
		t.Fatalf("classification without source profile was accepted: %v", err)
	}
}
