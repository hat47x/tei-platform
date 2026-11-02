package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func packageWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Data: []semantic.Data{{
					ID: "sales.order",
					Relations: []semantic.Relation{{
						ID:          "sales.order.customer",
						Target:      "customer.customer",
						Cardinality: semantic.CardinalityOne,
					}},
				}},
			},
			{
				ID: "customer",
				Data: []semantic.Data{{
					ID: "customer.customer",
				}},
			},
		},
		Applications: []application.Application{{
			ID:           "order_entry",
			SemanticUses: []core.CanonicalID{"sales", "customer"},
			Patterns: []application.Pattern{
				{ID: "order_entry.recent_orders", Kind: application.PatternBrowse, Data: "sales.order"},
				{ID: "order_entry.order_record", Kind: application.PatternRecord, Data: "sales.order"},
			},
			Packages: []application.Package{{
				ID:      "order_entry.list_to_record",
				Version: "1",
				Members: []application.PackageMember{
					{Kind: application.PatternBrowse, Pattern: "order_entry.recent_orders"},
					{Kind: application.PatternRecord, Pattern: "order_entry.order_record"},
				},
				Requires: []application.PackageRequirement{
					{Kind: "semantic", ID: "sales.order"},
					{Kind: "relation", ID: "sales.order.customer"},
				},
				Projection:  "screen",
				Conformance: "list_to_record",
			}},
			PackageBindings: []application.PackageBinding{{
				ID:      "order_entry.list_to_record",
				Version: "1",
			}},
		}},
	}
}

func packagePattern(w *ir.World) *application.Package {
	return &w.Applications[0].Packages[0]
}

func TestValidateApplicationPackage(t *testing.T) {
	w := packageWorld()
	for _, diagnostic := range validateApplicationPackages(w) {
		t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
	}
}

func TestValidateApplicationPackageRejectsUnknownMember(t *testing.T) {
	w := packageWorld()
	packagePattern(&w).Members[1].Pattern = "order_entry.missing"
	if diagnosticByCode(validateApplicationPackages(w), "UNKNOWN_APPLICATION_PACKAGE_MEMBER") == nil {
		t.Fatal("unknown member was accepted")
	}
}

func TestValidateApplicationPackageRejectsMemberKindMismatch(t *testing.T) {
	w := packageWorld()
	packagePattern(&w).Members[0].Kind = application.PatternRecord
	if diagnosticByCode(validateApplicationPackages(w), "APPLICATION_PACKAGE_MEMBER_KIND_MISMATCH") == nil {
		t.Fatal("member kind mismatch was accepted")
	}
}

func TestValidateApplicationPackageRejectsDuplicateMember(t *testing.T) {
	w := packageWorld()
	pkg := packagePattern(&w)
	pkg.Members = append(pkg.Members, pkg.Members[0])
	if diagnosticByCode(validateApplicationPackages(w), "APPLICATION_PACKAGE_MEMBER_DUPLICATE") == nil {
		t.Fatal("duplicate member was accepted")
	}
}

func TestValidateApplicationPackageRejectsUnusedRequirement(t *testing.T) {
	w := packageWorld()
	w.Applications[0].SemanticUses = []core.CanonicalID{"customer"}
	if diagnosticByCode(validateApplicationPackages(w), "APPLICATION_PACKAGE_REQUIREMENT_SEMANTIC_NOT_USED") == nil {
		t.Fatal("unused semantic requirement was accepted")
	}
}

func TestValidateApplicationPackageRejectsCrossPackageCollision(t *testing.T) {
	w := packageWorld()
	first := *packagePattern(&w)
	second := first
	second.ID = "order_entry.other_list"
	second.Members = []application.PackageMember{first.Members[0]}
	w.Applications[0].Packages = append(w.Applications[0].Packages, second)
	if diagnosticByCode(validateApplicationPackages(w), "APPLICATION_PACKAGE_MEMBER_COLLISION") == nil {
		t.Fatal("cross-package member collision was accepted")
	}
}

func TestValidateApplicationPackageRejectsDuplicatePackage(t *testing.T) {
	w := packageWorld()
	w.Applications[0].Packages = append(w.Applications[0].Packages, w.Applications[0].Packages[0])
	if diagnosticByCode(validateApplicationPackages(w), "APPLICATION_PACKAGE_DUPLICATE") == nil {
		t.Fatal("duplicate package was accepted")
	}
}

func TestValidateApplicationPackageBindingRejectsUnknownPackage(t *testing.T) {
	w := packageWorld()
	w.Applications[0].PackageBindings[0].ID = "order_entry.missing"
	if diagnosticByCode(validateApplicationPackages(w), "UNKNOWN_APPLICATION_PACKAGE_BINDING") == nil {
		t.Fatal("unknown package binding was accepted")
	}
}

func TestValidateApplicationPackageBindingRejectsVersionMismatch(t *testing.T) {
	w := packageWorld()
	w.Applications[0].PackageBindings[0].Version = "2"
	if diagnosticByCode(validateApplicationPackages(w), "APPLICATION_PACKAGE_BINDING_VERSION_MISMATCH") == nil {
		t.Fatal("version mismatch was accepted")
	}
}

func TestValidateApplicationPackageBindingRejectsDuplicate(t *testing.T) {
	w := packageWorld()
	w.Applications[0].PackageBindings = append(w.Applications[0].PackageBindings, w.Applications[0].PackageBindings[0])
	if diagnosticByCode(validateApplicationPackages(w), "APPLICATION_PACKAGE_BINDING_DUPLICATE") == nil {
		t.Fatal("duplicate package binding was accepted")
	}
}

func TestPackageRemovalDoesNotChangeSemanticFingerprint(t *testing.T) {
	w := packageWorld()
	before, err := semantic.Fingerprint(w.Semantics)
	if err != nil {
		t.Fatal(err)
	}
	w.Applications[0].Packages = nil
	w.Applications[0].PackageBindings = nil
	after, err := semantic.Fingerprint(w.Semantics)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("package removal changed the semantic fingerprint: %s != %s", before, after)
	}
}
