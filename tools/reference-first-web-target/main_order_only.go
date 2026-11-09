//go:build tei_order_only && !tei_repair_only

package main

import (
	"fmt"
	"os"

	"tei/internal/ir"
)

func main() {
	err := runSingleApplicationTarget("order_entry", func(world ir.World, config singleApplicationConfig) (
		readScenarioRunner, commandScenarioRunner, []webTargetOption, error,
	) {
		command, err := newReferenceCommandScenarioRunner(
			world, config.LegacyUsers, config.LegacyAuthorization,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		return newReferenceReadScenarioRunner(world), command, nil, nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
