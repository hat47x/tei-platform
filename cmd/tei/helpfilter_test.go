package main

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// Every command shares one FlagSet, so the flag package's own -h listing
// enumerates every flag in the binary. validate showed 70 that way while
// accepting 3, and flags such as -explain (a PostgreSQL EXPLAIN payload) read
// as though validate used them (#3956).
//
// printHelp already prints only the allow-listed flags. This test holds the
// routing: a requested listing must reach printHelp for every command, not only
// for the ones that happen to parse with ContinueOnError.
func TestHelpListsOnlyTheFlagsTheCommandReads(t *testing.T) {
	bin := buildCLIForLifecycle(t)
	flagLine := regexp.MustCompile(`(?m)^  -([a-z0-9-]+)`)

	// The two commands that used ExitOnError, so -h bypassed printHelp.
	cases := []struct {
		command string
		allowed []string
	}{
		{"validate", []string{"workspace", "world", "json"}},
		{"world", []string{"workspace", "world"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.command, func(t *testing.T) {
			allowed, ok := selectedReadOnlyFlagAllowlist[tc.command]
			if !ok {
				t.Fatalf("%s is not in the allow-list", tc.command)
			}
			out, err := exec.Command(bin, tc.command, "-h").CombinedOutput()
			if err != nil {
				t.Fatalf("%s -h: %v\n%s", tc.command, err, out)
			}
			text := string(out)
			if !strings.HasPrefix(text, "usage: tei "+tc.command) {
				t.Fatalf("%s -h does not use the filtered listing:\n%s", tc.command, text)
			}
			shown := flagLine.FindAllStringSubmatch(text, -1)
			permitted := make(map[string]bool, len(allowed))
			for _, name := range allowed {
				permitted[name] = true
			}
			if len(shown) != len(allowed) {
				t.Fatalf("%s -h lists %d flags, the command reads %d:\n%s",
					tc.command, len(shown), len(allowed), text)
			}
			for _, match := range shown {
				if !permitted[match[1]] {
					t.Fatalf("%s -h lists -%s, which the command does not read", tc.command, match[1])
				}
			}
		})
	}
}
