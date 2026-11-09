package multilingualoverlays

// O3a is a deterministic, read-only distribution *candidate*. Its digest
// detects accidental changes; neither it nor the Plugin Lock authenticates
// the publisher, a Host grant, or an accepted Workspace revision.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/environment"
	"tei/internal/modelref"
	"tei/internal/plugin"
	resources "tei/reference/multilingual-resources"
)

const LocaleBundleKind = "tei.multilingual.locale-bundle.o3"

// LocaleBundleRequest explicitly selects one ordinary plugin, typed display
// resources, and output languages. No implicit fallback or locale inference.
type LocaleBundleRequest struct {
	Target  plugin.Identity
	Keys    []resources.Key
	Locales []string
}

// LocaleBundle contains only the requested display text, not the full
// translation catalog, plugin execution metadata, or business data.
type LocaleBundle struct {
	Kind            string                `json:"kind"`
	Environment     string                `json:"environment"`
	LockFingerprint string                `json:"lockFingerprint"`
	Target          DisplayPluginIdentity `json:"target"`
	Locales         []string              `json:"locales"`
	Displays        []DisplayManifest     `json:"displays"`
	Digest          string                `json:"digest,omitempty"`
}

func lessDisplayKey(a, b resources.Key) bool {
	if a.Owner != b.Owner {
		return a.Owner < b.Owner
	}
	if a.Target.Scope != b.Target.Scope {
		return a.Target.Scope < b.Target.Scope
	}
	if a.Target.Subject.Kind != b.Target.Subject.Kind {
		return a.Target.Subject.Kind < b.Target.Subject.Kind
	}
	if a.Target.Subject.ID != b.Target.Subject.ID {
		return a.Target.Subject.ID < b.Target.Subject.ID
	}
	return a.Field < b.Field
}

