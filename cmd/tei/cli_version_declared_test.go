package main

import (
	"runtime/debug"
	"strings"
	"testing"
)

const testDeclaredRevision = "0123456789abcdef0123456789abcdef01234567"

// withDeclaredRevision sets the build-time variable for one test and restores
// it afterwards, so the package-level value cannot leak between tests.
func withDeclaredRevision(t *testing.T, value string) {
	t.Helper()
	previous := declaredBuildRevision
	declaredBuildRevision = value
	t.Cleanup(func() { declaredBuildRevision = previous })
}

// #3558: a build with -buildvcs=false can name its source commit with
// -ldflags "-X main.declaredBuildRevision=<commit>". That value is the
// builder's own statement, so it is reported under a different name and a
// different source, never as the toolchain's vcsRevision.
func TestBuildIdentityReportsOnlyTheDeclaredRevisionWithoutBuildInfo(t *testing.T) {
	withDeclaredRevision(t, testDeclaredRevision)
	identity := buildIdentity(nil, false)
	if identity.Source != "ldflags" || identity.DeclaredRevision != testDeclaredRevision {
		t.Fatalf("declared revision was not reported as such: %+v", identity)
	}
	if identity.VCSRevision != "" || identity.VCSModified {
		t.Fatalf("declared revision leaked into the toolchain stamp: %+v", identity)
	}
}

func TestBuildIdentityWithoutAnyRevisionStaysUnavailable(t *testing.T) {
	withDeclaredRevision(t, "")
	identity := buildIdentity(nil, false)
	if identity.Source != "unavailable" || identity.DeclaredRevision != "" {
		t.Fatalf("a missing revision must not be invented: %+v", identity)
	}
}

// When the toolchain stamped the binary, that stamp stays the identity even if
// a declared revision was also passed. The declared value is kept separately
// so a reader can see both without confusing their strength.
func TestBuildIdentityKeepsTheToolchainStampFirst(t *testing.T) {
	withDeclaredRevision(t, testDeclaredRevision)
	info := &debug.BuildInfo{
		GoVersion: "go1.26.10",
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "89abcdef0123456789abcdef0123456789abcdef"},
		},
	}
	identity := buildIdentity(info, true)
	if identity.Source != "build-info" || identity.VCSRevision != "89abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("toolchain stamp was replaced by the declared revision: %+v", identity)
	}
	if identity.DeclaredRevision != testDeclaredRevision {
		t.Fatalf("declared revision was dropped: %+v", identity)
	}
}

// A malformed -X value is not a revision. Reporting it would let a mistaken
// flag look like the identity of the build.
func TestDeclaredRevisionRejectsMalformedValues(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		want  string
	}{
		{name: "full hex", value: testDeclaredRevision, want: testDeclaredRevision},
		{name: "upper case is normalised", value: strings.ToUpper(testDeclaredRevision), want: testDeclaredRevision},
		{name: "surrounding space", value: "  " + testDeclaredRevision + "  ", want: testDeclaredRevision},
		{name: "short hash", value: "0123456", want: ""},
		{name: "not hex", value: "0123456789abcdef0123456789abcdef012345zz", want: ""},
		{name: "empty", value: "", want: ""},
		{name: "a branch name", value: "feature/some-branch-name-that-is-not-a-commit", want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withDeclaredRevision(t, tc.value)
			if got := declaredRevision(); got != tc.want {
				t.Fatalf("declaredRevision(%q)=%q want %q", tc.value, got, tc.want)
			}
		})
	}
}

// The text form names the declared revision on its own line so that a reader
// of the plain output cannot mistake it for the toolchain stamp.
func TestVersionTextOutputSeparatesTheDeclaredRevision(t *testing.T) {
	withDeclaredRevision(t, testDeclaredRevision)
	info := &debug.BuildInfo{}
	identity := buildIdentity(info, false)
	if identity.Source != "ldflags" {
		t.Fatalf("source=%q want ldflags", identity.Source)
	}
}
