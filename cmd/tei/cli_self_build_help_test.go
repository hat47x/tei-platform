package main

import (
	"bytes"
	"strings"
	"testing"
)

// `-h` is how a caller discovers a command. Leaving the §4 flags out of it made
// the catalog route reachable only by reading the design document, which is the
// opposite of discoverable. These tests pin the listing to the grammar the
// command actually accepts.

func selfBuildHelp(t *testing.T) string {
	t.Helper()
	var out bytes.Buffer
	code, err := runSelfBuildCommand([]string{"-h"}, &out)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if code != 0 {
		t.Fatalf("code = %d, want 0: -h is a request to read the grammar", code)
	}
	return out.String()
}

func TestSelfBuildHelpListsTheCatalogRoute(t *testing.T) {
	help := selfBuildHelp(t)
	// Every flag that makes the L2-b route reachable must appear, or a caller
	// cannot find it from the command line at all.
	for _, name := range []string{
		"-build-catalog",
		"-build-catalog-digest",
		"-build-source-root",
		"-intent-resolver",
		"-expected-plan-fingerprint",
		"-rebuild-check",
		"-supply-verifier",
	} {
		if !strings.Contains(help, name) {
			t.Fatalf("-h does not list %s:\n%s", name, help)
		}
	}
}

func TestSelfBuildHelpStillListsThePrimitiveForm(t *testing.T) {
	help := selfBuildHelp(t)
	// The primitive -child/-artifact form is still accepted, so its own flags
	// must not be dropped by deriving the text from the §4 FlagSet.
	for _, name := range []string{"-child", "-child-arg", "-child-arg-file", "-artifact", "-claimed-self"} {
		if !strings.Contains(help, name) {
			t.Fatalf("-h does not list %s:\n%s", name, help)
		}
	}
}

// The default that §4.30 introduced has to be visible where a caller looks for
// it: the flag's own line.
func TestSelfBuildHelpStatesTheRebuildCheckDefault(t *testing.T) {
	help := selfBuildHelp(t)
	index := strings.Index(help, "-rebuild-check")
	if index < 0 {
		t.Fatal("-rebuild-check is not listed")
	}
	rest := help[index:]
	if !strings.Contains(rest, "§4.30") {
		t.Fatalf("the -rebuild-check line does not explain the in-process default:\n%s", rest)
	}
}

// The other commands' help must be unchanged by the shared rendering helper.
func TestOtherCommandHelpIsUnchanged(t *testing.T) {
	var out bytes.Buffer
	code, err := runViewCommand([]string{"-h"}, &out)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "usage: tei view [options]") {
		t.Fatalf("view help lost its usage line:\n%s", out.String())
	}
}
