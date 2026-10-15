package driverscenarios

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"

	"tei/internal/driver"
	"tei/internal/scenariotest"
)

// CheckCLIWebBrowseDerivations re-derives the CLI argv structure and the
// single supported Web Browse case from the same persisted Scenario source.
// Old, partially regenerated or hand-edited derived output is not accepted.
// A successful check qualifies only this synthetic Browse projection. It is
// not an attestation of target build, authority, browser or Host transport.
func CheckCLIWebBrowseDerivations(
	scenarioSource, fixtureSource []byte,
	scenarioPath, workspacePath string,
	cli scenariotest.Plan,
	web WebBrowseCase,
	resolver driver.Resolver,
	binding WebBrowseBinding,
) error {
	scenario, err := scenariotest.Parse(scenarioSource)
	if err != nil {
		return fmt.Errorf("parse current Scenario: %w", err)
	}
	expectedCLI, err := scenariotest.Generate(
		scenario, scenarioSource, scenarioPath, workspacePath,
	)
	if err != nil {
		return fmt.Errorf("regenerate CLI from current Scenario: %w", err)
	}
	if cli.SchemaVersion != expectedCLI.SchemaVersion ||
		cli.ScenarioID != expectedCLI.ScenarioID ||
		cli.ScenarioSHA256 != expectedCLI.ScenarioSHA256 ||
		len(cli.Commands) != len(expectedCLI.Commands) {
		return fmt.Errorf("stale or incomplete derived CLI plan")
	}
	// The CLI runner is responsible for its authoritative workspace/lock
	// comparison. This Reference checks only their explicit pin structure,
	// not whether these values are trusted revision attestations.
	if (cli.WorldSHA256 == "") != (cli.PluginLockSHA256 == "") {
		return fmt.Errorf("partial CLI World/Plugin Lock pins")
	}
	for i, expected := range expectedCLI.Commands {
		actual := cli.Commands[i]
		want := append([]string(nil), expected.Argv...)
		if cli.WorldSHA256 != "" {
			want = append(want,
				"-world-sha256", cli.WorldSHA256,
				"-plugin-lock-sha256", cli.PluginLockSHA256,
			)
		}
		if digest, ok := cli.FixtureSHA256[expected.StepID]; ok {
			want = append(want, "-fixture-sha256", digest)
		}
		if actual.StepID != expected.StepID ||
			!reflect.DeepEqual(actual.Argv, want) {
			return fmt.Errorf("stale or modified CLI argv at Step %s", expected.StepID)
		}
	}
	// No Web result may use a fixture that differs from the one pinned by
	// the CLI plan for the same Browse Step.
	digest := sha256.Sum256(fixtureSource)
	if cli.FixtureSHA256[binding.StepID] != hex.EncodeToString(digest[:]) {
		return fmt.Errorf("stale or missing Browse fixture pin")
	}
	expectedWeb, err := GenerateWebBrowseCase(
		scenarioSource, fixtureSource, cli, resolver, binding,
	)
	if err != nil {
		return fmt.Errorf("regenerate Web Browse from current Scenario: %w", err)
	}
	if !reflect.DeepEqual(web, expectedWeb) {
		return fmt.Errorf("stale or modified derived Web Browse case: regenerate from Scenario")
	}
	return nil
}
