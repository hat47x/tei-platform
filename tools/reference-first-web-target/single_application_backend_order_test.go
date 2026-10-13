//go:build tei_order_only && !tei_repair_only

package main

import "testing"

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
