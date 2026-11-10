package main

import (
	"bytes"
	"flag"
	"io"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// helpFlagNames reads the flag names out of a usage listing.
func helpFlagNames(text string) []string {
	pattern := regexp.MustCompile(`(?m)^\s+-([A-Za-z0-9-]+)`)
	seen := map[string]bool{}
	var names []string
	for _, match := range pattern.FindAllStringSubmatch(text, -1) {
		if !seen[match[1]] {
			seen[match[1]] = true
			names = append(names, match[1])
		}
	}
	sort.Strings(names)
	return names
}

// TestHelpDescribesOnlyTheFlagsTheCommandAccepts pins #3943. Every command
// shares one FlagSet, so `-h` used to describe all 70 flags of every command:
// `tei validate -h` listed flags validate refuses, and a reader could not tell
// the grammar from the dump. The listing must name the accepted flags only.
func TestHelpDescribesOnlyTheFlagsTheCommandAccepts(t *testing.T) {
	cases := []struct {
		command string
		want    []string
	}{
		{"validate", []string{"json", "workspace", "world"}},
		{"world", []string{"workspace", "world"}},
		{"requirements", []string{"workspace", "world"}},
		{"dbcheck", []string{"dsn"}},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			// Both grammars share one FlagSet, so the help text is filtered by
			// the command name rather than by what the FlagSet happens to declare.
			fs := flag.NewFlagSet(tc.command, flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			fs.String("workspace", "", "TEI workspace directory")
			fs.String("world", "", "normalized M0 IR JSON")
			fs.String("pattern", "", "at most one of -pattern, -package, -command")
			fs.String("advance", "", "periods: move this period one step")
			fs.String("grant-capability", "", "issue-grant: the capability")
			fs.String("qualification-requirements", "", "knowledge qualification requirements JSON")
			fs.String("period-states", "", "periods: period=state pairs")
			fs.Bool("json", false, "machine-readable JSON for validate -workspace")
			fs.String("dsn", "", "dbcheck: PostgreSQL DSN")
			var out bytes.Buffer
			printHelp(&out, fs, tc.command)
			got := helpFlagNames(out.String())
			want := append([]string(nil), tc.want...)
			sort.Strings(want)
			if len(got) != len(want) {
				t.Fatalf("help lists %v, want %v", got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("help lists %v, want %v", got, want)
				}
			}
			// The shared FlagSet's other flags must not appear.
			for _, other := range []string{"pattern", "advance", "grant-capability",
				"qualification-requirements", "period-states"} {
				if strings.Contains(out.String(), "-"+other) {
					t.Fatalf("help leaked the unrelated flag -%s", other)
				}
			}
		})
	}
}

// A help request must be recognised only in flag position, so a value that
// happens to be "-h" is not taken for one.
func TestRequestedHelpRecognisesOnlyFlagPosition(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{[]string{"-h"}, true},
		{[]string{"-help"}, true},
		{[]string{"--help"}, true},
		{[]string{"-workspace", "w", "-h"}, true},
		{[]string{"-workspace", "w"}, false},
		{[]string{"--", "-h"}, false},
		{nil, false},
	}
	for _, tc := range cases {
		if got := requestedHelp(tc.args); got != tc.want {
			t.Fatalf("requestedHelp(%v)=%v, want %v", tc.args, got, tc.want)
		}
	}
}
