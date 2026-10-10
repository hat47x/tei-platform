package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestSelectedReadOnlyFlagsDisallowUnrelatedArguments(t *testing.T) {
	for _, tc := range []struct {
		command string
		allowed []string
		bad     string
	}{
		{"trace", []string{"workspace", "scope", "subject-kind", "subject-id", "direction", "mode", "offset", "limit", "expected-fingerprint"}, "environment"},
		{"inspect", []string{"workspace", "subject-kind", "subject-id"}, "scope"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			fs := flag.NewFlagSet(tc.command, flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			for _, name := range append(append([]string{}, tc.allowed...), tc.bad) {
				fs.String(name, "", "")
			}
			if err := fs.Parse([]string{"-" + tc.bad, "unexpected"}); err != nil {
				t.Fatal(err)
			}
			if err := validateSelectedReadOnlyFlags(fs, tc.command); err == nil ||
				!strings.Contains(err.Error(), tc.bad) {
				t.Fatalf("unrelated flag was silently accepted: %v", err)
			}
			var out bytes.Buffer
			code, err := selectedReadOnlyFlagError(&out, tc.command, fmt.Errorf("invalid CLI argument"))
			if err != nil || code != 2 || !json.Valid(out.Bytes()) {
				t.Fatalf("error is not a single JSON document: %d %v %q", code, err, out.String())
			}
			var report struct {
				OK bool `json:"ok"`
				Error struct { Code string `json:"code"` } `json:"error"`
			}
			if err := json.Unmarshal(out.Bytes(), &report); err != nil ||
				report.OK || report.Error.Code != "INVALID_ARGUMENT" {
				t.Fatalf("invalid JSON argument envelope: %+v %v", report, err)
			}
		})
	}
}

func TestSelectedReadOnlyFlagsAllowDocumentedArguments(t *testing.T) {
	for _, command := range []string{"inspect", "trace"} {
		fs := flag.NewFlagSet(command, flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		fs.String("workspace", "", "")
		fs.String("subject-kind", "", "")
		fs.String("subject-id", "", "")
		if command == "trace" {
			fs.String("scope", "", "")
		}
		if err := fs.Parse([]string{"-workspace", "test", "-subject-kind", "semantic-data", "-subject-id", "a"}); err != nil {
			t.Fatal(err)
		}
		if err := validateSelectedReadOnlyFlags(fs, command); err != nil {
			t.Fatalf("documented flag rejected by %s: %v", command, err)
		}
	}
}
