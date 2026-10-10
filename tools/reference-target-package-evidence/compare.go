package main

import (
	"fmt"
	"reflect"
	"sort"
)

// PackageChange describes changed compiled-source or import edges in one
// package shared by two static Go build entry closures.
type PackageChange struct {
	ImportPath string        `json:"importPath"`
	Before     SourcePackage `json:"before"`
	After      SourcePackage `json:"after"`
}

// ClosureComparison describes E0 source-level differences, not which symbols
// survive linking or whether a runtime capability has been safely removed.
type ClosureComparison struct {
	SourceRevision  string          `json:"sourceRevision"`
	Module          string          `json:"module"`
	GoVersion       string          `json:"goVersion"`
	GOOS            string          `json:"goos"`
	GOARCH          string          `json:"goarch"`
	CGOEnabled      string          `json:"cgoEnabled"`
	GoFlags         string          `json:"goFlags"`
	TargetFeatures  map[string]string `json:"targetFeatures,omitempty"`
	BeforeEntry     string          `json:"beforeEntry"`
	AfterEntry      string          `json:"afterEntry"`
	BeforeBuildTags string          `json:"beforeBuildTags,omitempty"`
	AfterBuildTags  string          `json:"afterBuildTags,omitempty"`
	BeforeOnly      []string        `json:"beforeOnly"`
	AfterOnly       []string        `json:"afterOnly"`
	Changed         []PackageChange `json:"changed"`
}

func packageMap(report Report) (map[string]SourcePackage, error) {
	if report.Entry == "" || report.Module == "" || report.SourceRevision == "" ||
		report.GoVersion == "" || report.GOOS == "" || report.GOARCH == "" || report.CGOEnabled == "" {
		return nil, fmt.Errorf("incomplete Go package report identity")
	}
	pkgs := make(map[string]SourcePackage, len(report.Packages))
	for _, p := range report.Packages {
		if p.ImportPath == "" {
			return nil, fmt.Errorf("empty package path")
		}
		if _, exists := pkgs[p.ImportPath]; exists {
			return nil, fmt.Errorf("duplicate package %q", p.ImportPath)
		}
		p.GoFiles = append([]string(nil), p.GoFiles...)
		p.CgoFiles = append([]string(nil), p.CgoFiles...)
		p.Imports = append([]string(nil), p.Imports...)
		p.AuxiliaryInputs = append([]SourceInput(nil), p.AuxiliaryInputs...)
		sort.Strings(p.GoFiles)
		sort.Strings(p.CgoFiles)
		sort.Strings(p.Imports)
		sort.Slice(p.AuxiliaryInputs, func(i, j int) bool {
			if p.AuxiliaryInputs[i].Kind != p.AuxiliaryInputs[j].Kind {
				return p.AuxiliaryInputs[i].Kind < p.AuxiliaryInputs[j].Kind
			}
			return p.AuxiliaryInputs[i].Path < p.AuxiliaryInputs[j].Path
		})
		pkgs[p.ImportPath] = p
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("empty Go package closure")
	}
	if _, ok := pkgs[report.Entry]; !ok {
		return nil, fmt.Errorf("entry %q absent from package closure", report.Entry)
	}
	return pkgs, nil
}

// compareReports requires the same source revision and Go toolchain profile.
// Explicit build tags may differ and remain visible in the comparison; a
// change in build tags is not itself evidence of a minimal binary.
func compareReports(before, after Report) (ClosureComparison, error) {
	a, err := packageMap(before)
	if err != nil {
		return ClosureComparison{}, fmt.Errorf("before: %w", err)
	}
	b, err := packageMap(after)
	if err != nil {
		return ClosureComparison{}, fmt.Errorf("after: %w", err)
	}
	if before.SourceRevision != after.SourceRevision || before.Module != after.Module ||
		before.GoVersion != after.GoVersion || before.GOOS != after.GOOS ||
		before.GOARCH != after.GOARCH || before.CGOEnabled != after.CGOEnabled ||
		before.GoFlags != after.GoFlags || !reflect.DeepEqual(before.TargetFeatures, after.TargetFeatures) {
		return ClosureComparison{}, fmt.Errorf("source revision, module, or Go build environment differs")
	}
	out := ClosureComparison{
		SourceRevision: before.SourceRevision, Module: before.Module,
		GoVersion: before.GoVersion, GOOS: before.GOOS, GOARCH: before.GOARCH,
		CGOEnabled: before.CGOEnabled, GoFlags: before.GoFlags,
		TargetFeatures: before.TargetFeatures,
		BeforeEntry: before.Entry, AfterEntry: after.Entry,
		BeforeBuildTags: before.BuildTags, AfterBuildTags: after.BuildTags,
		BeforeOnly: []string{}, AfterOnly: []string{}, Changed: []PackageChange{},
	}
	for name, original := range a {
		revised, exists := b[name]
		if !exists {
			out.BeforeOnly = append(out.BeforeOnly, name)
			continue
		}
		if !reflect.DeepEqual(original, revised) {
			out.Changed = append(out.Changed, PackageChange{ImportPath: name, Before: original, After: revised})
		}
	}
	for name := range b {
		if _, exists := a[name]; !exists {
			out.AfterOnly = append(out.AfterOnly, name)
		}
	}
	sort.Strings(out.BeforeOnly)
	sort.Strings(out.AfterOnly)
	sort.Slice(out.Changed, func(i, j int) bool { return out.Changed[i].ImportPath < out.Changed[j].ImportPath })
	return out, nil
}
