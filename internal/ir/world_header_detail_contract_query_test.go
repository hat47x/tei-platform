package ir

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/realization"
)

func TestWorldOmitsEmptyHeaderDetailContractQueryBindings(t *testing.T) {
	encoded, err := json.Marshal(World{})
	if err != nil {
		t.Fatalf("WorldをJSONへ変換できません: %v", err)
	}
	if strings.Contains(string(encoded), "headerDetailContractQueryBindings") {
		t.Fatalf("空のheader-detail Contract Query bindingが既存World JSONへ追加されています: %s", encoded)
	}
}

func TestWorldSerializesExplicitHeaderDetailContractQueryBinding(t *testing.T) {
	world := World{
		HeaderDetailContractQueryBindings: []realization.HeaderDetailContractQueryBinding{
			{
				Application: "order_entry",
				Pattern:     "order_entry.order_lines",
				Datasource:  "sales",
				Target: realization.ContractQueryTarget{
					Database: "sales_core",
					Contract: "sales_core.order_line",
					Query:    "find_by_order",
				},
				Inputs: []realization.HeaderDetailContractQueryInputBinding{
					{SourceAttribute: "sales.order.order_id", Input: "order_id"},
				},
			},
		},
	}

	encoded, err := json.Marshal(world)
	if err != nil {
		t.Fatalf("header-detail Contract Query bindingを含むWorldをJSONへ変換できません: %v", err)
	}
	text := string(encoded)
	for _, expected := range []string{
		`"headerDetailContractQueryBindings"`,
		`"application":"order_entry"`,
		`"pattern":"order_entry.order_lines"`,
		`"sourceAttribute":"sales.order.order_id"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("World JSONに明示binding %sがありません: %s", expected, text)
		}
	}
}
