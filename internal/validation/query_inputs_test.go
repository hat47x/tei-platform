package validation

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/query"
)

func queryInputWorld() ir.World {
	w := validApplicationPatternWorld()
	w.Queries[0].Inputs = []query.Input{{Name: "status", Type: "std.code"}}
	return w
}

func TestValidateQueryInputs(t *testing.T) {
	for _, diagnostic := range validateQueryInputs(queryInputWorld()) {
		t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
	}
}

func TestValidateQueryInputsRejectDuplicate(t *testing.T) {
	w := queryInputWorld()
	w.Queries[0].Inputs = append(w.Queries[0].Inputs, query.Input{Name: "status", Type: "std.code"})
	if diagnosticByCode(validateQueryInputs(w), "QUERY_INPUT_DUPLICATE") == nil {
		t.Fatal("duplicate query input was accepted")
	}
}

func TestValidateQueryInputsCheckSQLReference(t *testing.T) {
	w := queryInputWorld()
	w.Queries[0].Source = "select order_id from sales.order where status = :status"
	if diagnosticByCode(validateQueryInputs(w), "QUERY_INPUT_NOT_REFERENCED") != nil {
		t.Fatal("referenced query input was reported")
	}
	w.Queries[0].Source = "select order_id from sales.order"
	if diagnosticByCode(validateQueryInputs(w), "QUERY_INPUT_NOT_REFERENCED") == nil {
		t.Fatal("unreferenced query input was accepted")
	}
}

func TestValidateQueryInputsRejectUnknownType(t *testing.T) {
	w := queryInputWorld()
	w.Queries[0].Inputs = []query.Input{{Name: "status", Type: "missing.type"}}
	if diagnosticByCode(validateQueryInputs(w), "UNKNOWN_QUERY_INPUT_TYPE") == nil {
		t.Fatal("unknown query input type was accepted")
	}
}
