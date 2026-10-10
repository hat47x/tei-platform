package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func comparisonFixture(entry string, packages ...SourcePackage) Report {
	return Report{
		Entry: entry, SourceRevision: "same-verified-git-head", Module: "tei",
		GoVersion: "go1.23.2", GOOS: "linux", GOARCH: "amd64",
		CGOEnabled: "0", GoFlags: "", Packages: packages,
	}
}

func TestCompareDistinctCompileTimeEntries(t *testing.T) {
	all := comparisonFixture("tei/cmd/all",
		SourcePackage{ImportPath: "tei/cmd/all", GoFiles: []string{"main.go"}, Imports: []string{"tei/impl/repair", "tei/impl/order"}},
		SourcePackage{ImportPath: "tei/impl/order", GoFiles: []string{"z.go", "a.go"}},
		SourcePackage{ImportPath: "tei/impl/repair", GoFiles: []string{"repair.go"}},
	)
	order := comparisonFixture("tei/cmd/order",
		SourcePackage{ImportPath: "tei/cmd/order", GoFiles: []string{"main.go"}, Imports: []string{"tei/impl/order"}},
		SourcePackage{ImportPath: "tei/impl/order", GoFiles: []string{"a.go", "z.go"}},
	)
	got, err := compareReports(all, order)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.BeforeOnly, []string{"tei/cmd/all", "tei/impl/repair"}) ||
		!reflect.DeepEqual(got.AfterOnly, []string{"tei/cmd/order"}) || len(got.Changed) != 0 {
		t.Fatalf("unexpected Go source closure difference: %+v", got)
	}
	// Changing report enumeration order and source file order does not change
	// the comparison, since physical source membership is a set.
	all.Packages[1], all.Packages[2] = all.Packages[2], all.Packages[1]
	order.Packages[1].GoFiles = []string{"z.go", "a.go"}
	reordered, err := compareReports(all, order)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(got)
	b, _ := json.Marshal(reordered)
	if string(a) != string(b) {
		t.Fatalf("ordering affected Go dependency comparison: %s vs %s", a, b)
	}
}

func TestCompareRejectsMixedSourceRevisionAndTargetProfile(t *testing.T) {
	before := comparisonFixture("tei/cmd/all", SourcePackage{ImportPath: "tei/cmd/all", GoFiles: []string{"main.go"}})
	after := before
	tests := []struct {
		name   string
		mutate func(*Report)
	}{
		{"git-source-drift", func(r *Report) { r.SourceRevision = "other-revision" }},
		{"go-version-drift", func(r *Report) { r.GoVersion = "go1.24.0" }},
		{"goos-drift", func(r *Report) { r.GOOS = "windows" }},
		{"goarch-drift", func(r *Report) { r.GOARCH = "arm64" }},
		{"cgo-drift", func(r *Report) { r.CGOEnabled = "1" }},
		{"goflags-drift", func(r *Report) { r.GoFlags = "-race" }},
		{"module-drift", func(r *Report) { r.Module = "other" }},
		{"empty-entry", func(r *Report) { r.Entry = "" }},
		{"duplicate-package", func(r *Report) { r.Packages = append(r.Packages, r.Packages[0]) }},
		{"entry-not-reported", func(r *Report) { r.Entry = "tei/cmd/missing" }},
		{"empty-closure", func(r *Report) { r.Packages = nil }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			changed := after
			tc.mutate(&changed)
			if _, err := compareReports(before, changed); err == nil {
				t.Fatal("incompatible or incomplete Go package report was accepted")
			}
		})
	}
}

func TestCompareBuildTagsAndEmbeddedInputsWithinOnePackage(t *testing.T) {
	before := comparisonFixture("tei/cmd/one", SourcePackage{
		ImportPath: "tei/cmd/one", GoFiles: []string{"main.go"}, Imports: []string{"fmt"},
	})
	after := comparisonFixture("tei/cmd/one", SourcePackage{
		ImportPath: "tei/cmd/one", GoFiles: []string{"with_tag.go", "main.go"},
		AuxiliaryInputs: []SourceInput{{Kind: "embed", Path: "assets/data.txt"}}, Imports: []string{"fmt"},
	})
	after.BuildTags = "optional"
	got, err := compareReports(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if got.BeforeBuildTags != "" || got.AfterBuildTags != "optional" ||
		len(got.BeforeOnly) != 0 || len(got.AfterOnly) != 0 ||
		len(got.Changed) != 1 || got.Changed[0].ImportPath != "tei/cmd/one" ||
		len(got.Changed[0].After.AuxiliaryInputs) != 1 {
		t.Fatalf("lost shared package file/embedded-source delta: %+v", got)
	}
}
