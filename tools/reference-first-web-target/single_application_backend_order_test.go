//go:build tei_order_only && !tei_repair_only

package main

import (
	"flag"
	"testing"
)

func TestOrderBackendRejectsIncompleteOrUncomposedSelection(t *testing.T) {
	for _, config := range []singleApplicationConfig{
		{PostgresDSNFile: "missing"},
		{PostgresFence: 1},
		{PostgresDSNFile: "missing", PostgresFence: 1},
		{CompositionScenarios: true, PostgresDSNFile: "missing"},
		{CompositionScenarios: true, PostgresFence: 1},
		{CompositionScenarios: true, PostgresDSNFile: "missing", PostgresFence: -1},
	} {
		if err := validateSingleApplicationBackend(config); err == nil {
			t.Fatalf("incomplete backend accepted: %+v", config)
		}
	}
	for _, config := range []singleApplicationConfig{
		{}, {CompositionScenarios: true},
		{CompositionScenarios: true, PostgresDSNFile: "explicit", PostgresFence: 1},
	} {
		if err := validateSingleApplicationBackend(config); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOrderNonScreenSelectionRequiresPersistentServices(t *testing.T) {
	for _, test := range []struct {
		args  []string
		valid bool
	}{
		{[]string{"-non-screen", "report"}, false},
		{[]string{"-non-screen", "batch", "-postgres-dsn-file", "explicit", "-postgres-fencing-token", "1"}, false},
		{[]string{"-non-screen", "unknown", "-postgres-dsn-file", "explicit", "-postgres-fencing-token", "1", "-composition-scenarios"}, false},
		{[]string{"-non-screen", "report", "-postgres-dsn-file", "explicit", "-postgres-fencing-token", "1", "-composition-scenarios"}, true},
		{[]string{"-non-screen", "batch", "-postgres-dsn-file", "explicit", "-postgres-fencing-token", "1", "-composition-scenarios"}, true},
		{[]string{"-non-screen", "batch-observe", "-postgres-dsn-file", "explicit", "-postgres-fencing-token", "1", "-composition-scenarios"}, true},
	} {
		flags := flag.NewFlagSet("order", flag.ContinueOnError)
		config := singleApplicationConfig{}
		registerSingleApplicationBackendFlags(flags, &config)
		flags.BoolVar(&config.CompositionScenarios, "composition-scenarios", false, "")
		if err := flags.Parse(test.args); err != nil {
			t.Fatal(err)
		}
		if (validateSingleApplicationBackend(config) == nil) != test.valid || config.Backend.servesHTTP() {
			t.Fatalf("backend selection disagrees with mode: %v", test.args)
		}
	}
}
