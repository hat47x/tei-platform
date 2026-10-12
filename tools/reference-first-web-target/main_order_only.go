//go:build tei_order_only && !tei_repair_only

package main

import (
	"context"
	"fmt"
	"os"

	"tei/internal/composition"
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
		command.model = config.Model
		read := newReferenceReadScenarioRunner(world)
		if config.CompositionScenarios {
			digest, err := runningExecutableDigest()
			if err != nil {
				return nil, nil, nil, err
			}
			r, c, err := wireComposedScenarios(context.Background(), composition.NewRegistry(), read, command, digest,
				referenceInstanceGuard(digest, map[string]composition.Contract{"order-read": readServiceContract, "order-submit": submitServiceContract}))
			return r, c, nil, err
		}
		return read, command, nil, nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
