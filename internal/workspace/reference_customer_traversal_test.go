package workspace_test

import (
	"path/filepath"
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestReferenceCustomerTraversalKeepsThreeLayerMeaningExplicit(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	app := referenceCustomerApplication(t, result.World.Applications)
	pattern := referenceCustomerPattern(t, app.Patterns)
	if pattern.Relation != "sales.order.customer" || pattern.Datasource != "master" || pattern.ContractQuery == nil {
		t.Fatalf("customer referenceの明示境界が期待値と異なります: %#v", pattern)
	}
	if pattern.ContractQuery.Contract != "master_data.customer" || pattern.ContractQuery.Query != "find_by_id" {
		t.Fatalf("Contract / Query identityが期待値と異なります: %#v", pattern.ContractQuery)
	}
	if len(pattern.ContractQuery.Inputs) != 1 ||
		pattern.ContractQuery.Inputs[0].TargetAttribute != "customer.customer.customer_id" ||
		pattern.ContractQuery.Inputs[0].Input != "id" {
		t.Fatalf("Relation target keyからQuery inputへの明示bindingが期待値と異なります: %#v", pattern.ContractQuery.Inputs)
	}

	relation := referenceCustomerRelation(t, result.World.Semantics)
	if len(relation.Keys) != 1 ||
		relation.Keys[0].SourceAttribute != "sales.order.customer_id" ||
		relation.Keys[0].TargetAttribute != "customer.customer.customer_id" {
		t.Fatalf("customer Relationのtraversal keyが期待値と異なります: %#v", relation.Keys)
	}

	query := referenceCustomerContractQuery(t, result.World.Databases)
	if len(query.Inputs) != 1 || query.Inputs[0].Name != "id" || query.Inputs[0].Type != "customer.customer_id" {
		t.Fatalf("find_by_idのtyped inputが期待値と異なります: %#v", query.Inputs)
	}
	if query.Result != "customer.customer" || query.ResultCardinality != database.QueryResultZeroOrOne {
		t.Fatalf("find_by_idのresult契約が期待値と異なります: %#v", query)
	}

	validators := []struct {
		name string
		run  func() []validation.Diagnostic
	}{
		{name: "Semantic Relation key", run: func() []validation.Diagnostic { return validation.ValidateSemanticRelationKeys(result.World) }},
		{name: "Database Contract Query signature", run: func() []validation.Diagnostic {
			return validation.ValidateDatabaseContractQuerySignatures(result.World)
		}},
		{name: "Reference Contract Query invocation", run: func() []validation.Diagnostic {
			return validation.ValidateReferenceContractQueryInvocations(result.World)
		}},
		{name: "Reference Contract Query cross-layer", run: func() []validation.Diagnostic {
			return validation.ValidateReferenceContractQueryCrossLayer(result.World)
		}},
	}
	for _, validator := range validators {
		if diagnostics := validator.run(); len(diagnostics) != 0 {
			t.Fatalf("%s Validationが実Referenceを拒否しています: %#v", validator.name, diagnostics)
		}
	}
}

// Two authored applications share a query contract, not a business relation,
// datasource alias, or a new shared Plugin implementation.
func TestTwoApplicationsUseSharedCustomerContractWithLocalBindings(t *testing.T) {
	ws := filepath.Join(repoRoot(t), "reference", "workspace")
	loaded, err := workspace.Load(ws)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := workspace.BuildSourceIndex(ws, loaded)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		application string
		pattern     string
		relation    string
		datasource  string
		source      string
	}{
		{
			application: "order_entry",
			pattern:     "order_entry.customer_reference",
			relation:    "sales.order.customer",
			datasource:  "master",
			source:      "applications/order-entry/patterns/customer-reference.tei",
		},
		{
			application: "repair_intake",
			pattern:     "repair_intake.customer_reference",
			relation:    "repair_intake.intake.customer",
			datasource:  "customer_master",
			source:      "applications/repair-intake/patterns/customer-reference.tei",
		},
	}
	for _, tc := range cases {
		t.Run(tc.application, func(t *testing.T) {
			app := referenceApplicationByID(t, loaded.World.Applications, tc.application)
			pattern := applicationPatternByID(t, app.Patterns, tc.pattern)
			if pattern.Kind != application.PatternReference ||
				string(pattern.Relation) != tc.relation ||
				string(pattern.Datasource) != tc.datasource {
				t.Fatalf("案件固有のRelation／接続名が変わりました: %#v", pattern)
			}
			if pattern.ContractQuery == nil ||
				pattern.ContractQuery.Contract != "master_data.customer" ||
				pattern.ContractQuery.Query != "find_by_id" ||
				len(pattern.ContractQuery.Inputs) != 1 ||
				pattern.ContractQuery.Inputs[0].TargetAttribute != "customer.customer.customer_id" ||
				pattern.ContractQuery.Inputs[0].Input != "id" {
				t.Fatalf("共有される照会契約の参照が変わりました: %#v", pattern.ContractQuery)
			}
			db, ok := app.DatabaseByAliasExact(pattern.Datasource)
			if !ok || db.TargetDatabase != "master_data" || db.Access != database.AccessRead ||
				len(db.RequiredContracts) != 1 ||
				db.RequiredContracts[0].ContractID != "master_data.customer" ||
				db.RequiredContracts[0].CompatibleMajor != 2 {
				t.Fatalf("案件ごとの読み取り専用接続を解決できません: %#v, ok=%v", db, ok)
			}
			// An alias belongs to its application, even when the target contract is shared.
			otherAlias := pattern.Datasource
			if tc.datasource == "master" {
				otherAlias = "customer_master"
			} else {
				otherAlias = "master"
			}
			if _, ok := app.DatabaseByAliasExact(otherAlias); ok {
				t.Fatalf("別案件の接続名 %s を暗黙に取り込んでいます", otherAlias)
			}

			refs := idx.ReferencesFor(sourceindex.Subject{Kind: "application-pattern", ID: tc.pattern})
			if len(refs) != 1 || refs[0].Role != sourceindex.RoleDefinition || refs[0].ArtifactPath != tc.source {
				t.Fatalf("案件固有の出典が別の場所を指しています: %#v", refs)
			}
		})
	}

	// Both consumers must satisfy the existing signature and cross-layer checks.
	for _, check := range []struct {
		name string
		run  func() []validation.Diagnostic
	}{
		{"invocations", func() []validation.Diagnostic {
			return validation.ValidateReferenceContractQueryInvocations(loaded.World)
		}},
		{"cross-layer", func() []validation.Diagnostic {
			return validation.ValidateReferenceContractQueryCrossLayer(loaded.World)
		}},
	} {
		if diagnostics := check.run(); len(diagnostics) != 0 {
			t.Fatalf("%s: 共有契約の参照が不整合です: %#v", check.name, diagnostics)
		}
	}
}

