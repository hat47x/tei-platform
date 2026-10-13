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
