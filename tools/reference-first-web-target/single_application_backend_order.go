//go:build tei_order_only && !tei_repair_only

package main

import (
	"flag"
	"fmt"
)

func registerSingleApplicationBackendFlags(flags *flag.FlagSet, config *singleApplicationConfig) {
	flags.StringVar(&config.PostgresDSNFile, "postgres-dsn-file", "", "reference: explicit PostgreSQL connection file")
	flags.Int64Var(&config.PostgresFence, "postgres-fencing-token", 0, "reference: host-owned positive fencing token")
}

func validateSingleApplicationBackend(config singleApplicationConfig) error {
	if config.PostgresDSNFile == "" && config.PostgresFence == 0 {
		return nil
	}
	if !config.CompositionScenarios || config.PostgresDSNFile == "" || config.PostgresFence <= 0 {
		return fmt.Errorf("PostgreSQL requires -composition-scenarios, -postgres-dsn-file and a positive -postgres-fencing-token")
	}
	return nil
}