// localeBundleDigest hashes JSON of the canonical bundle without the digest
// field itself. Consumers must compare against a separately trusted expected
// digest; a digest embedded with its own payload is not authentication.
func localeBundleDigest(b LocaleBundle) (string, error) {
	b.Digest = ""
	payload, err := json.Marshal(b)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// BuildLockedLocaleBundle resolves the real environment lock exactly once,
// composes the existing O1/E1 display projection, and exports only the named
// languages and resource keys in deterministic order. A missing or unreviewed
// required translation blocks the whole bundle, rather than silently shipping
// a partially localized UI.
func BuildLockedLocaleBundle(
	catalog plugin.Catalog,
	env environment.Environment,
	lock plugin.Lock,
	graph modelref.Graph,
	surfaces []Surface,
	packs []Pack,
	grants []Grant,
	overrides []Override,
	req LocaleBundleRequest,
) (LocaleBundle, error) {
	fail := func(message string) (LocaleBundle, error) {
		return LocaleBundle{}, fmt.Errorf("locale-bundle: %s", message)
	}
	if req.Target.ID == "" || req.Target.Version == "" || len(req.Keys) == 0 || len(req.Locales) == 0 {
		return fail("incomplete-selection")
	}
	resolved, err := catalog.ResolveEnvironmentWithLock(env, lock)
	if err != nil {
		return fail("plugin-lock: " + err.Error())
	}
	if !selected(resolved, req.Target) {
		return fail("unselected-target")
	}
	fingerprint, err := plugin.EnvironmentLockFingerprint(lock, env.ID)
	if err != nil {
		return fail("environment-lock: " + err.Error())
	}
	projection, diagnostic := Compose(Input{
		Resolved: resolved, Graph: graph, Surfaces: surfaces,
		Packs: packs, Grants: grants, Overrides: overrides,
	})
	if diagnostic != "" {
		return fail("overlay-composition: " + diagnostic)
	}
	keys := append([]resources.Key(nil), req.Keys...)
	locales := append([]string(nil), req.Locales...)
	sort.Slice(keys, func(i, j int) bool { return lessDisplayKey(keys[i], keys[j]) })
	sort.Strings(locales)
	for i, key := range keys {
		if key.Owner != string(req.Target.ID) {
			return fail("foreign-resource-owner")
		}
		if i > 0 && key == keys[i-1] {
			return fail("duplicate-resource")
		}
	}
	for i, locale := range locales {
		if locale == "" || (i > 0 && locale == locales[i-1]) {
			return fail("duplicate-or-empty-locale")
		}
	}
	out := LocaleBundle{
		Kind: LocaleBundleKind, Environment: string(env.ID),
		LockFingerprint: fingerprint,
		Target:          DisplayPluginIdentity{ID: string(req.Target.ID), Version: req.Target.Version},
		Locales:         locales,
		Displays:        make([]DisplayManifest, 0, len(keys)*len(locales)),
	}
	// Locale first, then typed resource: stable regardless of authoring order.
	for _, locale := range locales {
		for _, key := range keys {
			result := projection.Resolve(Lookup{Target: req.Target, Key: key, Locale: locale})
			if result.Diagnostic != "" {
				return fail("display-selection: " + result.Diagnostic)
			}
			if result.Text == "" || result.SourceRevision == "" || result.Fallback || result.Locale != locale {
				return fail("invalid-exact-display")
			}
			out.Displays = append(out.Displays, DisplayManifest{
				Kind: DisplayManifestKind, Environment: out.Environment,
				LockFingerprint: out.LockFingerprint, Target: out.Target,
				Resource: key, Locale: locale, Text: result.Text,
				SourceRevision: result.SourceRevision,
				FromSource:     result.FromSource, Fallback: false,
			})
		}
	}
	out.Digest, err = localeBundleDigest(out)
	if err != nil {
		return fail("digest: " + err.Error())
	}
	return out, nil
}

// CheckLocaleBundleIntegrity validates local manifest consistency and the
// self-contained content digest. It is NOT verification of signing identity,
// Host authority, accepted Source, or trusted delivery.
func CheckLocaleBundleIntegrity(bundle LocaleBundle) error {
	if bundle.Kind != LocaleBundleKind || bundle.Environment == "" ||
		bundle.LockFingerprint == "" || bundle.Target.ID == "" ||
		bundle.Target.Version == "" || len(bundle.Locales) == 0 || len(bundle.Displays) == 0 {
		return fmt.Errorf("locale-bundle: invalid-header")
	}
	if len(bundle.Displays)%len(bundle.Locales) != 0 {
		return fmt.Errorf("locale-bundle: incomplete-display-matrix")
	}
	perLocale := len(bundle.Displays) / len(bundle.Locales)
	for i, locale := range bundle.Locales {
		if locale == "" || (i > 0 && locale <= bundle.Locales[i-1]) {
			return fmt.Errorf("locale-bundle: invalid-locale-order")
		}
		for j := 0; j < perLocale; j++ {
			d := bundle.Displays[i*perLocale+j]
			if d.Kind != DisplayManifestKind || d.Environment != bundle.Environment ||
				d.LockFingerprint != bundle.LockFingerprint || d.Target != bundle.Target ||
				d.Locale != locale || d.Resource.Owner != bundle.Target.ID ||
				d.Text == "" || d.SourceRevision == "" || d.Fallback {
				return fmt.Errorf("locale-bundle: invalid-display")
			}
			if i > 0 && d.Resource != bundle.Displays[j].Resource {
				return fmt.Errorf("locale-bundle: different-resource-sets")
			}
			if j > 0 && !lessDisplayKey(bundle.Displays[i*perLocale+j-1].Resource, d.Resource) {
				return fmt.Errorf("locale-bundle: invalid-resource-order")
			}
		}
	}
	expected, err := localeBundleDigest(bundle)
	if err != nil {
		return err
	}
	if bundle.Digest != expected {
		return fmt.Errorf("locale-bundle: digest-mismatch")
	}
	return nil
}

// CheckLocaleBundleAgainstLock binds a structurally consistent bundle to the
// Host's independently supplied current catalog, environment, and lock. This
// catches stale/different plugin selections at a distribution boundary. It
// still cannot prove the received label matches accepted Source or grant
// decisions: callers must obtain those from trusted Host mechanisms.
func CheckLocaleBundleAgainstLock(
	bundle LocaleBundle,
	catalog plugin.Catalog,
	env environment.Environment,
	lock plugin.Lock,
) error {
	if err := CheckLocaleBundleIntegrity(bundle); err != nil {
		return err
	}
	resolved, err := catalog.ResolveEnvironmentWithLock(env, lock)
	if err != nil {
		return fmt.Errorf("locale-bundle: plugin-lock: %w", err)
	}
	fingerprint, err := plugin.EnvironmentLockFingerprint(lock, env.ID)
	if err != nil {
		return fmt.Errorf("locale-bundle: environment-lock: %w", err)
	}
	if bundle.Environment != string(env.ID) || bundle.LockFingerprint != fingerprint {
		return fmt.Errorf("locale-bundle: environment-lock-mismatch")
	}
	found := false
	for _, identity := range resolved.Plugins {
		if string(identity.ID) == bundle.Target.ID && identity.Version == bundle.Target.Version {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("locale-bundle: target-not-in-current-lock")
	}
	return nil
}
