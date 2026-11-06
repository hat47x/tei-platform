package main

import (
	"fmt"
	"path/filepath"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

// lineageCapability is the identity the tool acts as when it writes its
// observation. The grant is the exact path the caller asked for, so the tool
// cannot write anywhere else; moving the grant to the environment is the next
// step.
const lineageCapability = core.CanonicalID("tei.frozen-world.lineage")

// writeLineage writes the lineage observation through the guarded filesystem.
func writeLineage(path string, data []byte) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", path, err)
	}
	files := access.FileSystem{
		Root:       filepath.Dir(absolute),
		Capability: lineageCapability,
		Guard: access.Guard{Granted: []permission.Spec{{
			Kind:   permission.KindFilesystem,
			Access: permission.AccessWrite,
			Scope:  absolute,
		}}},
	}
	return files.WriteFile(filepath.Base(absolute), data, 0o600)
}
