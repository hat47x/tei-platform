package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/query"
	"tei/internal/syntax"
)

func TestParseQueryAuthoringReadsExplicitResultMapping(t *testing.T) {
	src := `query order_entry.recent_orders {
    datasource sales
    sql postgres file "recent_orders.sql"
    result {
        column order_id -> sales.order.order_id
        column total_amount -> sales.order.total
    }
}`

	got, err := syntax.NewParser(src).ParseQueryAuthoring(core.MustCanonicalID("order_entry"))
	if err != nil {
		t.Fatalf("result mappingを含むQueryを読み取れません: %v", err)
	}
	want := []query.ResultColumnMapping{
		{Column: "order_id", Attribute: "sales.order.order_id"},
		{Column: "total_amount", Attribute: "sales.order.total"},
	}
	if len(got.Value.ResultColumns) != len(want) {
		t.Fatalf("ResultColumns=%#v", got.Value.ResultColumns)
	}
	for i := range want {
		if got.Value.ResultColumns[i] != want[i] {
			t.Fatalf("ResultColumns[%d]=%#v want=%#v", i, got.Value.ResultColumns[i], want[i])
		}
	}
}

func TestParseQueryAuthoringRejectsDuplicateResultColumn(t *testing.T) {
	src := `query order_entry.recent_orders {
    datasource sales
    sql postgres file "recent_orders.sql"
    result {
        column order_id -> sales.order.order_id
        column order_id -> sales.order.customer_id
    }
}`

	_, err := syntax.NewParser(src).ParseQueryAuthoring(core.MustCanonicalID("order_entry"))
	if err == nil || !strings.Contains(err.Error(), `result column "order_id"が重複しています`) {
		t.Fatalf("重複result columnを拒否していません: %v", err)
	}
}

func TestParseQueryAuthoringRejectsDuplicateResultAttribute(t *testing.T) {
	src := `query order_entry.recent_orders {
    datasource sales
    sql postgres file "recent_orders.sql"
    result {
        column order_id -> sales.order.order_id
        column copied_order_id -> sales.order.order_id
    }
}`

	_, err := syntax.NewParser(src).ParseQueryAuthoring(core.MustCanonicalID("order_entry"))
	if err == nil || !strings.Contains(err.Error(), "result attribute sales.order.order_idが重複しています") {
		t.Fatalf("重複result attributeを拒否していません: %v", err)
	}
}

func TestParseQueryAuthoringRejectsEmptyResultMapping(t *testing.T) {
	src := `query order_entry.recent_orders {
    datasource sales
    sql postgres file "recent_orders.sql"
    result {
    }
}`

	_, err := syntax.NewParser(src).ParseQueryAuthoring(core.MustCanonicalID("order_entry"))
	if err == nil || !strings.Contains(err.Error(), "少なくとも一つのcolumn mapping") {
		t.Fatalf("空のresult mappingを拒否していません: %v", err)
	}
}

func TestParseQueryAuthoringRejectsMultipleResultBlocks(t *testing.T) {
	src := `query order_entry.recent_orders {
    datasource sales
    sql postgres file "recent_orders.sql"
    result {
        column order_id -> sales.order.order_id
    }
    result {
        column total_amount -> sales.order.total
    }
}`

	_, err := syntax.NewParser(src).ParseQueryAuthoring(core.MustCanonicalID("order_entry"))
	if err == nil || !strings.Contains(err.Error(), "resultは一つのQueryで一度だけ定義できます") {
		t.Fatalf("複数result blockを拒否していません: %v", err)
	}
}
