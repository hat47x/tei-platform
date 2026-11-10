package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// TestRepeatedSingleValueFlagIsRefused pins #3675: Go's flag package keeps the
// last value and reports nothing, so a caller that concatenates an environment
// default with an explicit override lost the earlier value silently. Every
// advertised flag except the documented repeatable ones now refuses a repeat.
func TestRepeatedSingleValueFlagIsRefused(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	cases := []struct {
		name string
		args []string
	}{
		{"model catalog -namespace", []string{"-workspace", reference, "-namespace", "sales", "-namespace", "std"}},
		{"trace -mode", []string{"-workspace", reference, "-scope", "s", "-subject-kind", "semantic-data",
			"-subject-id", "sales.order", "-mode", "summary", "-mode", "full"}},
		{"inspect -subject-kind", []string{"-workspace", reference, "-subject-kind", "semantic-data",
			"-subject-kind", "a", "-subject-id", "sales.order"}},
		{"source-index -workspace", []string{"-workspace", reference, "-workspace", reference}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			command := strings.Fields(tc.name)[0]
			if command == "model" {
				command = "model catalog"
			}
			err := validateRepeatableArguments(command, tc.args)
			if err == nil {
				t.Fatalf("a repeated single-value flag was accepted: %v", tc.args)
			}
			if !strings.Contains(err.Error(), "more than once") {
				t.Fatalf("message does not state the rule: %v", err)
			}
			_ = out
		})
	}
}

// A documented repeatable flag keeps working: view -input collects pairs and
// validate -analysis collects files.
func TestRepeatableFlagsAreStillAccepted(t *testing.T) {
	for _, tc := range []struct {
		command string
		args    []string
	}{
		{"view", []string{"-workspace", "w", "-application", "a", "-input", "a=1", "-input", "b=2"}},
		{"view", []string{"-workspace", "w", "-application", "a", "-allow-capability", "x", "-allow-capability", "y"}},
		// validate does not read -analysis/-schema/-plan/-evidence; those belong to
		// observed-index and deployment-*. A repeat of one of them is refused
		// because validate does not accept the flag at all (#3956).
		{"view", []string{"-workspace", "w", "-application", "a", "-input", "x=1", "-input", "y=2"}},
	} {
		if err := validateRepeatableArguments(tc.command, tc.args); err != nil {
			t.Fatalf("%s: a repeatable flag was refused: %v", tc.command, err)
		}
	}
}

// A flag value that starts with "-" must not be mistaken for a flag, and a
// positional argument must end the scan the way flag parsing does.
func TestRepeatScanSkipsValues(t *testing.T) {
	// -input takes its value as the next argument, and that value starts with
	// a dash. It must not be counted as a -b flag.
	if err := validateRepeatableArguments("view", []string{"-input", "--not-a-flag", "-input", "--also-not"}); err != nil {
		t.Fatalf("a dash-leading value was treated as a flag: %v", err)
	}
	// A positional argument stops flag parsing, so a repeat after it is not a
	// flag at all and must not be reported.
	if err := validateRepeatableArguments("inspect", []string{"-workspace", "w", "positional", "-workspace", "x"}); err != nil {
		t.Fatalf("a flag after a positional argument was scanned: %v", err)
	}
}

// The catalogue must advertise which flags repeat, so a caller can build
// arguments without guessing.
func TestCatalogueAdvertisesRepeatableFlags(t *testing.T) {
	catalog := commandCatalog()
	found := map[string][]string{}
	for _, record := range catalog.Commands {
		if len(record.RepeatableFlags) > 0 {
			found[record.Name] = record.RepeatableFlags
		}
	}
	view, ok := found["view"]
	if !ok || !containsAll(view, "-input", "-allow-capability") {
		t.Fatalf("view repeatable flags = %v", view)
	}
	// validate advertises none: every flag it reads holds one value, and the
	// repeatable list it used to carry named flags it refuses (#3956).
	if validation, ok := found["validate"]; ok {
		t.Fatalf("validate advertises repeatable flags %v but refuses them", validation)
	}
	// A command whose flags all hold one value advertises nothing.
	for _, name := range []string{"inspect", "trace", "model catalog", "source-index"} {
		if flags, ok := found[name]; ok {
			t.Fatalf("%s advertises repeatable flags %v", name, flags)
		}
	}
}

// The wiring must be reachable from the CLI: a repeated flag on a real
// invocation is refused with one JSON envelope and exit 2.
func TestRepeatedFlagIsRefusedByTheCatalogueCommand(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	var out bytes.Buffer
	code, err := runModelCommand([]string{"catalog", "-workspace", reference,
		"-namespace", "sales", "-namespace", "std"}, &out)
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	if code != 2 {
		t.Fatalf("exit=%d, want 2 (%s)", code, out.String())
	}
	var response struct {
		OK    bool `json:"ok"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatalf("answer is not JSON: %v (%q)", err, out.String())
	}
	if response.OK || response.Error == nil || response.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("repeated flag result = %+v", response)
	}
	if !strings.Contains(response.Error.Message, "-namespace was given more than once") {
		t.Fatalf("message does not name the flag: %q", response.Error.Message)
	}
	// Omitting the second occurrence still succeeds, so the check does not turn
	// the flag into a required one.
	var single bytes.Buffer
	if code, err := runModelCommand([]string{"catalog", "-workspace", reference, "-namespace", "sales"}, &single); err != nil || code != 0 {
		t.Fatalf("single -namespace: code=%d err=%v (%s)", code, err, single.String())
	}
}

func containsAll(values []string, want ...string) bool {
	set := map[string]bool{}
	for _, value := range values {
		set[value] = true
	}
	for _, name := range want {
		if !set[name] {
			return false
		}
	}
	return true
}
