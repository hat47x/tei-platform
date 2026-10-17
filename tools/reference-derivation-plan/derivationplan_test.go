package derivationplan

import (
	"reflect"
	"testing"
)

// The fixture follows Data Derivation D2 (#1640): available count = stock -
// reserved for five products, where C and D tie at 50.
func fixtureRows() []Row {
	return []Row{
		{ID: "A", Stock: "120", Reserved: "100"},
		{ID: "B", Stock: "100", Reserved: "10"},
		{ID: "C", Stock: "80", Reserved: "30"},
		{ID: "D", Stock: "70", Reserved: "20"},
		{ID: "E", Stock: "40", Reserved: "70"},
	}
}

func fixtureRules() []DerivedRule {
	return []DerivedRule{{Field: "availableCount", Expression: "stock - reserved"}}
}

func pipeline(steps ...PlanStep) Input {
	return Input{Rows: fixtureRows(), DerivedRules: fixtureRules(), Steps: steps}
}

func sortDesc() PlanStep {
	return PlanStep{Op: OpSort, Field: "availableCount", Direction: SortDesc, TieBreaker: "id"}
}

func TestFilterSortLimitMatchTheFixture(t *testing.T) {
	report := Run(pipeline(
		PlanStep{Op: OpDerive, Field: "availableCount"},
		PlanStep{Op: OpFilter, Field: "availableCount", Comparator: ">=", Operand: "50"},
		sortDesc(),
	))
	if !report.Accepted || report.Code != CodeRows || !reflect.DeepEqual(report.IDs, []string{"B", "C", "D"}) {
		t.Fatalf("report=%+v want B,C,D", report)
	}
}

func TestLimitAndOffsetTakeTheSortedPrefix(t *testing.T) {
	base := []PlanStep{
		PlanStep{Op: OpDerive, Field: "availableCount"},
		PlanStep{Op: OpFilter, Field: "availableCount", Comparator: ">=", Operand: "50"},
		sortDesc(),
	}
	{
		steps := append(append([]PlanStep{}, base...), PlanStep{Op: OpLimit, Limit: 1})
		report := Run(pipeline(steps...))
		if !report.Accepted || !reflect.DeepEqual(report.IDs, []string{"B"}) {
			t.Fatalf("report=%+v want B", report)
		}
	}
	{
		steps := append(append([]PlanStep{}, base...), PlanStep{Op: OpLimit, Limit: 2, Offset: 1})
		report := Run(pipeline(steps...))
		if !report.Accepted || !reflect.DeepEqual(report.IDs, []string{"C", "D"}) {
			t.Fatalf("report=%+v want C,D", report)
		}
	}
}

func TestCountCountsAfterTheFilter(t *testing.T) {
	report := Run(pipeline(
		PlanStep{Op: OpDerive, Field: "availableCount"},
		PlanStep{Op: OpFilter, Field: "availableCount", Comparator: ">=", Operand: "50"},
		PlanStep{Op: OpCount},
	))
	if !report.Accepted || report.Code != CodeCount || report.Count == nil || *report.Count != 3 {
		t.Fatalf("report=%+v want count 3", report)
	}
}

func TestSortWithUndeclaredTieOrderDoesNotGuaranteeStablePaging(t *testing.T) {
	// C and D tie at 50; without a tie-breaker the page split is not stable.
	report := Run(pipeline(
		PlanStep{Op: OpDerive, Field: "availableCount"},
		PlanStep{Op: OpFilter, Field: "availableCount", Comparator: ">=", Operand: "50"},
		PlanStep{Op: OpSort, Field: "availableCount", Direction: SortDesc},
		PlanStep{Op: OpLimit, Limit: 1},
	))
	if report.Accepted || report.Code != CodeNoStableOrder {
		t.Fatalf("report=%+v want NO_STABLE_ORDER", report)
	}
}

func TestSortWithoutDirectionDoesNotImposeAPhysicalOrder(t *testing.T) {
	report := Run(pipeline(
		PlanStep{Op: OpDerive, Field: "availableCount"},
		PlanStep{Op: OpSort, Field: "availableCount"},
	))
	if report.Accepted || report.Code != CodeNoStableOrder {
		t.Fatalf("report=%+v want NO_STABLE_ORDER", report)
	}
}

func TestDeriveBeforeFilterIsRequired(t *testing.T) {
	report := Run(pipeline(
		PlanStep{Op: OpFilter, Field: "availableCount", Comparator: ">=", Operand: "50"},
	))
	if report.Accepted || report.Code != CodeDeriveUnresolved {
		t.Fatalf("report=%+v want DERIVED_ATTR_UNRESOLVED", report)
	}
}

func TestFilterAfterLimitIsAnIllegalPlan(t *testing.T) {
	report := Run(pipeline(
		PlanStep{Op: OpDerive, Field: "availableCount"},
		PlanStep{Op: OpLimit, Limit: 1},
		PlanStep{Op: OpFilter, Field: "availableCount", Comparator: ">=", Operand: "50"},
	))
	if report.Accepted || report.Code != CodeIllegalPlanOrder {
		t.Fatalf("report=%+v want ILLEGAL_PLAN_ORDER", report)
	}
}

func TestUnknownFieldIsRejected(t *testing.T) {
	report := Run(pipeline(
		PlanStep{Op: OpDerive, Field: "availableCount"},
		PlanStep{Op: OpFilter, Field: "margin", Comparator: ">=", Operand: "50"},
	))
	if report.Accepted || report.Code != CodeUnknownField {
		t.Fatalf("report=%+v want UNKNOWN_FIELD", report)
	}
}

func TestMissingInputStopsThePlan(t *testing.T) {
	rows := fixtureRows()
	rows[2].StockState = StateMissing
	report := Run(Input{
		Rows:         rows,
		DerivedRules: fixtureRules(),
		Steps: []PlanStep{
			PlanStep{Op: OpDerive, Field: "availableCount"},
			PlanStep{Op: OpFilter, Field: "availableCount", Comparator: ">=", Operand: "50"},
		},
	})
	if report.Accepted || report.Code != CodeIncompleteInput {
		t.Fatalf("report=%+v want INCOMPLETE_INPUT", report)
	}
}

func TestNullComparisonStaysUnmatchedAndIsDisclosed(t *testing.T) {
	rows := fixtureRows()
	rows[2].StockState = StateNull
	report := Run(Input{
		Rows:         rows,
		DerivedRules: fixtureRules(),
		Steps: []PlanStep{
			PlanStep{Op: OpDerive, Field: "availableCount"},
			PlanStep{Op: OpFilter, Field: "availableCount", Comparator: ">=", Operand: "50"},
		},
	})
	if !report.Accepted || report.Code != CodeRows || report.NullInputCount == nil || *report.NullInputCount != 1 {
		t.Fatalf("report=%+v want rows with one null disclosed", report)
	}
	if !reflect.DeepEqual(report.IDs, []string{"B", "D"}) {
		t.Fatalf("ids=%v want B,D", report.IDs)
	}
}

func TestUnrecognizedExpressionIsNotComputed(t *testing.T) {
	report := Run(Input{
		Rows:         fixtureRows(),
		DerivedRules: []DerivedRule{{Field: "availableCount", Expression: "stock * reserved"}},
		Steps:        []PlanStep{PlanStep{Op: OpDerive, Field: "availableCount"}},
	})
	if report.Accepted || report.Code != CodeComputeUnavailable {
		t.Fatalf("report=%+v want COMPUTE_UNAVAILABLE", report)
	}
}
