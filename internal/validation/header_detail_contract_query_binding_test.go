package validation

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/semantic"
)

func headerDetailContractQueryBindingWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Types: []semantic.Type{
					{ID: "sales.order_id", Symbol: "OrderId", Kind: semantic.TypeAlias, Underlying: "std.identifier"},
					{ID: "sales.order_line_id", Symbol: "OrderLineId", Kind: semantic.TypeAlias, Underlying: "std.identifier"},
				},
				Data: []semantic.Data{
					{
						ID: "sales.order",
						Attributes: []semantic.Attribute{
							{ID: "sales.order.order_id", Symbol: "orderId", TypeRef: "sales.order_id"},
						},
						Relations: []semantic.Relation{
							{ID: "sales.order.lines", Symbol: "lines", Target: "sales.order_line", Cardinality: semantic.CardinalityMany},
						},
					},
					{
						ID: "sales.order_line",
						Attributes: []semantic.Attribute{
							{ID: "sales.order_line.order_line_id", Symbol: "orderLineId", TypeRef: "sales.order_line_id"},
						},
					},
				},
			},
			{
				ID: "std",
				Types: []semantic.Type{
					{ID: "std.identifier", Symbol: "Identifier", Kind: semantic.TypePrimitive},
				},
			},
		},
		Applications: []application.Application{
			{
				ID: "order_entry",
				Databases: []application.DatabaseRequirement{
					{
						Alias:          "sales",
						TargetDatabase: "sales_core",
						Access:         database.AccessReadWrite,
						RequiredContracts: []database.ContractRequirement{
							{ContractID: "sales_core.order_line", CompatibleMajor: 1},
						},
					},
				},
				Patterns: []application.Pattern{
					{ID: "order_entry.order_lines", Kind: application.PatternHeaderDetail, Relation: "sales.order.lines"},
				},
			},
		},
		Databases: []database.Database{
			{
				ID: "sales_core",
				Contracts: []database.Contract{
					{
						ID:      "sales_core.order_line",
						Version: database.ContractVersion{Major: 1},
						Queries: []database.QueryContract{
							{
								Name: "find_by_order",
								Inputs: []database.QueryInput{
									{Name: "order_id", Type: "sales.order_id"},
								},
								Result:            "sales.order_line",
								ResultCardinality: database.QueryResultMany,
							},
						},
					},
				},
			},
		},
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
}

// A rule whose operator reads a column of every row is asked of the rows of
// the relation, so a binding that does not state it returns every row leaves
// the rule answerable only as undecided or not-satisfied. The warning has to
// name that rule, as it already names the built-in kinds about all the rows.
func TestValidateHeaderDetailContractQueryBindingsWarnsAboutARowScopedOperatorRule(t *testing.T) {
	world := headerDetailContractQueryBindingWorld()
	world.Semantics[0].Data[1].Attributes = append(world.Semantics[0].Data[1].Attributes,
		semantic.Attribute{ID: "sales.order_line.quantity", Symbol: "quantity", TypeRef: "std.decimal"})
	world.Semantics[0].Operators = append(world.Semantics[0].Operators, semantic.Operator{
		ID: "sales.every_at_most", Symbol: "every_at_most", Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
			{Name: "limit", Type: semantic.OperatorInputNumber, OfEachRow: true},
		},
		// A refutation is about the rows a rule reads, so the declaration
		// states it is about rows at all.
		RefutesOnSomeRows: true,
	})
	world.Semantics[0].Validations = append(world.Semantics[0].Validations, semantic.Validation{
		ID: "sales.order_line.quantity_at_most", Symbol: "quantity_at_most",
		Kind: semantic.ValidationOperator, Attribute: "sales.order_line.quantity",
		Operator: "sales.every_at_most", Relation: "sales.order.lines",
	})
	world.Semantics[1].Types = append(world.Semantics[1].Types,
		semantic.Type{ID: "std.decimal", Symbol: "Decimal", Kind: semantic.TypePrimitive})
	world.Applications[0].Validations = []core.CanonicalID{"sales.order_line.quantity_at_most"}

	var named []string
	for _, diagnostic := range Validate(world) {
		if diagnostic.Code != "HEADER_DETAIL_BINDING_COVERAGE_UNDECLARED" {
			continue
		}
		named = append(named, diagnostic.Message)
	}
	if len(named) != 1 {
		t.Fatalf("coverage warnings=%d, want one for the row-scoped rule: %v", len(named), named)
	}
	if !strings.Contains(named[0], "sales.order_line.quantity_at_most") {
		t.Fatalf("the warning does not name the row-scoped rule: %q", named[0])
	}
}

