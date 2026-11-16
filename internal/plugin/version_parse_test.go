package plugin_test

import (
	"testing"

	"tei/internal/plugin"
)

// ParseVersion accepts exactly major.minor.patch, three dot-separated
// components of non-negative decimal integers. That is the contract every
// version gate in the package rests on: CompatibleMajor compares the major, and
// CompareVersion compares all three. Nothing pinned it, so a decoder that
// accepted "1.0" and answered 0.0.0 for it, or that accepted a signed
// component, would move every downstream answer without a failing test.
func TestParseVersionAcceptedSpellings(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    [3]int
	}{
		{name: "plain", version: "1.2.3", want: [3]int{1, 2, 3}},
		{name: "zeros", version: "0.0.0", want: [3]int{0, 0, 0}},
		{name: "large components", version: "10.200.3000", want: [3]int{10, 200, 3000}},
		{name: "leading zero in minor", version: "1.02.3", want: [3]int{1, 2, 3}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := plugin.ParseVersion(tc.version)
			if err != nil {
				t.Fatalf("ParseVersion(%q) refused: %v", tc.version, err)
			}
			if got != tc.want {
				t.Fatalf("ParseVersion(%q) = %v, want %v", tc.version, got, tc.want)
			}
		})
	}
}

// Every rejected spelling is part of the contract too: a version that parses
// becomes selectable, so a form the package does not mean to support must not
// slip through as 0.0.0 or as a partial match.
func TestParseVersionRejectedSpellings(t *testing.T) {
	for _, version := range []string{
		"",           // nothing to parse
		"1",          // major only
		"1.0",        // two components
		"1.0.0.0",    // four components
		"1..0",       // empty component
		".1.0",       // leading separator
		"1.0.",       // trailing separator
		"1.0.0 ",     // trailing space is not trimmed
		" 1.0.0",     // leading space is not trimmed
		"v1.0.0",     // no prefix support
		"1.0.x",      // non-numeric
		"1.-1.0",     // negative component
		"-1.0.0",     // negative major
		"1.0.0-beta", // pre-release suffix
		"1,0,0",      // wrong separator
	} {
		if got, err := plugin.ParseVersion(version); err == nil {
			t.Errorf("ParseVersion(%q) accepted it as %v", version, got)
		}
	}
}

// A leading zero and a leading plus are the two spellings that a decimal
// decoder accepts by accident, and they decide whether two declarations name the
// same version. Pin what ParseVersion does with each so a change to the decoder
// is deliberate.
func TestParseVersionAcceptsAmbiguousComponents(t *testing.T) {
	// A leading zero decodes to the same value as the plain form, so the two
	// spellings are the same version to every comparison in this package. That
	// is a consequence of Atoi, not an explicit rule, and it is recorded here.
	leading, err := plugin.ParseVersion("1.02.3")
	if err != nil {
		t.Fatalf("a leading zero in a component should decode: %v", err)
	}
	plain, err := plugin.ParseVersion("1.2.3")
	if err != nil {
		t.Fatalf("plain version: %v", err)
	}
	if leading != plain {
		t.Fatalf("1.02.3 (%v) and 1.2.3 (%v) must compare equal, or a lock could pin one spelling and match the other", leading, plain)
	}
	if plugin.DefinitionFingerprint(plugin.Definition{Version: "1.02.3"}) ==
		plugin.DefinitionFingerprint(plugin.Definition{Version: "1.2.3"}) {
		t.Fatal("the fingerprint hashes the written version, so two spellings differ there even though they parse equal")
	}

	// A leading plus is accepted, because strconv.Atoi reads "+1" as 1 while
	// refusing "-1". That asymmetry is the current contract: "1.+1.0" parses to
	// the same triple as "1.1.0", so a declaration written that way is
	// selectable. It is pinned here rather than described as intended, because
	// whether a signed component should be a spelling error is the Owner's call.
	plus, err := plugin.ParseVersion("+1.0.0")
	if err != nil {
		t.Fatalf("a leading plus currently parses; if that changed, update this test: %v", err)
	}
	if plus != [3]int{1, 0, 0} {
		t.Fatalf("+1.0.0 parsed as %v, want {1 0 0}", plus)
	}
	if !plugin.CompatibleMajor("+1.0.0", 1) {
		t.Fatal("a plus-prefixed major must still match major 1 while the spelling parses")
	}
}

// CompatibleMajor is a thin wrapper over ParseVersion, so its answers follow
// from the parse. A version it cannot parse matches no major, which is the
// behaviour the resolver's filtering depends on.
func TestCompatibleMajorFollowsTheParse(t *testing.T) {
	for _, tc := range []struct {
		version string
		major   int
		want    bool
	}{
		{version: "1.2.3", major: 1, want: true},
		{version: "1.2.3", major: 2, want: false},
		{version: "0.0.0", major: 0, want: true},
		{version: "1.0", major: 1, want: false},
		{version: "1.0.0.0", major: 1, want: false},
		{version: "", major: 1, want: false},
	} {
		if got := plugin.CompatibleMajor(tc.version, tc.major); got != tc.want {
			t.Errorf("CompatibleMajor(%q, %d) = %v, want %v", tc.version, tc.major, got, tc.want)
		}
	}
}
