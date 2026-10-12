//go:build tei_repair_only && !tei_order_only

package main

import (
	"context"
	"fmt"
	"os"

	"tei/internal/composition"
	"tei/internal/ir"
)

func main() {
	err := runSingleApplicationTarget("repair_intake", func(world ir.World, config singleApplicationConfig) (
		readScenarioRunner, commandScenarioRunner, []webTargetOption, error,
	) {
		var lookup customerLookupRunner = newReferenceCustomerLookupScenarioRunner(world)
		if config.CompositionScenarios {
			digest, err := runningExecutableDigest()
			if err != nil {
				return nil, nil, nil, err
			}
			lookup, err = wireComposedCustomerLookup(context.Background(), composition.NewRegistry(), lookup, config.Model, digest,
				referenceInstanceGuard(digest, map[string]composition.Contract{"repair-customer": repairLookupContract}))
			if err != nil {
				return nil, nil, nil, err
			}
		}
		return nil, nil, []webTargetOption{withCustomerLookup(lookup)}, nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
