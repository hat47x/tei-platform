package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

// issueGrantOptions is what the CLI needs to issue one grant.
type issueGrantOptions struct {
	Capability string
	Kind       string
	Access     string
	Scope      string
	Reason     string
	RunKey     string
	OutPath    string
	LogPath    string
}

// runIssueGrantCommand issues one grant for one run, records the issuance, and
// writes the grants file the adapters read. The issuance is recorded first: if it
// cannot be recorded, nothing is issued or written.
func runIssueGrantCommand(options issueGrantOptions) error {
	if options.Capability == "" || options.Scope == "" || options.OutPath == "" || options.LogPath == "" {
		return fmt.Errorf("issue-grant requires -grant-capability, -grant-scope, -grant-out, and -issue-log")
	}
	capability, err := core.ParseCanonicalID(options.Capability)
	if err != nil {
		return fmt.Errorf("grant capability %q: %w", options.Capability, err)
	}
	spec := permission.Spec{
		Kind:   permission.Kind(options.Kind),
		Access: permission.Access(options.Access),
		Scope:  options.Scope,
	}
	if spec.Kind == "" {
		spec.Kind = permission.KindFilesystem
	}
	if spec.Access == "" {
		spec.Access = permission.AccessWrite
	}

	log := access.IssueFile{Path: options.LogPath}
	if _, err := access.Issue(log, options.RunKey, capability, spec, options.Reason); err != nil {
		return err
	}

	document := grantsDocument{Grants: []grantedPermission{{
		CapabilityKind: "requested",
		CapabilityID:   string(capability),
		Kind:           string(spec.Kind),
		Access:         string(spec.Access),
		Scope:          spec.Scope,
	}}}
	raw, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(options.OutPath), 0o755); err != nil {
		return fmt.Errorf("prepare %s: %w", options.OutPath, err)
	}
	if err := os.WriteFile(options.OutPath, raw, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", options.OutPath, err)
	}
	return nil
}
