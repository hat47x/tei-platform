package main

import (
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

// lineageCapability is the identity the tool acts as when it writes its
// observation. The grant names the scope the caller asked for, so the tool
// cannot write anywhere else; moving the grant to the environment is the next
// step.
const lineageCapability = core.CanonicalID("tei.frozen_world.lineage")

// loadGrants reads the grants file through the shared parser, so the tool and
// `tei grants` keep one format, and refuses a file the tool itself issued. A
// capability that can write the grant it is checked against has no check: the
// refusal is what makes the issuer field mean something.
//
// When a declaration is supplied, the grants must also stay inside it. That is
// what turns "a grant exists" into "a grant the environment allows". The check
// is skipped when no declaration is given, because the reference workspace does
// not declare this capability yet; that gap is recorded in the design memo.
func loadGrants(path, declarationPath string) ([]permission.Spec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read grants %s: %w", path, err)
	}
	document, err := access.ParseDocument(raw)
	if err != nil {
		return nil, fmt.Errorf("grants %s: %w", path, err)
	}
	if err := access.AcceptIssuedBy(document, lineageCapability); err != nil {
		return nil, fmt.Errorf("grants %s: %w", path, err)
	}
	if declarationPath != "" {
		declaredRaw, err := os.ReadFile(declarationPath)
		if err != nil {
			return nil, fmt.Errorf("read declaration %s: %w", declarationPath, err)
		}
		declared, err := access.ParseDocument(declaredRaw)
		if err != nil {
			return nil, fmt.Errorf("declaration %s: %w", declarationPath, err)
		}
		// The resolutions that tie a declared name to this run's place are the
		// caller's claim, not the tool's: the tool only checks that the grant
		// stays inside the declaration as the caller mapped it.
		if err := access.Ground(document, declared, nil, lineageCapability); err != nil {
			return nil, fmt.Errorf("grants %s against declaration %s: %w", path, declarationPath, err)
		}
	}
	grants := make([]permission.Spec, 0, len(document.Grants))
	for _, entry := range document.Grants {
		grants = append(grants, permission.Spec{
			Kind:   permission.Kind(entry.Kind),
			Access: permission.Access(entry.Access),
			Scope:  entry.Scope,
		})
	}
	return grants, nil
}

// writeLineageWithGrants writes the observation using the supplied grants. The
// requested path must be covered by them; the guarded filesystem checks it.
func writeLineageWithGrants(path string, data []byte, grants []permission.Spec) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", path, err)
	}
	if len(grants) == 0 {
		return fmt.Errorf("no grants were supplied, so %s is not written", absolute)
	}
	files := access.FileSystem{
		Root:       filepath.Dir(absolute),
		Capability: lineageCapability,
		Guard:      access.Guard{Granted: grants},
	}
	return files.WriteFile(filepath.Base(absolute), data, 0o600)
}
