package multilingualoverlays

import (
	"errors"
	"fmt"

	"tei/internal/environment"
	"tei/internal/modelref"
	"tei/internal/plugin"
	resources "tei/reference/multilingual-resources"
)

// DisplayPluginIdentity is a stable UI serialization, independent from the
// runtime plugin.Identity Go JSON representation.
type DisplayPluginIdentity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// DisplayManifest is a read-only boundary between the selected TEI plugins
// and a standard UI. It is not a runtime grant or an editable definition.
type DisplayManifest struct {
	Kind            string                `json:"kind"`
	Environment     string                `json:"environment"`
	LockFingerprint string                `json:"lockFingerprint"`
	Target          DisplayPluginIdentity `json:"target"`
	Resource        resources.Key         `json:"resource"`
	Locale          string                `json:"locale"`
	Text            string                `json:"text"`
	SourceRevision  string                `json:"sourceRevision"`
	FromSource      bool                  `json:"fromSource"`
	Fallback        bool                  `json:"fallback"`
}

const DisplayManifestKind = "tei.multilingual.display-manifest.o2"

// BuildLockedDisplay binds a single standard display surface to a real
// environment Plugin Lock. Unlike O1's synthetic ResolvedSet, it verifies the
// exact plugin definitions and environment before composing the display
// resources. Trusted, already-accepted Surfaces/Grants are caller inputs:
// accepting them is outside this reference's authority.
func BuildLockedDisplay(
	catalog plugin.Catalog,
	env environment.Environment,
	lock plugin.Lock,
	graph modelref.Graph,
	surfaces []Surface,
	packs []Pack,
	grants []Grant,
	overrides []Override,
	request Lookup,
) (DisplayManifest, error) {
	resolved, err := catalog.ResolveEnvironmentWithLock(env, lock)
	if err != nil {
		return DisplayManifest{}, fmt.Errorf("plugin-lock: %w", err)
	}
	fingerprint, err := plugin.EnvironmentLockFingerprint(lock, env.ID)
	if err != nil {
		return DisplayManifest{}, fmt.Errorf("environment-lock: %w", err)
	}
	projection, diagnostic := Compose(Input{
		Resolved: resolved, Graph: graph, Surfaces: surfaces,
		Packs: packs, Grants: grants, Overrides: overrides,
	})
	if diagnostic != "" {
		return DisplayManifest{}, errors.New("overlay-composition: " + diagnostic)
	}
	result := projection.Resolve(request)
	if result.Diagnostic != "" {
		return DisplayManifest{}, errors.New("display-selection: " + result.Diagnostic)
	}
	if result.Text == "" || result.SourceRevision == "" {
		return DisplayManifest{}, errors.New("display-selection: empty-presentation")
	}
	return DisplayManifest{
		Kind: DisplayManifestKind, Environment: string(env.ID),
		LockFingerprint: fingerprint,
		Target:          DisplayPluginIdentity{ID: string(request.Target.ID), Version: request.Target.Version}, Resource: request.Key,
		Locale: result.Locale, Text: result.Text,
		SourceRevision: result.SourceRevision,
		FromSource:     result.FromSource, Fallback: result.Fallback,
	}, nil
}
