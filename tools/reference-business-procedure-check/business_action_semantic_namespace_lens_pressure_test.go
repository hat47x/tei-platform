package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/ir"
	"tei/internal/workspace"
)

type businessActionSemanticNamespaceLensRowProbe struct {
	StepID                          string   `json:"step"`
	Namespaces                      []string `json:"namespaces,omitempty"`
	NoNamespaceTraceFromCurrentRefs bool     `json:"noNamespaceTraceFromCurrentRefs"`
}

func TestBusinessActionSemanticNamespaceLensDerivesCurrentComprehensionGroups(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	rows, err := deriveBusinessActionSemanticNamespaceLensProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	byStep := businessActionSemanticNamespaceRowsByStepProbe(rows)

	assertSemanticNamespaceLensProbe(t, byStep["BP-00"], []string{"sales"}, false)
	assertSemanticNamespaceLensProbe(t, byStep["BP-01"], []string{"sales"}, false)
	assertSemanticNamespaceLensProbe(t, byStep["BP-02"], []string{"customer", "sales"}, false)
	assertSemanticNamespaceLensProbe(t, byStep["BP-03"], []string{"sales"}, false)
	assertSemanticNamespaceLensProbe(t, byStep["BP-05"], []string{"sales"}, false)
	assertSemanticNamespaceLensProbe(t, byStep["BP-06"], []string{"security"}, false)
	assertSemanticNamespaceLensProbe(t, byStep["BP-07"], []string{"sales"}, false)

	for _, stepID := range []string{"BP-04", "BP-08", "BP-09"} {
		assertSemanticNamespaceLensProbe(t, byStep[stepID], nil, true)
	}
}

func TestSemanticNamespaceLensDoesNotBecomeSubjectAreaAuthority(t *testing.T) {
	typ := reflect.TypeOf(businessActionSemanticNamespaceLensRowProbe{})
	want := []string{"StepID", "Namespaces", "NoNamespaceTraceFromCurrentRefs"}
	if got := fieldNames(typ); !reflect.DeepEqual(got, want) {
		t.Fatalf("Semantic Namespace lens boundaryが変化しました: got=%v want=%v", got, want)
	}
	for _, forbidden := range []string{"SubjectArea", "BusinessArea", "Subsystem", "Ownership", "Canonical", "Complete"} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("derived Namespace lensへ未証明のauthority field %qを追加しました", forbidden)
		}
	}
}

func TestSemanticNamespaceLensKeepsCrossNamespaceBusinessActionVisible(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	rows, err := deriveBusinessActionSemanticNamespaceLensProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	bp02 := businessActionSemanticNamespaceRowsByStepProbe(rows)["BP-02"]
	if !reflect.DeepEqual(bp02.Namespaces, []string{"customer", "sales"}) {
		t.Fatalf("BP-02のcross-namespace traceを一つのBusiness Areaへ丸めました: %+v", bp02)
	}
}

func deriveBusinessActionSemanticNamespaceLensProbe(world ir.World, f fixture) ([]businessActionSemanticNamespaceLensRowProbe, error) {
	dataRows, err := deriveBusinessActionDataProjectionProbe(world, f)
	if err != nil {
		return nil, err
	}

	rows := make([]businessActionSemanticNamespaceLensRowProbe, 0, len(dataRows))
	for _, dataRow := range dataRows {
		namespaceSet := map[string]struct{}{}
		for _, dataID := range dataRow.Data {
			namespaceID, ok := findSemanticNamespaceForDataExactlyOnceProbe(world, dataID)
			if !ok {
				return nil, fmt.Errorf("Data %sのSemantic Namespaceをexactに解決できません", dataID)
			}
			namespaceSet[namespaceID] = struct{}{}
		}
		namespaces := make([]string, 0, len(namespaceSet))
		for namespaceID := range namespaceSet {
			namespaces = append(namespaces, namespaceID)
		}
		sort.Strings(namespaces)
		rows = append(rows, businessActionSemanticNamespaceLensRowProbe{
			StepID:                          dataRow.StepID,
			Namespaces:                      namespaces,
			NoNamespaceTraceFromCurrentRefs: len(namespaces) == 0 && dataRow.NoDataTraceFromCurrentRefs,
		})
	}
	return rows, nil
}

func findSemanticNamespaceForDataExactlyOnceProbe(world ir.World, dataID string) (string, bool) {
	found := ""
	count := 0
	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			if string(data.ID) == dataID {
				found = string(namespace.ID)
				count++
			}
		}
	}
	return found, count == 1
}

func businessActionSemanticNamespaceRowsByStepProbe(rows []businessActionSemanticNamespaceLensRowProbe) map[string]businessActionSemanticNamespaceLensRowProbe {
	result := make(map[string]businessActionSemanticNamespaceLensRowProbe, len(rows))
	for _, row := range rows {
		result[row.StepID] = row
	}
	return result
}

func assertSemanticNamespaceLensProbe(t *testing.T, row businessActionSemanticNamespaceLensRowProbe, namespaces []string, noTrace bool) {
	t.Helper()
	if !(len(row.Namespaces) == 0 && len(namespaces) == 0) && !reflect.DeepEqual(row.Namespaces, namespaces) {
		t.Fatalf("step %s Namespaces=%v want=%v", row.StepID, row.Namespaces, namespaces)
	}
	if row.NoNamespaceTraceFromCurrentRefs != noTrace {
		t.Fatalf("step %s NoNamespaceTraceFromCurrentRefs=%v want=%v", row.StepID, row.NoNamespaceTraceFromCurrentRefs, noTrace)
	}
}
