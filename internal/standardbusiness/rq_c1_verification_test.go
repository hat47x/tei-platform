package standardbusiness_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
	"tei/internal/validation"
)

// RQ-C1の「確認方法」を一箇所で固定する試験です。
//
// 要求文は次の3つを求めています。
//
//  1. 二つの独立したパッケージ（検索一覧、親子保守）を同じCoreの意味に対して
//     定義し、同じ適合試験の枠組みで比較する。
//  2. パッケージを取り外したときに、受理済みの意味と他の投影が壊れないこと。
//  3. パッケージの追加がCoreの要素種別の追加を要求しないこと。
//
// ここでは「Coreの要素種別」を`semantic.Namespace`の要素の種類（Data・Relation・
// Query・Capability・Type・Calendar・Period）と読み、パッケージが既存のPatternを
// 束ねるだけで新しい種別を足さないことを確かめます。

func rqPackageWorld(packages []application.Package, patterns []application.Pattern, bindings []application.PackageBinding) ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			// The standard prelude, so the fixture's attribute types resolve. The
			// loader supplies this in a real run.
			{ID: "std", Types: []semantic.Type{
				{ID: "std.identifier", Symbol: "identifier", Kind: semantic.TypePrimitive},
				{ID: "std.integer", Symbol: "integer", Kind: semantic.TypePrimitive},
			}},
			{
				ID: "sales",
				Data: []semantic.Data{
					{
						ID:         "sales.order",
						Attributes: []semantic.Attribute{{ID: "sales.order.order_id", TypeRef: "std.identifier"}},
						Relations: []semantic.Relation{{
							ID:          "sales.order.lines",
							Symbol:      "lines",
							Target:      "sales.line",
							Cardinality: semantic.CardinalityMany,
						}},
					},
					{ID: "sales.line", Attributes: []semantic.Attribute{{ID: "sales.line.line_no", TypeRef: "std.integer"}}},
				},
			},
		},
		Queries:   []query.EmbeddedSQL{rqListQuery()},
		Databases: []database.Database{{ID: "sales_core", SemanticUses: []core.CanonicalID{"sales"}, Kind: "relational", Ownership: database.OwnershipManagedHere}},
		Applications: []application.Application{{
			ID:              "order_entry",
			SemanticUses:    []core.CanonicalID{"sales"},
			Databases:       []application.DatabaseRequirement{{Alias: "sales", TargetDatabase: "sales_core", Access: database.AccessReadWrite}},
			Patterns:        patterns,
			Packages:        packages,
			PackageBindings: bindings,
		}},
	}
}

func rqListPattern() application.Pattern {
	return application.Pattern{
		ID:    "order_entry.order_list",
		Kind:  application.PatternBrowse,
		Data:  "sales.order",
		Query: "order_entry.order_list_query",
	}
}

// rqListQuery is the query the browse pattern names. A browse pattern requires a
// query, so the fixture carries one; the package tests are about the package, not
// the query.
func rqListQuery() query.EmbeddedSQL {
	return query.EmbeddedSQL{
		ID:          "order_entry.order_list_query",
		Application: "order_entry",
		Datasource:  "sales",
		Dialect:     environment.ProviderPostgres,
		Source:      "SELECT order_id FROM sales_order",
	}
}

// rqHeaderDetailPattern names only the relation: a header-detail pattern does not
// also name the data, because the relation already identifies the header and the
// detail.
func rqHeaderDetailPattern() application.Pattern {
	return application.Pattern{
		ID:       "order_entry.order_lines",
		Kind:     application.PatternHeaderDetail,
		Relation: "sales.order.lines",
	}
}

