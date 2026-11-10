package main

import "testing"

// §4.30: the default is decided by which route the caller chose, and by whether
// the caller spoke about the second build at all. These cases pin all four
// combinations so a later change to one cannot silently move the others.

func TestApplyEvidenceDefaults(t *testing.T) {
	catalog := selfBuildInProcessArgs{
		CatalogPath:   "catalog.json",
		CatalogDigest: "sha256:abc",
		SourceRoot:    "src",
	}
	cases := []struct {
		name     string
		parsed   selfBuildOuterArgs
		wantTrue bool
	}{
		{
			name:     "in-process without the flag turns the second build on",
			parsed:   selfBuildOuterArgs{InProcess: catalog},
			wantTrue: true,
		},
		{
			name:     "in-process with an explicit false keeps it off",
			parsed:   selfBuildOuterArgs{InProcess: catalog, RebuildCheck: false, RebuildCheckSet: true},
			wantTrue: false,
		},
		{
			name:     "in-process with an explicit true stays on",
			parsed:   selfBuildOuterArgs{InProcess: catalog, RebuildCheck: true, RebuildCheckSet: true},
			wantTrue: true,
		},
		{
			// The child route cannot establish L2-b, so paying for a second build
			// there would add cost without adding a claim.
			name:     "the child route is not changed by the default",
			parsed:   selfBuildOuterArgs{IntentResolver: "resolver"},
			wantTrue: false,
		},
		{
			name:     "the child route keeps an explicit true",
			parsed:   selfBuildOuterArgs{IntentResolver: "resolver", RebuildCheck: true, RebuildCheckSet: true},
			wantTrue: true,
		},
		{
			name:     "an attempt with no derivation is not changed",
			parsed:   selfBuildOuterArgs{},
			wantTrue: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := applyEvidenceDefaults(tc.parsed)
			if got.RebuildCheck != tc.wantTrue {
				t.Fatalf("RebuildCheck = %v, want %v", got.RebuildCheck, tc.wantTrue)
			}
			// The helper decides one field and must not disturb the rest.
			if got.InProcess != tc.parsed.InProcess || got.IntentResolver != tc.parsed.IntentResolver {
				t.Fatal("the helper changed an argument it does not own")
			}
		})
	}
}

// A catalog named without its digest is refused before the default applies, so a
// half-specified catalog never gains the second build.
func TestParseSelfBuildOuterArgsRecordsWhetherTheCallerSpoke(t *testing.T) {
	base := []string{"-child", "c", "-workspace", "w", "-state", "s", "-intent-file", "i"}

	unset, err := parseSelfBuildOuterArgs(base)
	if err != nil {
		t.Fatal(err)
	}
	if unset.RebuildCheckSet {
		t.Fatal("RebuildCheckSet is true although the caller did not name the flag")
	}

	off, err := parseSelfBuildOuterArgs(append(append([]string{}, base...), "-rebuild-check=false"))
	if err != nil {
		t.Fatal(err)
	}
	if !off.RebuildCheckSet || off.RebuildCheck {
		t.Fatalf("explicit false not recorded: set=%v value=%v", off.RebuildCheckSet, off.RebuildCheck)
	}

	on, err := parseSelfBuildOuterArgs(append(append([]string{}, base...), "-rebuild-check"))
	if err != nil {
		t.Fatal(err)
	}
	if !on.RebuildCheckSet || !on.RebuildCheck {
		t.Fatalf("explicit true not recorded: set=%v value=%v", on.RebuildCheckSet, on.RebuildCheck)
	}
}
