//go:build tei_repair_only && !tei_order_only

package main

import (
	"fmt"
	"os"

	"tei/internal/ir"
)

func main() {
	err := runSingleApplicationTarget("repair_intake", func(world ir.World, _ singleApplicationConfig) (
		readScenarioRunner, commandScenarioRunner, []webTargetOption, error,
	) {
		return nil, nil, []webTargetOption{
			withCustomerLookup(newReferenceCustomerLookupScenarioRunner(world)),
		}, nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
