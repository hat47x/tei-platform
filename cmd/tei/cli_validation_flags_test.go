package main

import (
	"flag"
	"testing"
)

func TestWantsJSONValidationBeforeFlagParsing(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"-workspace", "ws"}, false},
		{[]string{"-json"}, true},
		{[]string{"--json=true"}, true},
		{[]string{"-json=invalid"}, true},
		{[]string{"-no-such-flag", "-json"}, true},
		{[]string{"-limit", "not-an-integer", "--json"}, true},
		{[]string{"-json=false"}, false},
		{[]string{"-json", "-json=false"}, false},
		{[]string{"-json=false", "-json"}, true},
	} {
		if got := wantsJSONValidation(tc.args); got != tc.want {
			t.Errorf("wantsJSONValidation(%q)=%t; want %t", tc.args, got, tc.want)
		}
	}
}

func TestJSONValidationRejectsOtherCommandFlags(t *testing.T) {
	for _, tc := range []struct {
		args []string
		wantError bool
	}{
		{[]string{"-workspace", "ws", "-json"}, false},
		{[]string{"-workspace", "ws", "-json", "-world="}, true},
		{[]string{"-workspace", "ws", "-json", "-scope", "other"}, true},
		{[]string{"-workspace", "ws", "-json", "-limit", "1"}, true},
	} {
		fs := flag.NewFlagSet("validate", flag.ContinueOnError)
		fs.String("workspace", "", "")
		fs.Bool("json", false, "")
		fs.String("world", "", "")
		fs.String("scope", "", "")
		fs.Int("limit", 0, "")
		if err := fs.Parse(tc.args); err != nil {
			t.Fatalf("flag parsing %q failed: %v", tc.args, err)
		}
		if err := validateJSONValidationFlags(fs); (err != nil) != tc.wantError {
			t.Errorf("JSON validation flags %q: error=%v; wantError=%t", tc.args, err, tc.wantError)
		}
	}
}
