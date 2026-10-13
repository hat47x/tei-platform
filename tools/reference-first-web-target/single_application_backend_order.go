//go:build tei_order_only && !tei_repair_only

package main

import (
	"flag"
	"fmt"
)

type singleApplicationBackend struct{ nonScreen nonScreenFlags }

func (b singleApplicationBackend) servesHTTP() bool { return !b.nonScreen.enabled() }

func registerSingleApplicationBackendFlags(flags *flag.FlagSet, config *singleApplicationConfig) {
	flags.StringVar(&config.PostgresDSNFile, "postgres-dsn-file", "", "reference: explicit PostgreSQL connection file")
	flags.Int64Var(&config.PostgresFence, "postgres-fencing-token", 0, "reference: host-owned positive fencing token")
	config.Backend.nonScreen = registerNonScreenFlags(flags)
}

func validateSingleApplicationBackend(config singleApplicationConfig) error {
	if config.PostgresDSNFile == "" && config.PostgresFence == 0 && config.Backend.servesHTTP() {
		return nil
	}
	if !config.CompositionScenarios || config.PostgresDSNFile == "" || config.PostgresFence <= 0 {
		return fmt.Errorf("PostgreSQL requires -composition-scenarios, -postgres-dsn-file and a positive -postgres-fencing-token")
	}
	if config.Backend.nonScreen.enabled() && *config.Backend.nonScreen.mode != "report" && *config.Backend.nonScreen.mode != "batch" && *config.Backend.nonScreen.mode != "batch-observe" {
		return fmt.Errorf("-non-screen must be report, batch or batch-observe")
	}
	return nil
}
