package multilingualoverlays

// O2b binds an ordinary Workspace model to a display-only localization. It
// does not introduce another acceptance or authorization authority.

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/modelref/sourcebridge"
	"tei/internal/plugin"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// SourceWitness identifies the definition bytes used for the localized label.
// This is provenance of a parsed Workspace, not an acceptance certificate.
type SourceWitness struct {
	Subject             modelref.Key `json:"subject"`
	ArtifactPath        string       `json:"artifactPath"`
	ArtifactFingerprint string       `json:"artifactFingerprint"`
	Original            string       `json:"original"`
	OriginalFingerprint string       `json:"originalFingerprint"`
	LockFingerprint     string       `json:"lockFingerprint"`
}

// WorkspaceDisplayConfig selects an existing Data declaration and an already
// pinned environment. The ordinary plugin is unrelated to language services.
type WorkspaceDisplayConfig struct {
	Environment core.CanonicalID
	Scope       string
	Data        core.CanonicalID
	Owner       plugin.Identity
	SourceLocale string
	Locale      string
	Packs       []Pack
	Grants      []Grant
	Overrides   []Override
}

// ReviewRequest is passed to an existing host's authorization/acceptance
// boundary. The verifier must inspect exact SourceWitness, lock, provider and
// granted content; returning nil is a trust decision made OUTSIDE this package.
type ReviewRequest struct {
	Environment core.CanonicalID
	Source      SourceWitness
	Surface     Surface
	Packs       []Pack
	Grants      []Grant
	Overrides   []Override
	Requested   Lookup
}

// ReviewGate must be backed by the existing host policy/acceptance mechanism.
// A test stub is not production authorization. A missing gate fails closed.
type ReviewGate func(ReviewRequest) error

func sourceDataLabel(loaded workspace.Result, id core.CanonicalID) (string, bool) {
	found := false
	var label string
	for _, ns := range loaded.World.Semantics {
		for _, d := range ns.Data {
			if d.ID != id {
				continue
			}
			if found || strings.TrimSpace(d.Symbol) == "" {
				return "", false
			}
			found, label = true, d.Symbol
		}
	}
	return label, found
}

func exactEnvironment(loaded workspace.Result, id core.CanonicalID) (environment.Environment, error) {
	var found *environment.Environment
	for i := range loaded.World.Environments {
		if loaded.World.Environments[i].ID != id {
			continue
		}
		if found != nil {
			return environment.Environment{}, fmt.Errorf("duplicate environment %s", id)
		}
		found = &loaded.World.Environments[i]
	}
	if found == nil {
		return environment.Environment{}, fmt.Errorf("unknown environment %s", id)
	}
	return *found, nil
}

// BuildWorkspaceDisplay is the O2b authoring-side adapter: it loads *actual*
// .tei Source through the existing Workspace loader, validates that World,
// checks an indexed Source anchor tied to the parsed bytes, requires the
// caller's external acceptance decision, then applies the O2a Plugin Lock
// and E1 display-selection mechanisms. It never changes Source or World.
func BuildWorkspaceDisplay(root string, lock plugin.Lock, cfg WorkspaceDisplayConfig, gate ReviewGate) (DisplayManifest, SourceWitness, error) {
	reject := func(message string) (DisplayManifest, SourceWitness, error) {
		return DisplayManifest{}, SourceWitness{}, fmt.Errorf("workspace-display: %s", message)
	}
	if gate == nil {
		return reject("review-gate-required")
	}
	if cfg.Scope == "" || cfg.Data == "" || cfg.SourceLocale == "" || cfg.Locale == "" {
		return reject("incomplete-binding")
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return reject("workspace-load: " + err.Error())
	}
	if validation.HasErrors(validation.Validate(loaded.World)) {
		return reject("invalid-world")
	}
	env, err := exactEnvironment(loaded, cfg.Environment)
	if err != nil {
		return reject(err.Error())
	}
	// Existing Derived Index supplies the typed node identity. It does not
	// provide acceptance authority or an author-approved localized label.
	key := modelref.Key{Scope: cfg.Scope, Subject: subject.Subject{Kind: "semantic-data", ID: string(cfg.Data)}}
	derived := derivedbridge.FromIndex(index.Build(loaded.World), cfg.Scope)
	var present bool
	for _, n := range derived.Graph.Nodes {
		if n.Key == key {
			present = true
			break
		}
	}
	if !present {
		return reject("data-not-in-typed-index")
	}
	label, found := sourceDataLabel(loaded, cfg.Data)
	if !found {
		return reject("data-label-unavailable")
	}
	idx, reader, err := workspace.IndexedLoadedSnapshot(root, loaded)
	if err != nil {
		return reject("source-index: " + err.Error())
	}
	resolver, err := sourcebridge.NewResolver(derived.Graph, idx, reader)
	if err != nil {
		return reject("source-resolver: " + err.Error())
	}
	nav := resolver.Locate(key)
	if nav.Status != "located" {
		return reject("source-anchor: " + nav.Reason)
	}
	// A display source must have one unambiguous definition. Related
	// observations and derived references cannot stand in for that definition.
	var path, fingerprint string
	for _, reference := range nav.References {
		if reference.Role != sourceindex.RoleDefinition {
			continue
		}
		for _, artifact := range nav.Artifacts {
			if artifact.Path == reference.ArtifactPath {
				if path != "" {
					return reject("ambiguous-source-definition")
				}
				path, fingerprint = artifact.Path, artifact.Fingerprint
			}
		}
	}
	if path == "" || fingerprint == "" {
		return reject("source-definition-unavailable")
	}
	catalog := plugin.Catalog{Plugins: loaded.World.Plugins, Bundles: loaded.World.Bundles}
	// Ensure the ReviewGate never sees a stale/foreign plugin lock.
	if _, err := catalog.ResolveEnvironmentWithLock(env, lock); err != nil {
		return reject("plugin-lock: " + err.Error())
	}
	lockFingerprint, err := plugin.EnvironmentLockFingerprint(lock, env.ID)
	if err != nil {
		return reject("plugin-lock-fingerprint: " + err.Error())
	}
	witness := SourceWitness{
		Subject: key, ArtifactPath: path, ArtifactFingerprint: fingerprint,
		Original: label, OriginalFingerprint: Fingerprint(label),
		LockFingerprint: lockFingerprint,
	}
	surface := Surface{Target: cfg.Owner, Subject: key, Field: "display-name", SourceLocale: cfg.SourceLocale}
	resource := resourceKey(surface)
	request := Lookup{Target: cfg.Owner, Key: resource, Locale: cfg.Locale}
	packs := append([]Pack(nil), cfg.Packs...)
	grants := append([]Grant(nil), cfg.Grants...)
	overrides := append([]Override(nil), cfg.Overrides...)
	// The verifier is authoritative only if supplied by the trusted host.
	// A caller cannot mistake a matching digest for acceptance.
	if err := gate(ReviewRequest{
		Environment: env.ID, Source: witness, Surface: surface,
		Packs: packs, Grants: grants, Overrides: overrides, Requested: request,
	}); err != nil {
		return reject("review-denied: " + err.Error())
	}
	// Only the observed, source-bound node is projected as a display facet;
	// all other canonical meaning and relationships remain untouched.
	graph := modelref.Graph{
		Schema: "tei.modelref.v0",
		Nodes: []modelref.Node{{Key: key, DisplayName: label}},
	}
	manifest, err := BuildLockedDisplay(catalog, env, lock, graph,
		[]Surface{surface}, packs, grants, overrides, request)
	if err != nil {
		return reject(err.Error())
	}
	return manifest, witness, nil
}
