package main

import (
	"strings"
	"testing"
)

func pinTestSet() ArtifactSet {
	return ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{
			{Role: "executable", Format: "application/octet-stream", Digest: digestA, Locator: "/tmp/first/app"},
			{Role: "manifest", Format: "application/json", Digest: digestB, Locator: "/tmp/first/manifest.json"},
		},
	}
}

func cloneArtifactSet(set ArtifactSet) ArtifactSet {
	clone := set
	clone.Members = append([]ArtifactMember(nil), set.Members...)
	return clone
}

// 指紋からRoleやFormatを外す変異を検出する。内容(digest)が同じでも、
// 集合の中でどのRoleがどの形式であるかは、集合の同一性の一部である。
func TestArtifactSetIdentityChangesWhenRoleOrFormatChanges(t *testing.T) {
	base := pinTestSet()
	baseFP, err := artifactSetFingerprint(base)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		mutate func(*ArtifactSet)
	}{
		{"role renamed", func(s *ArtifactSet) { s.Members[0].Role = "binary" }},
		{"format changed", func(s *ArtifactSet) { s.Members[1].Format = "text/plain" }},
		{"digests swapped between roles", func(s *ArtifactSet) {
			s.Members[0].Digest, s.Members[1].Digest = s.Members[1].Digest, s.Members[0].Digest
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			changed := cloneArtifactSet(base)
			tc.mutate(&changed)
			fp, err := artifactSetFingerprint(changed)
			if err != nil {
				t.Fatal(err)
			}
			if fp == baseFP {
				t.Fatalf("Artifact Set identity ignored %s", tc.name)
			}
		})
	}
}

func TestPinReproducibleArtifactSetReturnsIdentityForSameContent(t *testing.T) {
	first := pinTestSet()
	// 宣言順とlocatorだけが異なる集合は、同じ内容として再現したものと扱う。
	second := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{
			{Role: "manifest", Format: "application/json", Digest: uppercaseDigest(digestB), Locator: "store://other/manifest"},
			{Role: "executable", Format: "application/octet-stream", Digest: digestA, Locator: "store://other/app"},
		},
	}
	want, err := artifactSetFingerprint(first)
	if err != nil {
		t.Fatal(err)
	}
	got, err := pinReproducibleArtifactSet(first, second)
	if err != nil {
		t.Fatalf("a reproducible Artifact Set was refused: %v", err)
	}
	if got != want {
		t.Fatalf("pinned identity %s is not the content identity %s", got, want)
	}
}

func TestPinReproducibleArtifactSetRefusesNonReproducibleSets(t *testing.T) {
	cases := []struct {
		name         string
		mutate       func(*ArtifactSet)
		wantEvidence string
	}{
		{"member digest changed", func(s *ArtifactSet) { s.Members[0].Digest = digestC }, "changed: executable"},
		{"member format changed", func(s *ArtifactSet) { s.Members[1].Format = "text/plain" }, "changed: manifest"},
		{"member missing from second", func(s *ArtifactSet) { s.Members = s.Members[:1] }, "missing from second: manifest"},
		{"extra member in second", func(s *ArtifactSet) {
			s.Members = append(s.Members, ArtifactMember{Role: "symbols", Format: "application/octet-stream", Digest: digestC})
		}, "extra in second: symbols"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			first := pinTestSet()
			second := cloneArtifactSet(first)
			tc.mutate(&second)

			// 比較自体はEvidenceを返す。拒否するのは固定する関数の側である。
			comparison, err := compareArtifactSets(first, second)
			if err != nil {
				t.Fatal(err)
			}
			if comparison.Reproducible {
				t.Fatal("test fixture did not produce a non-reproducible comparison")
			}

			fp, err := pinReproducibleArtifactSet(first, second)
			if err == nil {
				t.Fatalf("a non-reproducible Artifact Set was pinned as canonical output: %s", fp)
			}
			if fp != "" {
				t.Fatalf("refusal still returned an identity: %q", fp)
			}
			if !strings.Contains(err.Error(), tc.wantEvidence) {
				t.Fatalf("refusal does not say which role drifted: %v", err)
			}
		})
	}
}

func TestPinReproducibleArtifactSetRefusesInvalidSets(t *testing.T) {
	valid := pinTestSet()
	duplicate := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{
			{Role: "executable", Format: "application/octet-stream", Digest: digestA},
			{Role: "executable", Format: "application/octet-stream", Digest: digestA},
		},
	}
	empty := ArtifactSet{FormatVersion: artifactSetFormatVersion}

	for name, pair := range map[string][2]ArtifactSet{
		"duplicate role in first":  {duplicate, valid},
		"duplicate role in second": {valid, duplicate},
		"empty first":              {empty, valid},
		"empty second":             {valid, empty},
	} {
		t.Run(name, func(t *testing.T) {
			fp, err := pinReproducibleArtifactSet(pair[0], pair[1])
			if err == nil || fp != "" {
				t.Fatalf("invalid Artifact Set was pinned: fp=%q err=%v", fp, err)
			}
		})
	}
}
