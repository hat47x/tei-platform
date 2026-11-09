package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const jsonA = `{"ImportPath":"tei/tools/sample","Name":"main","Module":{"Path":"tei","Main":true},"GoFiles":["z.go","a.go"],"Imports":["fmt","tei/internal/a"]}`
const jsonB = `{"ImportPath":"tei/internal/a","Module":{"Path":"tei","Main":true},"GoFiles":["x.go"],"Imports":["strings"]}`
const jsonStd = `{"ImportPath":"fmt","GoFiles":["print.go"]}`
const jsonStrings = `{"ImportPath":"strings","GoFiles":["strings.go"]}`

func TestDeterministicSourceClosure(t *testing.T) {
	a, err := parseGoList(strings.NewReader(strings.Join([]string{jsonA, jsonB, jsonStd, jsonStrings}, "\n")), "tei/tools/sample", "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := parseGoList(strings.NewReader(strings.Join([]string{jsonStrings, jsonStd, jsonB, jsonA}, "\n")), "tei/tools/sample", "")
	if err != nil {
		t.Fatal(err)
	}
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	if string(aj) != string(bj) {
		t.Fatalf("unstable output: %s vs %s", aj, bj)
	}
	if len(a.Packages) != 2 || a.Packages[0].ImportPath != "tei/internal/a" || a.Packages[1].GoFiles[0] != "a.go" {
		t.Fatalf("wrong closure: %+v", a)
	}
}

func TestMissingOrMalformedSourceFailsClosed(t *testing.T) {
	for _, scenario := range []struct {
		name, raw, entry string
	}{
		{"missing-entry", jsonB + "\n" + jsonStrings, "tei/tools/sample"},
		{"truncated-json", jsonA[:len(jsonA)-1], "tei/tools/sample"},
		{"missing-dependency", jsonA + "\n" + jsonStd, "tei/tools/sample"},
		{"duplicate", jsonA + "\n" + jsonA + "\n" + jsonB + "\n" + jsonStd + "\n" + jsonStrings, "tei/tools/sample"},
		{"package-error", `{"ImportPath":"tei/tools/sample","Name":"main","Module":{"Path":"tei","Main":true},"Error":{"Err":"broken"}}`, "tei/tools/sample"},
		{"empty-package-error", `{"ImportPath":"tei/tools/sample","Name":"main","Module":{"Path":"tei","Main":true},"Error":{}}`, "tei/tools/sample"},
		{"incomplete", `{"ImportPath":"tei/tools/sample","Name":"main","Module":{"Path":"tei","Main":true},"Incomplete":true}`, "tei/tools/sample"},
		{"non-executable-entry", `{"ImportPath":"tei/tools/sample","Name":"library","Module":{"Path":"tei","Main":true},"GoFiles":["lib.go"]}`, "tei/tools/sample"},
		{"entry-not-main-module", jsonStd + "\n" + jsonB + "\n" + jsonStrings, "fmt"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			if _, err := parseGoList(strings.NewReader(scenario.raw), scenario.entry, ""); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestEntryRejectsNonLiteralAndTraversal(t *testing.T) {
	for _, s := range []string{"tei/tools/app", "../app", "./tools/../app", "./tools/...", "./tools/*", "./tools/a...b", "./tools/foo...bar", "./tools//app", "./", "./tools/app "} {
		if validEntry(s) {
			t.Fatalf("unexpected accepted entry %q", s)
		}
	}
	if !validEntry("./tools/reference-first-web-target") {
		t.Fatal("valid entry rejected")
	}
}

func TestCgoPseudoImportIsNotAListedGoPackage(t *testing.T) {
	// Real go list -deps -json includes "C" in Imports for a cgo package,
	// but does not emit a package object with ImportPath "C".
	cgo := `{"ImportPath":"tei/tools/cgo-sample","Name":"main","Module":{"Path":"tei","Main":true},"CgoFiles":["main.go"],"Imports":["C","fmt"]}`
	got, err := parseGoList(strings.NewReader(cgo+"\n"+jsonStd), "tei/tools/cgo-sample", "")
	if err != nil {
		t.Fatalf("valid cgo dependency graph rejected: %v", err)
	}
	if len(got.Packages) != 1 || len(got.Packages[0].Imports) != 2 || got.Packages[0].Imports[0] != "C" {
		t.Fatalf("lost cgo source or import edge: %+v", got)
	}

	nonCgo := `{"ImportPath":"tei/tools/cgo-sample","Name":"main","Module":{"Path":"tei","Main":true},"GoFiles":["main.go"],"Imports":["C","fmt"]}`
	if _, err := parseGoList(strings.NewReader(nonCgo+"\n"+jsonStd), "tei/tools/cgo-sample", ""); err == nil {
		t.Fatal("pseudo import C must not excuse a missing Go dependency for non-cgo sources")
	}
}

func TestTrackedCompiledSourceInventory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module tei\n"), 0600); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(root, "tools", "sample")
	if err := os.MkdirAll(pkgDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(pkgDir, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.go", "native.go", "native.h", "assets/x.txt"} {
		if err := os.WriteFile(filepath.Join(pkgDir, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	base := listedPackage{
		ImportPath: "tei/tools/sample",
		Dir:        pkgDir,
		Module:     &listedModule{Path: "tei", Main: true, Dir: root, GoMod: filepath.Join(root, "go.mod")},
		GoFiles:    []string{"main.go"},
		CgoFiles:   []string{"native.go"},
		HFiles:     []string{"native.h"},
		EmbedFiles: []string{"assets/x.txt"},
	}
	tracked := map[string]struct{}{
		"go.mod":                    {},
		"tools/sample/main.go":      {},
		"tools/sample/native.go":    {},
		"tools/sample/native.h":     {},
		"tools/sample/assets/x.txt": {},
	}
	for _, tc := range []struct {
		name    string
		prepare func(*listedPackage)
		wantErr bool
	}{
		{name: "tracked-go-and-cgo", prepare: func(*listedPackage) {}},
		{name: "ignored-go-source", prepare: func(p *listedPackage) { p.GoFiles = append(p.GoFiles, "ignored.go") }, wantErr: true},
		{name: "ignored-cgo-source", prepare: func(p *listedPackage) { p.CgoFiles = append(p.CgoFiles, "ignored.go") }, wantErr: true},
		{name: "ignored-header", prepare: func(p *listedPackage) { p.HFiles = []string{"ignored.h"} }, wantErr: true},
		{name: "ignored-embedded-asset", prepare: func(p *listedPackage) { p.EmbedFiles = append(p.EmbedFiles, "assets/ignored.txt") }, wantErr: true},
		{name: "traversal-embedded-asset", prepare: func(p *listedPackage) { p.EmbedFiles = []string{"../secret.txt"} }, wantErr: true},
		{name: "missing-directory", prepare: func(p *listedPackage) { p.Dir = "" }, wantErr: true},
		{name: "wrong-main-module-directory", prepare: func(p *listedPackage) {
			m := *p.Module
			m.Dir = filepath.Join(root, "tools")
			p.Module = &m
		}, wantErr: true},
		{name: "wrong-main-module-modfile", prepare: func(p *listedPackage) {
			m := *p.Module
			m.GoMod = filepath.Join(root, "wrong.mod")
			p.Module = &m
		}, wantErr: true},
		{name: "out-of-root-directory", prepare: func(p *listedPackage) { p.Dir = filepath.Join(root, "..", "outside") }, wantErr: true},
		{name: "traversal-file", prepare: func(p *listedPackage) { p.GoFiles = []string{"../escape.go"} }, wantErr: true},
		{name: "absolute-file", prepare: func(p *listedPackage) { p.GoFiles = []string{filepath.Join(root, "elsewhere.go")} }, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			tc.prepare(&p)
			raw, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			err = validateTrackedSources(raw, root, tracked)
			if (err != nil) != tc.wantErr {
				t.Fatalf("unexpected error result: %v", err)
			}
		})
	}
}

func TestTrackedCompiledSymlinkCannotSupplyUntrackedBytes(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module tei\n"), 0600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "module")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	realSource := filepath.Join(root, "real.go")
	if err := os.WriteFile(realSource, []byte("package main\n"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.go")
	if err := os.Symlink(realSource, link); err != nil {
		t.Skipf("cannot create symlink on this host: %v", err)
	}
	p := listedPackage{
		ImportPath: "tei/module",
		Dir:        dir,
		Module:     &listedModule{Path: "tei", Main: true, Dir: root, GoMod: filepath.Join(root, "go.mod")},
		GoFiles:    []string{"link.go"},
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	tracked := map[string]struct{}{"go.mod": {}, "module/link.go": {}}
	if err := validateTrackedSources(raw, root, tracked); err == nil {
		t.Fatal("Git-tracked symlink to untracked bytes was accepted")
	}
}

func TestAuxiliaryPackageInputsAreRecordedInStableOrder(t *testing.T) {
	base := listedPackage{
		ImportPath: "tei/module",
		Name:       "main",
		Module:     &listedModule{Path: "tei", Main: true},
		GoFiles:    []string{"main.go"},
		EmbedFiles: []string{"assets/z.txt", "assets/a.txt"},
		HFiles:     []string{"native.h"},
		CFiles:     []string{"native.c"},
		Imports:    []string{"fmt"},
	}
	obj, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	report, err := parseGoList(strings.NewReader(string(obj)+"\n"+jsonStd), "tei/module", "")
	if err != nil {
		t.Fatal(err)
	}
	inputs := report.Packages[0].AuxiliaryInputs
	if len(inputs) != 4 || inputs[0].Kind != "c" || inputs[0].Path != "native.c" ||
		inputs[1].Kind != "embed" || inputs[1].Path != "assets/a.txt" ||
		inputs[2].Kind != "embed" || inputs[2].Path != "assets/z.txt" ||
		inputs[3].Kind != "header" || inputs[3].Path != "native.h" {
		t.Fatalf("unstable or missing native/embed source inventory: %+v", inputs)
	}
}
