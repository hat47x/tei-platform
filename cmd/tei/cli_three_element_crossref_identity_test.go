package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/threeelement"
)

// The cross-reference response is the only CLI surface that shows Data竊奪ata
// relations, and until #6113 it dropped the relation identity: two relations
// between the same pair of Data elements could not be told apart from the
// output, so the Table/Graph distinction SDM-01 asks for did not exist for this
// Consumer. This pins the identity in the real command response.
//
// The expectation is fixed by hand: the model declares r-2 before r-1, and the
// response must carry both ids in the projection's declared order.
func TestThreeElementCrossReferenceCarriesTheRelationIdentity(t *testing.T) {
	model := threeelement.NewThreeElementModel("parallel", "Parallel relations")
	model.DataElements = []threeelement.DataElement{
		{ID: "order", Name: "Order"},
		{
			ID: "customer", Name: "Customer",
			Relations: []threeelement.DataRelation{
				{ID: "r-2", TargetID: "order", Kind: "many-to-one", Directed: true},
				{ID: "r-1", TargetID: "order", Kind: "many-to-one", Directed: true},
			},
		},
	}
	raw, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	modelPath := filepath.Join(t.TempDir(), "model.json")
	if err := os.WriteFile(modelPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	code, err := runThreeElementCommand([]string{"cross-reference", "-model", modelPath}, &out)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}

	var response struct {
		OK     bool `json:"ok"`
		Result struct {
			DataData []threeelement.DataDataCrossReference `json:"dataData"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out.String()), &response); err != nil {
		t.Fatal(err)
	}
	if !response.OK {
		t.Fatalf("cross-reference failed: %s", out.String())
	}
	rows := response.Result.DataData
	if len(rows) != 2 {
		t.Fatalf("expected two parallel relations in the response, got %+v", rows)
	}
	if rows[0].RelationID != "r-1" || rows[1].RelationID != "r-2" {
		t.Fatalf("response does not carry the relation identity in order: %+v", rows)
	}
	for _, row := range rows {
		if row.SourceID != "customer" || row.TargetID != "order" || row.Kind != "many-to-one" {
			t.Fatalf("relation %s lost a projected field: %+v", row.RelationID, row)
		}
	}
}