// A rule decided by an operator that reads one record's values is about every
// record, not about the rows of a relation, so it needs no covers statement.
func TestValidateHeaderDetailContractQueryBindingsLeavesARecordRuleAlone(t *testing.T) {
	world := headerDetailContractQueryBindingWorld()
	world.Semantics[0].Data[1].Attributes = append(world.Semantics[0].Data[1].Attributes,
		semantic.Attribute{ID: "sales.order_line.quantity", Symbol: "quantity", TypeRef: "std.decimal"})
	world.Semantics[0].Operators = append(world.Semantics[0].Operators, semantic.Operator{
		ID: "sales.every_at_most", Symbol: "every_at_most", Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber},
		},
	})
	world.Semantics[0].Validations = append(world.Semantics[0].Validations, semantic.Validation{
		ID: "sales.order_line.quantity_at_most", Symbol: "quantity_at_most",
		Kind: semantic.ValidationOperator, Attribute: "sales.order_line.quantity",
		Operator: "sales.every_at_most",
	})
	world.Semantics[1].Types = append(world.Semantics[1].Types,
		semantic.Type{ID: "std.decimal", Symbol: "Decimal", Kind: semantic.TypePrimitive})
	world.Applications[0].Validations = []core.CanonicalID{"sales.order_line.quantity_at_most"}

	for _, diagnostic := range Validate(world) {
		if diagnostic.Code == "HEADER_DETAIL_BINDING_COVERAGE_UNDECLARED" {
			t.Fatalf("a record rule was told to declare covers: %s", diagnostic.Message)
		}
	}
}

// A rule that is about the rows of a relation cannot be answered from a
// binding that reads another relation: the read refuses it with "these are the
// rows of relation ..., not of ...". Validate reports it, so a Workspace does
// not pass here and fail at the first screen.
func TestValidateHeaderDetailContractQueryBindingsRefusesARuleOverAnotherRelation(t *testing.T) {
	world := headerDetailContractQueryBindingWorld()
	// A second many relation to the same Data, so both relations are valid
	// declarations and only the identity differs.
	world.Semantics[0].Data[0].Relations = append(world.Semantics[0].Data[0].Relations, semantic.Relation{
		ID: "sales.order.lines_copy", Symbol: "linesCopy",
		Target: "sales.order_line", Cardinality: semantic.CardinalityMany,
	})
	world.Semantics[0].Validations = []semantic.Validation{{
		ID: "sales.order_line.line_number_unique", Symbol: "line_number_unique",
		Kind:      semantic.ValidationRelationUnique,
		Attribute: "sales.order_line.order_line_id",
		Relation:  "sales.order.lines_copy",
	}}
	world.Applications[0].Validations = []core.CanonicalID{"sales.order_line.line_number_unique"}

	var found bool
	for _, diagnostic := range Validate(world) {
		if diagnostic.Code != "HEADER_DETAIL_BINDING_RULE_OTHER_RELATION" {
			continue
		}
		found = true
		if !strings.Contains(diagnostic.Message, "sales.order.lines_copy") ||
			!strings.Contains(diagnostic.Message, "sales.order.lines") {
			t.Fatalf("the finding does not name both relations: %q", diagnostic.Message)
		}
	}
	if !found {
		t.Fatal("a rule over another relation was not reported")
	}
}

func TestValidateHeaderDetailContractQueryBindingsAcceptsKeylessManyRelation(t *testing.T) {
	world := headerDetailContractQueryBindingWorld()
	if len(world.Semantics[0].Data[0].Relations[0].Keys) != 0 {
		t.Fatal("fixtureがkeyless Relationではありません")
	}
	if diagnostics := ValidateHeaderDetailContractQueryBindings(world); len(diagnostics) != 0 {
		t.Fatalf("keylessなmany Relationの明示Contract Query bindingを拒否しています: %#v", diagnostics)
	}
}

func TestValidateHeaderDetailContractQueryBindingsRejectsDuplicatePatternRealization(t *testing.T) {
	world := headerDetailContractQueryBindingWorld()
	world.HeaderDetailContractQueryBindings = append(world.HeaderDetailContractQueryBindings, world.HeaderDetailContractQueryBindings[0])
	assertHeaderDetailBindingDiagnostic(t, ValidateHeaderDetailContractQueryBindings(world), "DUPLICATE_HEADER_DETAIL_CONTRACT_QUERY_BINDING")
}

func TestValidateHeaderDetailContractQueryBindingsRequiresHeaderDetailPattern(t *testing.T) {
	world := headerDetailContractQueryBindingWorld()
	world.Applications[0].Patterns[0].Kind = application.PatternReference
	assertHeaderDetailBindingDiagnostic(t, ValidateHeaderDetailContractQueryBindings(world), "HEADER_DETAIL_BINDING_REQUIRES_HEADER_DETAIL_PATTERN")
}