// TestRQC1TwoPackagesShareTheSameCoreMeaning checks 確認方法1: two independent
// packages are defined over the same accepted meaning, and both are checked by the
// same validation.
func TestRQC1TwoPackagesShareTheSameCoreMeaning(t *testing.T) {
	packages := []application.Package{
		{
			ID:          "order_entry.search_package",
			Version:     "1",
			Members:     []application.PackageMember{{Kind: application.PatternBrowse, Pattern: "order_entry.order_list"}},
			Requires:    []application.PackageRequirement{{Kind: "semantic", ID: "sales.order"}},
			Projection:  "list",
			Conformance: "search-v1",
		},
		{
			ID:          "order_entry.header_detail_package",
			Version:     "1",
			Members:     []application.PackageMember{{Kind: application.PatternHeaderDetail, Pattern: "order_entry.order_lines"}},
			Requires:    []application.PackageRequirement{{Kind: "semantic", ID: "sales.order"}},
			Projection:  "header-detail",
			Conformance: "header-detail-v1",
		},
	}
	world := rqPackageWorld(packages,
		[]application.Pattern{rqListPattern(), rqHeaderDetailPattern()},
		[]application.PackageBinding{
			{ID: "order_entry.search_package", Version: "1"},
			{ID: "order_entry.header_detail_package", Version: "1"},
		})

	// Both packages pass the same validation over the same meaning.
	for _, diagnostic := range validation.Validate(world) {
		if diagnostic.Severity == validation.Error {
			t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
		}
	}
	// They are two different types over one meaning, which is what makes the type a
	// package rather than a Core kind.
	if packages[0].Members[0].Kind == packages[1].Members[0].Kind {
		t.Fatalf("the two packages declare the same kind: %s", packages[0].Members[0].Kind)
	}
}

// TestRQC1RemovingAPackageKeepsTheMeaning checks 確認方法2: removing a package leaves
// the accepted meaning and the other projection intact.
func TestRQC1RemovingAPackageKeepsTheMeaning(t *testing.T) {
	list := rqListPattern()
	headerDetail := rqHeaderDetailPattern()
	withBoth := rqPackageWorld([]application.Package{
		{
			ID:          "order_entry.search_package",
			Version:     "1",
			Members:     []application.PackageMember{{Kind: application.PatternBrowse, Pattern: "order_entry.order_list"}},
			Projection:  "screen",
			Conformance: "search-v1",
		},
		{
			ID:      "order_entry.header_detail_package",
			Version: "1",
			Members: []application.PackageMember{{Kind: application.PatternHeaderDetail, Pattern: "order_entry.order_lines"}},
		},
	}, []application.Pattern{list, headerDetail}, nil)

	// The same world with the header-detail package removed: the meaning is the
	// same, and only one projection remains.
	withoutOne := rqPackageWorld([]application.Package{
		{
			ID:          "order_entry.search_package",
			Version:     "1",
			Members:     []application.PackageMember{{Kind: application.PatternBrowse, Pattern: "order_entry.order_list"}},
			Projection:  "screen",
			Conformance: "search-v1",
		},
	}, []application.Pattern{list}, nil)

	if len(withBoth.Semantics) != len(withoutOne.Semantics) {
		t.Fatalf("the meaning changed: %d then %d namespaces", len(withBoth.Semantics), len(withoutOne.Semantics))
	}
	if len(withBoth.Semantics[0].Data) != len(withoutOne.Semantics[0].Data) {
		t.Fatalf("the meaning changed: %d then %d data", len(withBoth.Semantics[0].Data), len(withoutOne.Semantics[0].Data))
	}
	for _, diagnostic := range validation.Validate(withoutOne) {
		if diagnostic.Severity == validation.Error {
			t.Fatalf("removing a package broke the rest: %s %s", diagnostic.Code, diagnostic.Message)
		}
	}
	// A package that still refers to the removed pattern is reported, so the removal
	// is not silent.
	stale := rqPackageWorld([]application.Package{
		{
			ID:          "order_entry.search_package",
			Version:     "1",
			Members:     []application.PackageMember{{Kind: application.PatternBrowse, Pattern: "order_entry.order_list"}},
			Projection:  "screen",
			Conformance: "search-v1",
		},
		{
			ID:      "order_entry.header_detail_package",
			Version: "1",
			Members: []application.PackageMember{{Kind: application.PatternHeaderDetail, Pattern: "order_entry.order_lines"}},
		},
	}, []application.Pattern{list}, nil)
	var reported bool
	for _, diagnostic := range validation.Validate(stale) {
		if diagnostic.Code == "UNKNOWN_APPLICATION_PACKAGE_MEMBER" {
			reported = true
		}
	}
	if !reported {
		t.Fatal("a package referring to a removed pattern was accepted")
	}
}

