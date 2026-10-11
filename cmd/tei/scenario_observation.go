package main

import (
	"context"
	"fmt"
	"reflect"

	"tei/internal/driver"
	"tei/internal/scenariotest"
)

func executeScenarioObservation(step scenariotest.Step, fixture scenariotest.ObservationFixture) (any, error) {
	if fixture.ReadIdentity != step.ReadIdentity {
		return nil, fmt.Errorf("Observation read identity differs from Scenario")
	}
	read := func(context.Context) (any, error) {
		return fixture.Observed.Decode()
	}
	observation, err := driver.ExecuteObservationRead(context.Background(), fixture.ReadIdentity, driver.ObservationEvidenceSynthetic, read)
	if err != nil {
		return nil, err
	}
	want, err := step.ExpectObservation.Decode()
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(observation.Result, want) {
		return nil, fmt.Errorf("synthetic Observation differs from Scenario expectation")
	}
	return observation.Result, nil
}
