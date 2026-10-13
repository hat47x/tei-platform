//go:build tei_frozen_world

package main

import (
	"context"
	"errors"
	"tei/internal/ir"
)

// Frozen World identity is a builder-pinned digest, not a live Source revision.
// Do not import the Source loader or invent a source-revision authority here.
type scenarioModel struct{ revision string }

func (m *scenarioModel) Check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if m != nil {
		return errors.New("frozen reference cannot bind a live source model")
	}
	return nil
}
func loadSingleApplicationSelection(path, b, r, t, expected string) (ir.World, *scenarioModel, error) {
	if expected != "" {
		return ir.World{}, nil, errors.New("-model-revision is not supported by a frozen World; use the builder-pinned World digest")
	}
	world, err := loadSingleApplicationWorld(path, b, r, t)
	return world, nil, err
}