// TestRQC1AddingAPackageAddsNoCoreKind checks 確認方法3: the difference a package makes
// is inside the application, and no new semantic element kind appears.
//
// The check is structural: the package is declared in the application, its members
// are existing patterns, and the namespace's element kinds are unchanged. A change
// that added a Core kind would show as a new field or a new element kind here.
func TestRQC1AddingAPackageAddsNoCoreKind(t *testing.T) {
	list := rqListPattern()
	withoutPackage := rqPackageWorld(nil, []application.Pattern{list}, nil)
	withPackage := rqPackageWorld([]application.Package{{
		ID:          "order_entry.search_package",
		Version:     "1",
		Members:     []application.PackageMember{{Kind: application.PatternBrowse, Pattern: "order_entry.order_list"}},
		Projection:  "screen",
		Conformance: "search-v1",
	}}, []application.Pattern{list}, nil)

	before, after := withoutPackage.Semantics[0], withPackage.Semantics[0]
	// No element of the accepted meaning changed: the package lives in the
	// application, not in the semantic namespace.
	if len(before.Data) != len(after.Data) ||
		len(before.Capabilities) != len(after.Capabilities) ||
		len(before.Types) != len(after.Types) ||
		len(before.Calendars) != len(after.Calendars) ||
		len(before.Periods) != len(after.Periods) ||
		len(before.Validations) != len(after.Validations) ||
		len(before.EffectRules) != len(after.EffectRules) ||
		len(before.RunPlans) != len(after.RunPlans) {
		t.Fatalf("adding a package changed the semantic namespace: %+v -> %+v", before, after)
	}
	// The pattern itself is unchanged, so the package reused it rather than
	// replacing it with a new kind.
	if withPackage.Applications[0].Patterns[0].Kind != withoutPackage.Applications[0].Patterns[0].Kind {
		t.Fatal("adding a package changed the pattern kind")
	}
	// The only difference is the package declaration.
	if len(withoutPackage.Applications[0].Packages) != 0 || len(withPackage.Applications[0].Packages) != 1 {
		t.Fatalf("packages: %d then %d", len(withoutPackage.Applications[0].Packages), len(withPackage.Applications[0].Packages))
	}
	for _, diagnostic := range validation.Validate(withPackage) {
		if diagnostic.Severity == validation.Error {
			t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
		}
	}
}

// TestRQC1PatternCollisionIsReported checks the requirement's last clause: when two
// packages claim the same pattern, the collision is reported rather than resolved by
// declaration order.
func TestRQC1PatternCollisionIsReported(t *testing.T) {
	world := rqPackageWorld([]application.Package{
		{
			ID:          "order_entry.search_package",
			Version:     "1",
			Members:     []application.PackageMember{{Kind: application.PatternBrowse, Pattern: "order_entry.order_list"}},
			Projection:  "screen",
			Conformance: "search-v1",
		},
		{
			ID:      "order_entry.other_package",
			Version: "1",
			Members: []application.PackageMember{{Kind: application.PatternBrowse, Pattern: "order_entry.order_list"}},
		},
	}, []application.Pattern{rqListPattern()}, nil)
	var reported bool
	for _, diagnostic := range validation.Validate(world) {
		if diagnostic.Code == "APPLICATION_PACKAGE_MEMBER_COLLISION" {
			reported = true
		}
	}
	if !reported {
		t.Fatal("a pattern claimed by two packages was accepted")
	}
}
