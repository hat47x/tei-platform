package queryresult_test

import (
	"reflect"
	"testing"

	"tei/internal/driver/queryresult"
	"tei/internal/query"
)

func TestMapUsesExplicitColumnNamesInsteadOfRawOrder(t *testing.T) {
	q := query.EmbeddedSQL{
		ID: "order_entry.recent_orders",
		ResultColumns: []query.ResultColumnMapping{
			{Column: "order_id", Attribute: "sales.order.order_id"},
			{Column: "total_amount", Attribute: "sales.order.total"},
		},
	}

	mapped, err := queryresult.Map(q, queryresult.RawResult{
		Columns: []string{"total_amount", "order_id"},
		Rows:    [][]any{{1200, "O-001"}},
	})
	if err != nil {
		t.Fatalf("明示mappingを適用できません: %v", err)
	}

	wantColumns := []queryresult.Column{
		{Raw: "order_id", Attribute: "sales.order.order_id"},
		{Raw: "total_amount", Attribute: "sales.order.total"},
	}
	if !reflect.DeepEqual(mapped.Columns, wantColumns) {
		t.Fatalf("columns=%#v want=%#v", mapped.Columns, wantColumns)
	}
	if !reflect.DeepEqual(mapped.Rows, [][]any{{"O-001", 1200}}) {
		t.Fatalf("rows=%#v", mapped.Rows)
	}
}

func TestMapDoesNotInferMissingOrAdditionalColumns(t *testing.T) {
	q := query.EmbeddedSQL{
		ID: "order_entry.recent_orders",
		ResultColumns: []query.ResultColumnMapping{
			{Column: "order_id", Attribute: "sales.order.order_id"},
			{Column: "total_amount", Attribute: "sales.order.total"},
		},
	}

	t.Run("mappingにある列が実行結果になければ失敗する", func(t *testing.T) {
		if _, err := queryresult.Map(q, queryresult.RawResult{
			Columns: []string{"order_id"},
			Rows:    [][]any{{"O-001"}},
		}); err == nil {
			t.Fatal("存在しないraw列を補ってmappingしています")
		}
	})

	t.Run("mappingにないraw列を黙って捨てない", func(t *testing.T) {
		if _, err := queryresult.Map(q, queryresult.RawResult{
			Columns: []string{"order_id", "total_amount", "status"},
			Rows:    [][]any{{"O-001", 1200, "open"}},
		}); err == nil {
			t.Fatal("mappingにないraw列を無視しています")
		}
	})

	t.Run("同名raw列をordinalで推測しない", func(t *testing.T) {
		if _, err := queryresult.Map(q, queryresult.RawResult{
			Columns: []string{"order_id", "order_id"},
			Rows:    [][]any{{"O-001", "O-002"}},
		}); err == nil {
			t.Fatal("重複したraw列名をordinalで解決しています")
		}
	})

	t.Run("row幅がraw列数と違えば失敗する", func(t *testing.T) {
		if _, err := queryresult.Map(q, queryresult.RawResult{
			Columns: []string{"order_id", "total_amount"},
			Rows:    [][]any{{"O-001"}},
		}); err == nil {
			t.Fatal("raw resultの列数不一致を受理しています")
		}
	})
}

func TestMapRejectsAmbiguousAuthoredMapping(t *testing.T) {
	raw := queryresult.RawResult{
		Columns: []string{"order_id", "total_amount"},
		Rows:    [][]any{{"O-001", 1200}},
	}

	t.Run("同じraw列を二度mappingしない", func(t *testing.T) {
		q := query.EmbeddedSQL{
			ID: "order_entry.recent_orders",
			ResultColumns: []query.ResultColumnMapping{
				{Column: "order_id", Attribute: "sales.order.order_id"},
				{Column: "order_id", Attribute: "sales.order.customer_id"},
			},
		}
		if _, err := queryresult.Map(q, raw); err == nil {
			t.Fatal("重複したresult column mappingを受理しています")
		}
	})

	t.Run("同じSemantic attributeへ二つのraw列をmappingしない", func(t *testing.T) {
		q := query.EmbeddedSQL{
			ID: "order_entry.recent_orders",
			ResultColumns: []query.ResultColumnMapping{
				{Column: "order_id", Attribute: "sales.order.order_id"},
				{Column: "total_amount", Attribute: "sales.order.order_id"},
			},
		}
		if _, err := queryresult.Map(q, raw); err == nil {
			t.Fatal("重複したSemantic attribute mappingを受理しています")
		}
	})

	t.Run("空のraw列名やSemantic attributeを受理しない", func(t *testing.T) {
		q := query.EmbeddedSQL{
			ID: "order_entry.recent_orders",
			ResultColumns: []query.ResultColumnMapping{
				{Column: "", Attribute: "sales.order.order_id"},
			},
		}
		if _, err := queryresult.Map(q, raw); err == nil {
			t.Fatal("名前のないresult column mappingを受理しています")
		}

		q.ResultColumns = []query.ResultColumnMapping{{Column: "order_id", Attribute: ""}}
		if _, err := queryresult.Map(q, raw); err == nil {
			t.Fatal("Semantic attributeのないmappingを受理しています")
		}
	})
}

func TestMapRequiresAuthoredResultMapping(t *testing.T) {
	q := query.EmbeddedSQL{ID: "order_entry.recent_orders"}
	if _, err := queryresult.Map(q, queryresult.RawResult{
		Columns: []string{"order_id"},
		Rows:    [][]any{{"O-001"}},
	}); err == nil {
		t.Fatal("result mappingのないQueryをSemantic resultへ変換しています")
	}
}
