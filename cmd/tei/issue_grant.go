package main

import (
	"fmt"
	"os"

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
	Issuer     string
	Reason     string
	RunKey     string
	OutPath    string
	LogPath    string

	// DeclarationPath is the grants file the environment declared (`tei grants`).
	// When it is set, the issuance must stay inside it; when it is not, the
	// issuance is ungrounded and the CLI says so.
	DeclarationPath string
	// DeclaredScope is the name the declaration uses for the place the run writes
	// to, and ResolutionRoot is that place. Both are needed to map a logical name
	// onto a concrete path; one without the other is refused.
	DeclaredScope  string
	ResolutionRoot string
}

// runIssueGrantCommand issues one grant for one run, records the issuance, and
// writes the grants file the adapters read. The issuance is recorded first: if it
// cannot be recorded, nothing is issued or written.
//
// When -grant-declaration is given, the issuance must stay inside what the
// environment declared. That check is the difference between "a grant exists" and
// "a grant exists because the environment allows it".
func runIssueGrantCommand(options issueGrantOptions) error {
	if options.Capability == "" || options.Scope == "" || options.Issuer == "" ||
		options.Reason == "" || options.OutPath == "" || options.LogPath == "" {
		return fmt.Errorf("issue-grant requires -grant-capability, -grant-scope, -grant-issuer, -grant-reason, -grant-out, and -issue-log")
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

	document := access.Document{Grants: []access.Entry{{
		IssuedBy:       options.Issuer,
		CapabilityKind: "requested",
		CapabilityID:   capability,
		Kind:           string(spec.Kind),
		Access:         string(spec.Access),
		Scope:          spec.Scope,
	}}}

	resolutions, err := groundingResolutions(options, capability)
	if err != nil {
		return err
	}
	declared, err := declaredGrants(options.DeclarationPath)
	if err != nil {
		return err
	}
	if options.DeclarationPath != "" {
		if err := access.Ground(document, declared, resolutions, capability); err != nil {
			return err
		}
	} else {
		// An ungrounded issuance is still recorded, because a run may legitimately
		// need access no declaration covers yet. It is not silent: the caller is
		// told that nothing ties this grant to what the environment allows.
		fmt.Fprintf(os.Stderr, "注意: -grant-declaration が無いため、%s への付与は環境の宣言に照らされていません（記録には解像が残りません）\n", spec.Scope)
	}

	log := access.IssueFile{Path: options.LogPath}
	if _, err := access.IssueResolved(log, access.IssueRequest{
		RunKey:      options.RunKey,
		Issuer:      options.Issuer,
		Capability:  capability,
		Permission:  spec,
		Basis:       options.Reason,
		Resolutions: resolutions,
	}); err != nil {
		return err
	}
	return access.WriteDocument(options.OutPath, document)
}

// declaredGrants reads the declaration file, if one was named. The environment
// declares the place by a logical name and records who declared it, so the file
// comes from `tei grants` rather than from the capability that wants to use it.
func declaredGrants(path string) (access.Document, error) {
	if path == "" {
		return access.Document{}, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return access.Document{}, fmt.Errorf("read declaration %s: %w", path, err)
	}
	document, err := access.ParseDocument(raw)
	if err != nil {
		return access.Document{}, fmt.Errorf("declaration %s: %w", path, err)
	}
	return document, nil
}

// groundingResolutions builds the mapping from the declared scope name to the
// place this run uses. Both halves are required together: a name without a place
// maps nothing, and a place without a name cannot be traced to the declaration.
func groundingResolutions(options issueGrantOptions, capability core.CanonicalID) ([]access.Resolution, error) {
	hasScope := options.DeclaredScope != ""
	hasRoot := options.ResolutionRoot != ""
	if !hasScope && !hasRoot {
		return nil, nil
	}
	if !hasScope || !hasRoot {
		return nil, fmt.Errorf("a resolution needs both -grant-declared-scope and -grant-resolved-root; one without the other maps nothing")
	}
	if options.DeclarationPath == "" {
		return nil, fmt.Errorf("-grant-declared-scope names a declared scope, so -grant-declaration must also be given")
	}
	return []access.Resolution{{
		ResolvedBy: options.Issuer,
		Declared:   options.DeclaredScope,
		Root:       options.ResolutionRoot,
	}}, nil
}
