package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"tei/internal/environment"
	"tei/internal/workspace"
)

// grantedPermission is one grant in the file the guarded adapters read.
type grantedPermission struct {
	CapabilityKind string `json:"capabilityKind,omitempty"`
	CapabilityID   string `json:"capabilityId,omitempty"`
	Kind           string `json:"kind"`
	Access         string `json:"access"`
	Scope          string `json:"scope"`
}

// grantsDocument is the file shape: a plain list, so a tool does not depend on
// the workspace model to read it.
type grantsDocument struct {
	Application string              `json:"application,omitempty"`
	Grants      []grantedPermission `json:"grants"`
}

// runGrantsCommand writes the permission grants an application's environments
// declare. It reads the accepted declarations and writes a file; it does not
// decide anything about who may use them.
func runGrantsCommand(workspacePath, applicationID, outPath string) error {
	if workspacePath == "" {
		return fmt.Errorf("grants requires -workspace")
	}
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		return err
	}

	document := buildGrantsDocument(loaded.World.Environments, applicationID)

	raw, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if outPath == "" {
		_, err = os.Stdout.Write(raw)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("prepare %s: %w", outPath, err)
	}
	if err := os.WriteFile(outPath, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	return nil
}

// buildGrantsDocument collects the grants an application's environments declare
// and orders them so the file is stable between runs.
func buildGrantsDocument(declared []environment.Environment, applicationID string) grantsDocument {
	document := grantsDocument{Application: applicationID}
	for _, env := range declared {
		for _, grant := range env.PermissionGrants {
			document.Grants = append(document.Grants, grantedPermission{
				CapabilityKind: grant.CapabilityKind,
				CapabilityID:   string(grant.CapabilityID),
				Kind:           string(grant.Permission.Kind),
				Access:         string(grant.Permission.Access),
				Scope:          grant.Permission.Scope,
			})
		}
	}
	sort.Slice(document.Grants, func(i, j int) bool {
		a, b := document.Grants[i], document.Grants[j]
		if a.CapabilityID != b.CapabilityID {
			return a.CapabilityID < b.CapabilityID
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Scope != b.Scope {
			return a.Scope < b.Scope
		}
		return a.Access < b.Access
	})
	return document
}
