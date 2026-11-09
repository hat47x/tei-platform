//go:build tei_api_only && tei_order_only && !tei_repair_only

package main

import (
	"os"
	"strings"
	"testing"

	"tei/internal/ir"
)

func TestDiagnosticAPIPinnedBuildRevision(t *testing.T) {
	original := referenceAPISourceRevision
	t.Cleanup(func() { referenceAPISourceRevision = original })
	const T = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for _, tc := range []struct {
		pin   string
		arg   string
		allow bool
	}{
		{T, T, true},
		{"", T, false},
		{"unresolved", T, false},
		{strings.ToUpper(T), T, false},
		{strings.Repeat("b", 40), T, false},
		{T, strings.Repeat("b", 40), false},
		{T, "", false},
		{T, "T", false},
	} {
		referenceAPISourceRevision = tc.pin
		err := validateReferenceBuildRevision(tc.arg)
		if (err == nil) != tc.allow {
			t.Errorf("pin=%q requested=%q: error=%v wantAllow=%t", tc.pin, tc.arg, err, tc.allow)
		}
	}
}

func TestAPIOnlyRejectsUnpinnedTargetBeforeWorldAndRunner(t *testing.T) {
	originalArgs := os.Args
	originalPin := referenceAPISourceRevision
	t.Cleanup(func() {
		os.Args = originalArgs
		referenceAPISourceRevision = originalPin
	})
	const T = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	os.Args = []string{
		"order-entry", "-workspace", "this-path-does-not-exist",
		"-baseline-id", "B",
		"-baseline-revision", strings.Repeat("b", 40),
		"-target-revision", T,
		"-listen", "127.0.0.1:8081",
	}
	called := false
	factory := func(ir.World, singleApplicationConfig) (
		readScenarioRunner, commandScenarioRunner, []webTargetOption, error,
	) {
		called = true
		return nil, nil, nil, nil
	}
	for _, bad := range []string{"", strings.Repeat("c", 40)} {
		referenceAPISourceRevision = bad
		called = false
		err := runSingleApplicationTarget("order_entry", factory)
		if err == nil || !strings.Contains(err.Error(), "build revision") || called {
			t.Fatalf("pin %q did not fail before World/Runner (err=%v, called=%t)", bad, err, called)
		}
	}
	// A valid pin proceeds past the early guard. Loading the deliberately
	// nonexistent World should now fail, before any Runner factory is called.
	referenceAPISourceRevision = T
	err := runSingleApplicationTarget("order_entry", factory)
	if err == nil || !strings.Contains(err.Error(), "load reference Application model") || called {
		t.Fatalf("valid pin did not reach World load (err=%v, called=%t)", err, called)
	}
}
