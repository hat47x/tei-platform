//go:build !tei_frozen_world

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// scenarioModel is a host-selected source revision. It never accepts a proposal
// or interprets a decision ledger as authority. Shared Acceptance stays the only
// reference source writer; the host independently admits the selected revision.
type scenarioModel struct {
	root                string
	revision            string
	manifestPath        string
	manifestFingerprint string
}

func bindScenarioModel(root string, loaded workspace.Result, expected string) (*scenarioModel, error) {
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return nil, err
	}
	if expected == "" || sourceindex.Revision(index) != expected {
		return nil, errors.New("host-selected model revision does not match loaded source")
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			return nil, errors.New("host-selected model contains validation errors")
		}
	}
	manifestPath, manifestFingerprint, err := loaded.ManifestSource()
	if err != nil {
		return nil, err
	}
	model := &scenarioModel{root: root, revision: expected, manifestPath: filepath.Join(root, manifestPath), manifestFingerprint: manifestFingerprint}
	if err := model.Check(context.Background()); err != nil {
		return nil, err
	}
	return model, nil
}

// Check stays outside the kernel's nonblocking guard. The parsed World remains
// fixed for the host lifetime; source changes refuse new calls, never live-reload
// a partially edited model. This is not atomic with an arbitrary filesystem writer.
func (m *scenarioModel) Check(ctx context.Context) error {
	if m == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	manifest, err := os.ReadFile(m.manifestPath)
	if err != nil || sourceindex.Fingerprint(manifest) != m.manifestFingerprint {
		return errors.New("workspace manifest changed after model selection")
	}
	loaded, err := workspace.Load(m.root)
	if err != nil {
		return errors.New("selected model source is unavailable or invalid")
	}
	manifestPath, manifestFingerprint, err := loaded.ManifestSource()
	if err != nil || filepath.Join(m.root, manifestPath) != m.manifestPath || manifestFingerprint != m.manifestFingerprint {
		return errors.New("workspace manifest changed during model verification")
	}
	index, err := workspace.BuildSourceIndex(m.root, loaded)
	if err != nil || sourceindex.Revision(index) != m.revision {
		return errors.New("selected model source revision is stale; explicit reload and host admission required")
	}
	return ctx.Err()
}
