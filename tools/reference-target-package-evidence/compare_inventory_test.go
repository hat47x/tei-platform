package main

import "testing"

// A hand-written or corrupted comparison input must not manufacture package
// removal by adding impossible paths, duplicates, or arbitrary source kinds.
func TestCompareRejectsImpossibleSourceInventory(t *testing.T) {
	base := comparisonFixture("tei/cmd/one", SourcePackage{
		ImportPath: "tei/cmd/one",
		GoFiles:    []string{"main.go"},
		Imports:    []string{"fmt"},
		AuxiliaryInputs: []SourceInput{
			{Kind: "embed", Path: "assets/data.txt"},
		},
	})
	for _, tc := range []struct {
		name   string
		mutate func(*SourcePackage)
	}{
		{"duplicate-compiled-go", func(p *SourcePackage) { p.GoFiles = []string{"main.go", "main.go"} }},
		{"overlapping-go-cgo", func(p *SourcePackage) {
			p.CgoFiles = []string{"main.go"}
		}},
		{"traversal-go", func(p *SourcePackage) { p.GoFiles = []string{"../outside.go"} }},
		{"absolute-go", func(p *SourcePackage) { p.GoFiles = []string{"/tmp/main.go"} }},
		{"noncanonical-go", func(p *SourcePackage) { p.GoFiles = []string{"src/../main.go"} }},
		{"backslash-go", func(p *SourcePackage) { p.GoFiles = []string{`src\\main.go`} }},
		{"unknown-auxiliary-kind", func(p *SourcePackage) {
			p.AuxiliaryInputs = []SourceInput{{Kind: "opaque", Path: "assets/data.txt"}}
		}},
		{"duplicate-auxiliary-input", func(p *SourcePackage) {
			p.AuxiliaryInputs = []SourceInput{
				{Kind: "embed", Path: "assets/data.txt"},
				{Kind: "embed", Path: "assets/data.txt"},
			}
		}},
		{"traversal-embedded-file", func(p *SourcePackage) {
			p.AuxiliaryInputs = []SourceInput{{Kind: "embed", Path: "../secret.txt"}}
		}},
		{"noncanonical-embedded-file", func(p *SourcePackage) {
			p.AuxiliaryInputs = []SourceInput{{Kind: "embed", Path: "assets//data.txt"}}
		}},
		{"duplicate-import", func(p *SourcePackage) { p.Imports = []string{"fmt", "fmt"} }},
		{"unsafe-import", func(p *SourcePackage) { p.Imports = []string{"../fmt"} }},
		{"unsafe-package-path", func(p *SourcePackage) { p.ImportPath = "tei/../other" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			revised := base
			revised.Packages = append([]SourcePackage(nil), base.Packages...)
			tc.mutate(&revised.Packages[0])
			if _, err := compareReports(base, revised); err == nil {
				t.Fatal("impossible imported package inventory accepted")
			}
		})
	}
}

func TestCompareAcceptsSortedEquivalentPackageInventory(t *testing.T) {
	before := comparisonFixture("tei/cmd/one", SourcePackage{
		ImportPath: "tei/cmd/one",
		GoFiles:    []string{"z.go", "a.go"},
		Imports:    []string{"strings", "fmt"},
		AuxiliaryInputs: []SourceInput{
			{Kind: "embed", Path: "assets/z.txt"},
			{Kind: "embed", Path: "assets/a.txt"},
		},
	})
	after := comparisonFixture("tei/cmd/one", SourcePackage{
		ImportPath: "tei/cmd/one",
		GoFiles:    []string{"a.go", "z.go"},
		Imports:    []string{"fmt", "strings"},
		AuxiliaryInputs: []SourceInput{
			{Kind: "embed", Path: "assets/a.txt"},
			{Kind: "embed", Path: "assets/z.txt"},
		},
	})
	comparison, err := compareReports(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if len(comparison.BeforeOnly) != 0 || len(comparison.AfterOnly) != 0 || len(comparison.Changed) != 0 {
		t.Fatalf("enumeration order produced a false source change: %+v", comparison)
	}
}