func TestValidateHeaderDetailContractQueryBindingsRequiresReadableDatasourceAndMatchingDatabase(t *testing.T) {
	t.Run("write-only", func(t *testing.T) {
		world := headerDetailContractQueryBindingWorld()
		world.Applications[0].Databases[0].Access = database.AccessWrite
		assertHeaderDetailBindingDiagnostic(t, ValidateHeaderDetailContractQueryBindings(world), "HEADER_DETAIL_BINDING_DATASOURCE_NOT_READABLE")
	})

	t.Run("database mismatch", func(t *testing.T) {
		world := headerDetailContractQueryBindingWorld()
		world.HeaderDetailContractQueryBindings[0].Target.Database = "archive_sales"
		assertHeaderDetailBindingDiagnostic(t, ValidateHeaderDetailContractQueryBindings(world), "HEADER_DETAIL_BINDING_DATABASE_MISMATCH")
	})
}

func TestValidateHeaderDetailContractQueryBindingsRequiresApplicationContract(t *testing.T) {
	world := headerDetailContractQueryBindingWorld()
	world.Applications[0].Databases[0].RequiredContracts = nil
	assertHeaderDetailBindingDiagnostic(t, ValidateHeaderDetailContractQueryBindings(world), "HEADER_DETAIL_BINDING_CONTRACT_NOT_REQUIRED")
}

func TestValidateHeaderDetailContractQueryBindingsUsesRelationSourceAttributeOnly(t *testing.T) {
	world := headerDetailContractQueryBindingWorld()
	world.HeaderDetailContractQueryBindings[0].Inputs[0].SourceAttribute = "sales.order_line.order_line_id"
	assertHeaderDetailBindingDiagnostic(t, ValidateHeaderDetailContractQueryBindings(world), "HEADER_DETAIL_BINDING_SOURCE_ATTRIBUTE_OUTSIDE_RELATION_SOURCE")
}

func TestValidateHeaderDetailContractQueryBindingsChecksQueryInputAndSemanticType(t *testing.T) {
	t.Run("unknown input", func(t *testing.T) {
		world := headerDetailContractQueryBindingWorld()
		world.HeaderDetailContractQueryBindings[0].Inputs[0].Input = "missing"
		assertHeaderDetailBindingDiagnostic(t, ValidateHeaderDetailContractQueryBindings(world), "HEADER_DETAIL_BINDING_INPUT_NOT_IN_SIGNATURE")
	})

	t.Run("type mismatch", func(t *testing.T) {
		world := headerDetailContractQueryBindingWorld()
		world.Databases[0].Contracts[0].Queries[0].Inputs[0].Type = "sales.order_line_id"
		assertHeaderDetailBindingDiagnostic(t, ValidateHeaderDetailContractQueryBindings(world), "HEADER_DETAIL_BINDING_INPUT_TYPE_MISMATCH")
	})

	t.Run("unbound input", func(t *testing.T) {
		world := headerDetailContractQueryBindingWorld()
		world.Databases[0].Contracts[0].Queries[0].Inputs = append(
			world.Databases[0].Contracts[0].Queries[0].Inputs,
			database.QueryInput{Name: "tenant", Type: "sales.order_id"},
		)
		assertHeaderDetailBindingDiagnostic(t, ValidateHeaderDetailContractQueryBindings(world), "HEADER_DETAIL_BINDING_QUERY_INPUT_NOT_BOUND")
	})
}

func TestValidateHeaderDetailContractQueryBindingsChecksCollectionResultContract(t *testing.T) {
	t.Run("result Data", func(t *testing.T) {
		world := headerDetailContractQueryBindingWorld()
		world.Databases[0].Contracts[0].Queries[0].Result = "sales.order"
		assertHeaderDetailBindingDiagnostic(t, ValidateHeaderDetailContractQueryBindings(world), "HEADER_DETAIL_BINDING_RESULT_DATA_MISMATCH")
	})

	t.Run("cardinality", func(t *testing.T) {
		world := headerDetailContractQueryBindingWorld()
		world.Databases[0].Contracts[0].Queries[0].ResultCardinality = database.QueryResultZeroOrOne
		assertHeaderDetailBindingDiagnostic(t, ValidateHeaderDetailContractQueryBindings(world), "HEADER_DETAIL_BINDING_RESULT_CARDINALITY_NOT_MANY")
	})
}

func TestValidateHeaderDetailContractQueryBindingsDoesNotRequireRelationKey(t *testing.T) {
	world := headerDetailContractQueryBindingWorld()
	world.Semantics[0].Data[0].Relations[0].Keys = nil
	if diagnostics := ValidateHeaderDetailContractQueryBindings(world); len(diagnostics) != 0 {
		t.Fatalf("header-detail traversalへRelationKeyを暗黙必須化しています: %#v", diagnostics)
	}
}

func assertHeaderDetailBindingDiagnostic(t *testing.T, diagnostics []Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			if diagnostic.Subject.Kind != "header-detail-contract-query-binding" {
				t.Fatalf("Diagnostic %sのSubject kind=%s", code, diagnostic.Subject.Kind)
			}
			return
		}
	}
	codes := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	t.Fatalf("Diagnostic %sがありません: %v", code, codes)
}
