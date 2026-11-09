package gobuilder

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"sort"
	"testing"
)

func TestSourceClosureDigestPreservesReferenceEncodingAndOrder(t *testing.T) {
	members := []SourceMember{
		{Path: "pkg/z.go", Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		{Path: "go.mod", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
	ordered := append([]SourceMember(nil), members...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	legacyBytes := []byte("go.mod\x00sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n" +
		"pkg/z.go\x00sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n")
	digest := sha256.Sum256(legacyBytes)
	want := "sha256:" + hex.EncodeToString(digest[:])
	got := SourceClosureDigest(members)
	if got != want || SourceClosureDigest(ordered) != want {
		t.Fatalf("closure encoding changed: got %s want %s", got, want)
	}
	changed := append([]SourceMember(nil), members...)
	changed[0].Path = "pkg/renamed.go"
	if SourceClosureDigest(changed) == want {
		t.Fatal("build-relevant selected file path was ignored")
	}
	changed[0] = members[0]
	changed[0].Digest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	if SourceClosureDigest(changed) == want {
		t.Fatal("selected file bytes digest was ignored")
	}
	if !reflect.DeepEqual(members[0], SourceMember{Path: "pkg/z.go", Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}) {
		t.Fatal("fingerprinting mutated input members")
	}
}

func TestPackageBuildFilesPreservesSelectedCompilerInputs(t *testing.T) {
	got := PackageBuildFiles(GoListPackage{
		GoFiles:      []string{"z.go", "a.go"},
		CgoFiles:     []string{"cgo.go"},
		CFiles:       []string{"code.c"},
		CXXFiles:     []string{"code.cpp"},
		MFiles:       []string{"code.m"},
		HFiles:       []string{"header.h"},
		FFiles:       []string{"code.f"},
		SFiles:       []string{"code.s"},
		SwigFiles:    []string{"code.swig"},
		SwigCXXFiles: []string{"code.swigcxx"},
		SysoFiles:    []string{"resource.syso"},
		EmbedFiles:   []string{"settings.json"},
	})
	want := []string{"a.go", "cgo.go", "code.c", "code.cpp", "code.f", "code.m", "code.s", "code.swig", "code.swigcxx", "header.h", "resource.syso", "settings.json", "z.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selected Go input closure changed: got %#v want %#v", got, want)
	}
	// Ignore unselected alternative-target files. They must be absent from the
	// actual closure even when the file exists elsewhere in the repository.
	if len(PackageBuildFiles(GoListPackage{GoFiles: []string{"linux.go"}})) != 1 {
		t.Fatal("selected source resolver included unselected target files")
	}
}
