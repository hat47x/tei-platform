package main

import (
	"encoding/json"
	"strings"
	"testing"
)

const jsonA = `{"ImportPath":"tei/tools/sample","Module":{"Path":"tei","Main":true},"GoFiles":["z.go","a.go"],"Imports":["fmt","tei/internal/a"]}`
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
		{"package-error", `{"ImportPath":"tei/tools/sample","Module":{"Path":"tei","Main":true},"Error":{"Err":"broken"}}`, "tei/tools/sample"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			if _, err := parseGoList(strings.NewReader(scenario.raw), scenario.entry, ""); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestEntryRejectsNonLiteralAndTraversal(t *testing.T) {
	for _, s := range []string{"tei/tools/app", "../app", "./tools/../app", "./tools/...", "./tools/*", "./tools//app", "./", "./tools/app "} {
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
	cgo := `{"ImportPath":"tei/tools/cgo-sample","Module":{"Path":"tei","Main":true},"CgoFiles":["main.go"],"Imports":["C","fmt"]}`
	got, err := parseGoList(strings.NewReader(cgo+"\n"+jsonStd), "tei/tools/cgo-sample", "")
	if err != nil {
		t.Fatalf("valid cgo dependency graph rejected: %v", err)
	}
	if len(got.Packages) != 1 || len(got.Packages[0].Imports) != 2 || got.Packages[0].Imports[0] != "C" {
		t.Fatalf("lost cgo source or import edge: %+v", got)
	}

	nonCgo := `{"ImportPath":"tei/tools/cgo-sample","Module":{"Path":"tei","Main":true},"GoFiles":["main.go"],"Imports":["C","fmt"]}`
	if _, err := parseGoList(strings.NewReader(nonCgo+"\n"+jsonStd), "tei/tools/cgo-sample", ""); err == nil {
		t.Fatal("pseudo import C must not excuse a missing Go dependency for non-cgo sources")
	}
}
