//go:build !tei_frozen_world && ((tei_order_only && !tei_repair_only) || (tei_repair_only && !tei_order_only))

package main

import (
    "tei/internal/ir"
    "tei/internal/workspace"
)

// Default first-Web reference keeps loading the real Workspace at startup.
func loadSingleApplicationWorld(path, _, _, _ string) (ir.World, error) {
    loaded, err := workspace.Load(path)
    if err != nil {
        return ir.World{}, err
    }
    return loaded.World, nil
}
