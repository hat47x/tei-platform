//go:build tei_order_only && tei_api_only && !tei_repair_only && !tei_frozen_world

package main

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"tei/internal/ir"
)

func TestSingleApplicationClosesResourcesWhenFactoryFails(t *testing.T) {
	args := os.Args
	defer func() { os.Args = args }()
	buildRevision := referenceAPISourceRevision
	defer func() { referenceAPISourceRevision = buildRevision }()
	const revision = "1111111111111111111111111111111111111111"
	referenceAPISourceRevision = revision
	os.Args = []string{"target", "-workspace", referenceWorkspacePath(t), "-baseline-id", testBaselineID,
		"-baseline-revision", testBaselineRevision, "-target-revision", revision}
	var closed []int
	failure := errors.New("factory failed after resources opened")
	err := runSingleApplicationTarget("order_entry", func(_ ir.World, config singleApplicationConfig) (
		readScenarioRunner, commandScenarioRunner, []webTargetOption, error,
	) {
		config.OnClose(func() { closed = append(closed, 1) })
		config.OnClose(func() { closed = append(closed, 2) })
		return nil, nil, nil, failure
	})
	if !errors.Is(err, failure) || !reflect.DeepEqual(closed, []int{2, 1}) {
		t.Fatalf("failure=%v, cleanup=%v", err, closed)
	}
}

func TestSingleApplicationNonScreenExitsWithoutHTTPAndClosesResources(t *testing.T) {
	args := os.Args
	defer func() { os.Args = args }()
	buildRevision := referenceAPISourceRevision
	defer func() { referenceAPISourceRevision = buildRevision }()
	const revision = "1111111111111111111111111111111111111111"
	referenceAPISourceRevision = revision
	os.Args = []string{"target", "-workspace", referenceWorkspacePath(t), "-baseline-id", testBaselineID,
		"-baseline-revision", testBaselineRevision, "-target-revision", revision,
		"-composition-scenarios", "-postgres-dsn-file", "factory-owned", "-postgres-fencing-token", "1", "-non-screen", "report"}
	closed := false
	err := runSingleApplicationTarget("order_entry", func(_ ir.World, config singleApplicationConfig) (
		readScenarioRunner, commandScenarioRunner, []webTargetOption, error,
	) {
		config.OnClose(func() { closed = true })
		// A CLI factory intentionally returns no HTTP runners. Constructing
		// an HTTP server from these values would fail instead of completing.
		return nil, nil, nil, nil
	})
	if err != nil || !closed {
		t.Fatalf("CLI completion=%v, resources closed=%v", err, closed)
	}
}
