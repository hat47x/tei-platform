//go:build !tei_frozen_world && ((tei_order_only && !tei_repair_only) || (tei_repair_only && !tei_order_only))

package main

import (
	"tei/internal/ir"
	"tei/internal/workspace"
)

// Default first-Web reference keeps loading the real Workspace at startup.
func loadSingleApplicationWorld(path, _, _, _ string) (ir.World, error) {
	world, _, err := loadSingleApplicationSelection(path, "", "", "", "")
	return world, err
}

func loadSingleApplicationSelection(path, _, _, _, expected string) (ir.World, *scenarioModel, error) {
	loaded, err := workspace.Load(path)
	if err != nil {
		return ir.World{}, nil, err
	}
	var model *scenarioModel
	if expected != "" {
		model, err = bindScenarioModel(path, loaded, expected)
		if err != nil {
			return ir.World{}, nil, err
		}
	}
	return loaded.World, model, nil
}
