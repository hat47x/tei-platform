package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

type businessActionDataProjectionRowProbe struct {
	StepID                     string   `json:"step"`
	Data                       []string `json:"data,omitempty"`
	Relations                  []string `json:"relations,omitempty"`
	EvidencePaths              []string `json:"evidencePaths,omitempty"`
	NoDataTraceFromCurrentRefs bool     `json:"noDataTraceFromCurrentRefs"`
}

func TestBusinessActionDataProjectionDerivesUsefulRowsFromExactAcceptedRefs(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	rows, err := deriveBusinessActionDataProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	byStep := businessActionDataRowsByStep(rows)

	assertBusinessActionDataProjection(t, byStep["BP-00"], []string{"sales.order"}, nil, false)
	assertBusinessActionDataProjection(t, byStep["BP-01"], []string{"sales.order"}, nil, false)
	assertBusinessActionDataProjection(t, byStep["BP-02"], []string{"customer.customer", "sales.order"}, []string{"sales.order.customer"}, false)
	assertBusinessActionDataProjection(t, byStep["BP-03"], []string{"sales.order", "sales.order_line"}, []string{"sales.order.lines"}, false)
	assertBusinessActionDataProjection(t, byStep["BP-05"], []string{"sales.order", "sales.order_line"}, []string{"sales.order.lines"}, false)
	assertBusinessActionDataProjection(t, byStep["BP-06"], []string{"security.authenticated_subject", "security.authorization_denial", "security.authorization_grant"}, nil, false)
	assertBusinessActionDataProjection(t, byStep["BP-07"], []string{"sales.order"}, nil, false)
}

func TestBusinessActionDataProjectionKeepsNoTraceStepsUnresolvedInsteadOfClaimingNoData(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := deriveBusinessActionDataProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	byStep := businessActionDataRowsByStep(rows)

	for _, stepID := range []string{"BP-04", "BP-08", "BP-09"} {
		row := byStep[stepID]
		if !row.NoDataTraceFromCurrentRefs {
			t.Fatalf("%sにcurrent refsからData traceがないことを保持できません: %+v", stepID, row)
		}
		if len(row.Data) != 0 || len(row.Relations) != 0 || len(row.EvidencePaths) != 0 {
			t.Fatalf("%sへ名称・順序等からData/Relationを推測しました: %+v", stepID, row)
		}
	}
}

func TestBusinessActionDataProjectionDoesNotInferSubmitTargetDataForAuthorizationStep(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := deriveBusinessActionDataProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	row := businessActionDataRowsByStep(rows)["BP-06"]

	// O1 protected-target relation is still test-only and not persisted in the
	// authoritative Business source. The Data projection must therefore not
	// derive sales.order from adjacent BP-07 or from wording similarity.
	for _, dataID := range row.Data {
		if dataID == "sales.order" {
			t.Fatalf("BP-06へauthoritative refのないprotected-target Dataを推測しました: %+v", row)
		}
	}
	if row.NoDataTraceFromCurrentRefs {
		t.Fatalf("BP-06のsecurity.authorize Data contractまで失いました: %+v", row)
	}
}

func TestBusinessActionDataProjectionIsAProjectionNotASecondAuthority(t *testing.T) {
	projectionType := reflect.TypeOf(businessActionDataProjectionRowProbe{})
	want := []string{"StepID", "Data", "Relations", "EvidencePaths", "NoDataTraceFromCurrentRefs"}
	if got := fieldNames(projectionType); !reflect.DeepEqual(got, want) {
		t.Fatalf("Business Action x Data projectionが別authority modelへ膨張しました: got=%v want=%v", got, want)
	}
	for _, forbidden := range []string{"Access", "Ownership", "Datasource", "WritePolicy", "Canonical", "Authoritative", "Complete", "BusinessFlow", "Workflow"} {
		if _, found := projectionType.FieldByName(forbidden); found {
			t.Fatalf("derived Data projectionへauthority/completeness field %qを追加しました", forbidden)
		}
	}
}

