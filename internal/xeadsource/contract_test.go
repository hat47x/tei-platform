package xeadsource

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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
		"subsystem":          "Subsystem",
		"table":              "Table",
		"tableField":         "TableField",
		"relationship":       "Relationship",
		"function":           "Function",
		"task":               "Task",
		"taskAction":         "TaskAction",
		"taskFunctionIO":     "TaskFunctionIO",
		"ioTable":            "IOTable",
		"ioPanel":            "IOPanel",
		"ioSpool":            "IOSpool",
		"functionUsedByThis": "FunctionUsedByThis",
		"instance":           "Instance",
		"subjectArea":        "SubjectArea",
		"dataflowNode":       "DataflowNode",
		"dataflowLine":       "DataflowLine",
		"role":               "Role",
		"department":         "Department",
		"dataType":           "DataType",
		"functionType":       "FunctionType",
	}
	xeafNames := map[string]string{
		"subsystem": "Subsystem",
		"table":     "Table",
		"function":  "Function",
		"script":    "Script",
		"field":     "Field",
		"column":    "Column",
		"button":    "Button",
		"filter":    "Filter",
		"refer":     "Refer",
		"detail":    "Detail",
		"key":       "Key",
		"menu":      "Menu",
	}

	assertInventoryCoverage(t, FormatXead, inventory.Xead.Counts, xeadNames, contract.ConstructMappings["xead"])
	assertInventoryCoverage(t, FormatXeaf, inventory.Xeaf.Counts, xeafNames, contract.ConstructMappings["xeaf"])
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