// A Contract Query reused by another Application cannot authorize an alias
// absent from this Application. The failure must stay on the changed Pattern.
func TestReferenceCustomerAliasDoesNotBorrowAcrossApplications(t *testing.T) {
	cases := []struct {
		targetApp     string
		targetPattern string
		otherApp      string
		otherPattern  string
	}{
		{
			targetApp:     "order_entry",
			targetPattern: "order_entry.customer_reference",
			otherApp:      "repair_intake",
			otherPattern:  "repair_intake.customer_reference",
		},
		{
			targetApp:     "repair_intake",
			targetPattern: "repair_intake.customer_reference",
			otherApp:      "order_entry",
			otherPattern:  "order_entry.customer_reference",
		},
	}
	for _, tc := range cases {
		t.Run(tc.targetApp, func(t *testing.T) {
			loaded, err := workspace.Load(filepath.Join(repoRoot(t), "reference", "workspace"))
			if err != nil {
				t.Fatal(err)
			}
			if ds := validation.ValidateApplicationPatterns(loaded.World); len(ds) != 0 {
				t.Fatalf("元の参照Applicationが不整合です: %#v", ds)
			}

			other := referenceApplicationByID(t, loaded.World.Applications, tc.otherApp)
			foreignAlias := applicationPatternByID(t, other.Patterns, tc.otherPattern).Datasource
			if _, ok := other.DatabaseByAliasExact(foreignAlias); !ok {
				t.Fatalf("他方の接続名 %s は宣言されていません", foreignAlias)
			}

			found := false
			for appIndex := range loaded.World.Applications {
				app := &loaded.World.Applications[appIndex]
				if string(app.ID) != tc.targetApp {
					continue
				}
				if app.HasDatabaseAlias(foreignAlias) {
					t.Fatalf("試験の前提が崩れています: %s に %s が存在します", app.ID, foreignAlias)
				}
				for patternIndex := range app.Patterns {
					pattern := &app.Patterns[patternIndex]
					if string(pattern.ID) != tc.targetPattern {
						continue
					}
					pattern.Datasource = foreignAlias
					found = true
				}
			}
			if !found {
				t.Fatalf("変更するPattern %s が見つかりません", tc.targetPattern)
			}

			ds := validation.ValidateApplicationPatterns(loaded.World)
			if len(ds) != 1 || ds[0].Code != "UNKNOWN_APPLICATION_PATTERN_DATASOURCE" ||
				ds[0].Subject == nil || ds[0].Subject.Kind != "application-pattern" ||
				ds[0].Subject.ID != tc.targetPattern {
				t.Fatalf("他案件の接続名を借用して処理しています: %#v", ds)
			}
			// The other Application's original binding remains authoritative.
			other = referenceApplicationByID(t, loaded.World.Applications, tc.otherApp)
			if _, ok := other.DatabaseByAliasExact(foreignAlias); !ok {
				t.Fatalf("別案件の正規接続が変更されました: %s", tc.otherApp)
			}
		})
	}
}

func referenceCustomerApplication(t *testing.T, applications []application.Application) application.Application {
	t.Helper()
	for _, app := range applications {
		if app.ID == "order_entry" {
			return app
		}
	}
	t.Fatal("order_entry Applicationが見つかりません")
	return application.Application{}
}

func referenceCustomerPattern(t *testing.T, patterns []application.Pattern) application.Pattern {
	t.Helper()
	for _, pattern := range patterns {
		if pattern.ID == "order_entry.customer_reference" {
			return pattern
		}
	}
	t.Fatal("order_entry.customer_reference Patternが見つかりません")
	return application.Pattern{}
}

func referenceCustomerRelation(t *testing.T, namespaces []semantic.Namespace) semantic.Relation {
	t.Helper()
	for _, namespace := range namespaces {
		for _, data := range namespace.Data {
			for _, relation := range data.Relations {
				if relation.ID == "sales.order.customer" {
					return relation
				}
			}
		}
	}
	t.Fatal("sales.order.customer Relationが見つかりません")
	return semantic.Relation{}
}

func referenceCustomerContractQuery(t *testing.T, databases []database.Database) database.QueryContract {
	t.Helper()
	for _, db := range databases {
		if db.ID != "master_data" {
			continue
		}
		for _, contract := range db.Contracts {
			if contract.ID != "master_data.customer" {
				continue
			}
			for _, query := range contract.Queries {
				if query.Name == "find_by_id" {
					return query
				}
			}
		}
	}
	t.Fatal("master_data.customer / find_by_idが見つかりません")
	return database.QueryContract{}
}