func TestBusinessActionDataProjectionFailsClosedWhenCapabilityPayloadKindIsAmbiguous(t *testing.T) {
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

	// Capability payload IDs may point either to Semantic Data or Semantic Type.
	// If the same CanonicalID exists in both kinds, the Data projection must not
	// silently choose Data and claim an exact authority trace.
	loaded.World.Semantics[0].Types = append(loaded.World.Semantics[0].Types, semantic.Type{
		ID:     core.CanonicalID("security.authenticated_subject"),
		Symbol: "ambiguous authenticated subject",
		Kind:   semantic.TypePrimitive,
	})

	_, err = deriveBusinessActionDataProjectionProbe(loaded.World, f)
	if err == nil {
		t.Fatal("Capability payloadのData/Type kind ambiguityをDataへ暗黙解決しました")
	}
	if !strings.Contains(err.Error(), "Data/Type kind ambiguity") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func deriveBusinessActionDataProjectionProbe(world ir.World, f fixture) ([]businessActionDataProjectionRowProbe, error) {
	rows := make([]businessActionDataProjectionRowProbe, 0, len(f.Steps))
	for _, step := range f.Steps {
		dataSet := map[string]struct{}{}
		relationSet := map[string]struct{}{}
		evidenceSet := map[string]struct{}{}

		for _, patternID := range step.ApplicationPatterns {
			_, pattern, ok := findPatternOwner(world, patternID)
			if !ok {
				return nil, fmt.Errorf("step %s Application Pattern %sをexactに解決できません", step.ID, patternID)
			}
			prefix := "application-pattern:" + patternID
			if pattern.Data != "" {
				if !semanticDataExistsExactlyOnce(world, string(pattern.Data)) {
					return nil, fmt.Errorf("Pattern %s Data %sをexactに解決できません", patternID, pattern.Data)
				}
				addProjectionEvidence(dataSet, evidenceSet, string(pattern.Data), prefix+"/data:"+string(pattern.Data))
			}
			if pattern.Relation != "" {
				if err := addRelationProjectionEvidence(world, relationSet, dataSet, evidenceSet, string(pattern.Relation), prefix); err != nil {
					return nil, err
				}
			}
			if pattern.Capability != "" {
				if err := addCapabilityProjectionEvidence(world, dataSet, evidenceSet, string(pattern.Capability), prefix); err != nil {
					return nil, err
				}
			}
		}

		for _, dataID := range step.Data {
			if !semanticDataExistsExactlyOnce(world, dataID) {
				return nil, fmt.Errorf("step %s explicit Data %sをexactに解決できません", step.ID, dataID)
			}
			addProjectionEvidence(dataSet, evidenceSet, dataID, "business-explicit-data:"+dataID)
		}
		for _, relationID := range step.Relations {
			if err := addRelationProjectionEvidence(world, relationSet, dataSet, evidenceSet, relationID, "business-explicit-relation"); err != nil {
				return nil, err
			}
		}
		for _, validationID := range step.Validations {
			if err := addValidationProjectionEvidence(world, relationSet, dataSet, evidenceSet, validationID); err != nil {
				return nil, err
			}
		}
		for _, capabilityID := range step.Capabilities {
			if err := addCapabilityProjectionEvidence(world, dataSet, evidenceSet, capabilityID, "business-explicit-capability"); err != nil {
				return nil, err
			}
		}

		row := businessActionDataProjectionRowProbe{
			StepID:                     step.ID,
			Data:                       sortedProjectionKeys(dataSet),
			Relations:                  sortedProjectionKeys(relationSet),
			EvidencePaths:              sortedProjectionKeys(evidenceSet),
			NoDataTraceFromCurrentRefs: len(dataSet) == 0 && len(relationSet) == 0,
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func addValidationProjectionEvidence(world ir.World, relationSet, dataSet, evidenceSet map[string]struct{}, validationID string) error {
	validation, ok := findSemanticValidationExactlyOnce(world, validationID)
	if !ok {
		return fmt.Errorf("Validation %sをexactに解決できません", validationID)
	}
	if validation.Attribute != "" {
		ownerData, ok := findSemanticAttributeOwnerExactlyOnce(world, string(validation.Attribute))
		if !ok {
			return fmt.Errorf("Validation %s Attribute %sのowner Dataをexactに解決できません", validationID, validation.Attribute)
		}
		addProjectionEvidence(dataSet, evidenceSet, ownerData, "validation:"+validationID+"/attribute:"+string(validation.Attribute))
	}
	if validation.Relation != "" {
		if err := addRelationProjectionEvidence(world, relationSet, dataSet, evidenceSet, string(validation.Relation), "validation:"+validationID); err != nil {
			return err
		}
	}
	return nil
}

func addCapabilityProjectionEvidence(world ir.World, dataSet, evidenceSet map[string]struct{}, capabilityID, prefix string) error {
	capability, ok := findSemanticCapabilityExactlyOnce(world, capabilityID)
	if !ok {
		return fmt.Errorf("Capability %sをexactに解決できません", capabilityID)
	}
	addIfSemanticData := func(id string, path string) error {
		if id == "" {
			return nil
		}
		dataCount := semanticDataMatchCount(world, id)
		typeCount := semanticTypeMatchCount(world, id)
		if dataCount == 1 && typeCount == 0 {
			addProjectionEvidence(dataSet, evidenceSet, id, path)
			return nil
		}
		if dataCount == 0 && typeCount == 1 {
			// exactにSemantic Typeへ解決できるpayloadはData projectionへ含めない。
			return nil
		}
		if dataCount == 0 && typeCount == 0 {
			return fmt.Errorf("Capability %s payload identityをexactに解決できません: id=%s (data=%d type=%d)", capabilityID, id, dataCount, typeCount)
		}
		if dataCount > 0 && typeCount > 0 {
			return fmt.Errorf("Capability %s payload %s has Data/Type kind ambiguity (data=%d type=%d)", capabilityID, id, dataCount, typeCount)
		}
		return fmt.Errorf("Capability %s payload identity ambiguity: id=%s (data=%d type=%d)", capabilityID, id, dataCount, typeCount)
	}
	for _, input := range capability.Inputs {
		if err := addIfSemanticData(string(input.Type), prefix+"/capability:"+capabilityID+"/input:"+string(input.Type)); err != nil {
			return err
		}
	}
	if err := addIfSemanticData(string(capability.Output), prefix+"/capability:"+capabilityID+"/output:"+string(capability.Output)); err != nil {
		return err
	}
	for _, outcome := range capability.Outcomes {
		if err := addIfSemanticData(string(outcome.Payload), prefix+"/capability:"+capabilityID+"/outcome:"+string(outcome.Payload)); err != nil {
			return err
		}
	}
	return nil
}

func addRelationProjectionEvidence(world ir.World, relationSet, dataSet, evidenceSet map[string]struct{}, relationID, prefix string) error {
	sourceData, relation, ok := findSemanticRelationExactlyOnce(world, relationID)
	if !ok {
		return fmt.Errorf("Relation %sをexactに解決できません", relationID)
	}
	relationSet[relationID] = struct{}{}
	evidenceSet[prefix+"/relation:"+relationID] = struct{}{}
	if sourceData != "" {
		dataSet[sourceData] = struct{}{}
		evidenceSet[prefix+"/relation-source-data:"+sourceData] = struct{}{}
	}
	if relation.Target != "" {
		targetData := string(relation.Target)
		if !semanticDataExistsExactlyOnce(world, targetData) {
			return fmt.Errorf("Relation %s target Data %sをexactに解決できません", relationID, targetData)
		}
		dataSet[targetData] = struct{}{}
		evidenceSet[prefix+"/relation-target-data:"+targetData] = struct{}{}
	}
	return nil
}

func addProjectionEvidence(dataSet, evidenceSet map[string]struct{}, dataID, evidence string) {
	dataSet[dataID] = struct{}{}
	evidenceSet[evidence] = struct{}{}
}

func semanticDataExistsExactlyOnce(world ir.World, id string) bool {
	return semanticDataMatchCount(world, id) == 1
}

func semanticDataMatchCount(world ir.World, id string) int {
	count := 0
	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			if string(data.ID) == id {
				count++
			}
		}
	}
	return count
}

func semanticTypeMatchCount(world ir.World, id string) int {
	count := 0
	for _, namespace := range world.Semantics {
		for _, semanticType := range namespace.Types {
			if string(semanticType.ID) == id {
				count++
			}
		}
	}
	return count
}

func findSemanticAttributeOwnerExactlyOnce(world ir.World, attributeID string) (string, bool) {
	owner := ""
	count := 0
	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			for _, attribute := range data.Attributes {
				if string(attribute.ID) == attributeID {
					owner = string(data.ID)
					count++
				}
			}
		}
	}
	return owner, count == 1
}

