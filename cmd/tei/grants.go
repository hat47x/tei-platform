package main

import (
	"fmt"
	"sort"

	"tei/internal/access"
	"tei/internal/environment"
	"tei/internal/workspace"
)

// environmentIssuerPrefix names the party that decides a declared grant. The
// grant in a file `tei grants` writes comes from an accepted environment
// declaration, so the issuer is that environment and not the tool that reads it.
const environmentIssuerPrefix = "environment:"

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

	// Stdout is the "what would this write" view; a file is the artifact an
	// adapter reads. Both come from the same encoder, so the two cannot drift.
	if outPath == "" {
		raw, err := access.EncodeDocument(document)
		if err != nil {
			return err
		}
		_, err = fmt.Print(string(raw))
		return err
	}
	return access.WriteDocument(outPath, document)
}

// buildGrantsDocument collects the grants an application's environments declare
// and orders them so the file is stable between runs. Each entry records the
// environment that declared it, so a reader can tell a declared grant from one
// a capability wrote for itself.
func buildGrantsDocument(declared []environment.Environment, applicationID string) access.Document {
	document := access.Document{Application: applicationID, Grants: []access.Entry{}}
	for _, env := range declared {
		for _, grant := range env.PermissionGrants {
			document.Grants = append(document.Grants, access.Entry{
				IssuedBy:       environmentIssuerPrefix + string(env.ID),
				CapabilityKind: grant.CapabilityKind,
				CapabilityID:   grant.CapabilityID,
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
