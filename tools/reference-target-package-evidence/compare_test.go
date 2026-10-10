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
		{"goamd64-drift", func(r *Report) { r.TargetFeatures = map[string]string{"GOAMD64": "v3"} }},
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

func TestCompareRejectsUnreachableOrDanglingMainModulePackages(t *testing.T) {
	base := comparisonFixture("tei/cmd/one",
		SourcePackage{ImportPath: "tei/cmd/one", GoFiles: []string{"main.go"}, Imports: []string{"tei/impl/used"}},
		SourcePackage{ImportPath: "tei/impl/used", GoFiles: []string{"used.go"}},
	)
	tests := []struct {
		name   string
		mutate func(*Report)
	}{
		{"unreachable-phantom", func(r *Report) {
			r.Packages = append(r.Packages, SourcePackage{ImportPath: "tei/impl/phantom", GoFiles: []string{"phantom.go"}})
		}},
		{"dangling-module-import", func(r *Report) {
			r.Packages[0].Imports = append(r.Packages[0].Imports, "tei/impl/missing")
		}},
		{"missing-transitive-node", func(r *Report) {
			r.Packages = r.Packages[:1]
		}},
		{"foreign-node", func(r *Report) {
			r.Packages = append(r.Packages, SourcePackage{ImportPath: "elsewhere/phantom", GoFiles: []string{"phantom.go"}})
		}},
		{"empty-source", func(r *Report) {
			r.Packages[1].GoFiles = nil
		}},
		{"invalid-cgo-pseudo-import", func(r *Report) {
			r.Packages[0].Imports = append(r.Packages[0].Imports, "C")
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := base
			r.Packages = append([]SourcePackage(nil), base.Packages...)
			tc.mutate(&r)
			if _, err := compareReports(base, r); err == nil {
				t.Fatal("structurally impossible source-level closure was accepted")
			}
		})
	}
	if _, err := compareReports(base, base); err != nil {
		t.Fatalf("valid reachable source closure rejected: %v", err)
	}
}

func TestCompareRejectsCgoCompilerConfigurationDrift(t *testing.T) {
	before := comparisonFixture("tei/cmd/cgo", SourcePackage{
		ImportPath: "tei/cmd/cgo", CgoFiles: []string{"main.go"}, Imports: []string{"C"},
	})
	before.CGOEnabled = "1"
	before.CgoToolchain = map[string]string{
		"CC": "gcc", "CXX": "g++", "CGO_CFLAGS": "-O2 -g",
		"CGO_CPPFLAGS": "", "CGO_CXXFLAGS": "-O2 -g",
		"CGO_LDFLAGS": "-O2 -g", "PKG_CONFIG": "pkg-config",
	}
	after := before
	// Clone the map so both reports retain distinct captured toolchains.
	after.CgoToolchain = make(map[string]string, len(before.CgoToolchain))
	for name, value := range before.CgoToolchain {
		after.CgoToolchain[name] = value
	}
	if _, err := compareReports(before, after); err != nil {
		t.Fatalf("identical cgo build profile rejected: %v", err)
	}
	for _, tc := range []struct {
		key, alternate string
	}{
		{"CC", "clang"}, {"CGO_CFLAGS", "-O0"}, {"CGO_LDFLAGS", "-static"},
		{"PKG_CONFIG", "another-pkg-config"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			original := after.CgoToolchain[tc.key]
			after.CgoToolchain[tc.key] = tc.alternate
			if _, err := compareReports(before, after); err == nil {
				t.Fatalf("%s drift accepted as identical Go build profile", tc.key)
			}
			after.CgoToolchain[tc.key] = original
		})
	}
}