func findSemanticRelationExactlyOnce(world ir.World, relationID string) (string, semantic.Relation, bool) {
	owner := ""
	var found semantic.Relation
	count := 0
	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			for _, relation := range data.Relations {
				if string(relation.ID) == relationID {
					owner = string(data.ID)
					found = relation
					count++
				}
			}
		}
	}
	return owner, found, count == 1
}

func findSemanticValidationExactlyOnce(world ir.World, validationID string) (semantic.Validation, bool) {
	var found semantic.Validation
	count := 0
	for _, namespace := range world.Semantics {
		for _, validation := range namespace.Validations {
			if string(validation.ID) == validationID {
				found = validation
				count++
			}
		}
	}
	return found, count == 1
}

func findSemanticCapabilityExactlyOnce(world ir.World, capabilityID string) (semantic.Capability, bool) {
	var found semantic.Capability
	count := 0
	for _, namespace := range world.Semantics {
		for _, capability := range namespace.Capabilities {
			if string(capability.ID) == capabilityID {
				found = capability
				count++
			}
		}
	}
	return found, count == 1
}

func sortedProjectionKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func businessActionDataRowsByStep(rows []businessActionDataProjectionRowProbe) map[string]businessActionDataProjectionRowProbe {
	result := make(map[string]businessActionDataProjectionRowProbe, len(rows))
	for _, row := range rows {
		result[row.StepID] = row
	}
	return result
}

func assertBusinessActionDataProjection(t *testing.T, row businessActionDataProjectionRowProbe, data, relations []string, noTrace bool) {
	t.Helper()
	if !equalProjectionStringSlice(row.Data, data) {
		t.Fatalf("step %s Data=%v want=%v evidence=%v", row.StepID, row.Data, data, row.EvidencePaths)
	}
	if !equalProjectionStringSlice(row.Relations, relations) {
		t.Fatalf("step %s Relations=%v want=%v evidence=%v", row.StepID, row.Relations, relations, row.EvidencePaths)
	}
	if row.NoDataTraceFromCurrentRefs != noTrace {
		t.Fatalf("step %s NoDataTraceFromCurrentRefs=%v want=%v evidence=%v", row.StepID, row.NoDataTraceFromCurrentRefs, noTrace, row.EvidencePaths)
	}
}

func equalProjectionStringSlice(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
