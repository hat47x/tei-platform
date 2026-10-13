//go:build tei_order_only && !tei_repair_only

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"tei/internal/composition"
	"tei/internal/ir"
)

func main() {
	err := runSingleApplicationTarget("order_entry", orderApplicationRunners)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func orderApplicationRunners(world ir.World, config singleApplicationConfig) (
	readScenarioRunner, commandScenarioRunner, []webTargetOption, error,
) {
	if err := validateSingleApplicationBackend(config); err != nil {
		return nil, nil, nil, err
	}
	command, err := newReferenceCommandScenarioRunner(
		world, config.LegacyUsers, config.LegacyAuthorization,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	command.model = config.Model
	var read readScenarioRunner = newReferenceReadScenarioRunner(world)
	if config.CompositionScenarios {
		digest, err := runningExecutableDigest()
		if err != nil {
			return nil, nil, nil, err
		}
		submitContract := submitServiceContract
		var state *postgresScenarioState
		if config.PostgresDSNFile != "" {
			if config.OnClose == nil {
				return nil, nil, nil, fmt.Errorf("PostgreSQL requires a host-owned connection lifetime")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			db, err := openScenarioPostgres(ctx, config.PostgresDSNFile)
			cancel()
			if err != nil {
				return nil, nil, nil, err
			}
			config.OnClose(func() { _ = db.Close() })
			state = &postgresScenarioState{db: db, world: world, digest: digest, fence: config.PostgresFence, model: config.Model}
			read = &postgresReadScenario{state: state}
			command.persistence = state
			submitContract = postgresSubmitServiceContract
		}
		registry := composition.NewRegistry()
		r, c, err := wireComposedScenarios(context.Background(), registry, read, command, digest,
			referenceInstanceGuard(digest, map[string]composition.Contract{"order-read": readServiceContract, "order-submit": submitContract}))
		if err == nil && config.Backend.nonScreen.enabled() {
			err = config.Backend.nonScreen.run(registry, r, command, state, config.WorkspacePath)
			return nil, nil, nil, err
		}
		return r, c, nil, err
	}
	return read, command, nil, nil
}
