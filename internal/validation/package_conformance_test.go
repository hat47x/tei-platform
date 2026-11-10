package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
)

// packageConformanceWorld builds the fixture with one package carrying the fields
// under test.
func packageConformanceWorld(conformance, projection string) ir.World {
	world := validApplicationPatternWorld()
	world.Applications[0].Packages = []application.Package{{
		ID:          "order_entry.search_package",
		Version:     "1",
		Members:     []application.PackageMember{{Kind: application.PatternBrowse, Pattern: "order_entry.recent_orders"}},
		Projection:  projection,
		Conformance: conformance,
	}}
	return world
}

func conformanceCodes(w ir.World) []string {
	var out []string
	for _, diagnostic := range validatePackageConformance(w) {
		out = append(out, diagnostic.Code)
	}
	return out
}

// TestPackageConformanceRequiresBothFields checks that a package must name the
// conformance suite it satisfies and the projection it produces. Both fields were
// parsed and stored but never checked, so a package could claim anything.
func TestPackageConformanceRequiresBothFields(t *testing.T) {
	if codes := conformanceCodes(packageConformanceWorld("", "")); len(codes) != 2 {
		t.Fatalf("codes=%v", codes)
	}
	if codes := conformanceCodes(packageConformanceWorld("search-v1", "screen")); len(codes) != 0 {
		t.Fatalf("a complete declaration was refused: %v", codes)
	}
	if codes := conformanceCodes(packageConformanceWorld("search-v1", "")); len(codes) != 1 || codes[0] != "APPLICATION_PACKAGE_PROJECTION_REQUIRED" {
		t.Fatalf("codes=%v", codes)
	}
	if codes := conformanceCodes(packageConformanceWorld("", "screen")); len(codes) != 1 || codes[0] != "APPLICATION_PACKAGE_CONFORMANCE_REQUIRED" {
		t.Fatalf("codes=%v", codes)
	}
}

// TestPackageConformanceRefusesWhitespace checks that a suite name cannot be a
// sentence: the name is a contract identifier, and whitespace means it was not
// written as one.
func TestPackageConformanceRefusesWhitespace(t *testing.T) {
	codes := conformanceCodes(packageConformanceWorld("search v1", "screen"))
	if len(codes) != 1 || codes[0] != "APPLICATION_PACKAGE_CONFORMANCE_INVALID" {
		t.Fatalf("codes=%v", codes)
	}
}

// TestPackageConformanceComparesSharedKinds checks that two packages claiming the
// same pattern kind must name the same suite, because RQ-C1 asks for a comparison
// under one conformance framework.
func TestPackageConformanceComparesSharedKinds(t *testing.T) {
	world := validApplicationPatternWorld()
	world.Applications[0].Packages = []application.Package{
		{
			ID:          "order_entry.search_package",
			Version:     "1",
			Members:     []application.PackageMember{{Kind: application.PatternBrowse, Pattern: "order_entry.recent_orders"}},
			Projection:  "screen",
			Conformance: "screen-v1",
		},
		{
			ID:          "order_entry.other_package",
			Version:     "1",
			Members:     []application.PackageMember{{Kind: application.PatternBrowse, Pattern: "order_entry.recent_orders"}},
			Projection:  "print",
			Conformance: "print-v1",
		},
	}
	var found bool
	for _, diagnostic := range validatePackageConformance(world) {
		if diagnostic.Code == "APPLICATION_PACKAGE_CONFORMANCE_MISMATCH" {
			found = true
		}
	}
	if !found {
		t.Fatal("two packages claiming one kind with different suites were accepted")
	}

	// The same suite is accepted: the two are then comparable.
	same := world
	same.Applications[0].Packages[1].Conformance = "screen-v1"
	for _, diagnostic := range validatePackageConformance(same) {
		if diagnostic.Code == "APPLICATION_PACKAGE_CONFORMANCE_MISMATCH" {
			t.Fatalf("a shared suite was refused: %s", diagnostic.Message)
		}
	}
}

// TestPackageConformanceLeavesAnUnrelatedKindAlone checks that two packages claiming
// different kinds may name different suites: there is nothing to compare.
func TestPackageConformanceLeavesAnUnrelatedKindAlone(t *testing.T) {
	world := validApplicationPatternWorld()
	world.Applications[0].Patterns = append(world.Applications[0].Patterns, application.Pattern{
		ID:   "order_entry.order_record",
		Kind: application.PatternRecord,
		Data: "sales.order",
	})
	world.Applications[0].Packages = []application.Package{
		{
			ID:          "order_entry.search_package",
			Version:     "1",
			Members:     []application.PackageMember{{Kind: application.PatternBrowse, Pattern: "order_entry.recent_orders"}},
			Projection:  "screen",
			Conformance: "screen-v1",
		},
		{
			ID:          "order_entry.record_package",
			Version:     "1",
			Members:     []application.PackageMember{{Kind: application.PatternRecord, Pattern: "order_entry.order_record"}},
			Projection:  "form",
			Conformance: "form-v1",
		},
	}
	for _, diagnostic := range validatePackageConformance(world) {
		if diagnostic.Code == "APPLICATION_PACKAGE_CONFORMANCE_MISMATCH" {
			t.Fatalf("different kinds were compared: %s", diagnostic.Message)
		}
	}
}
