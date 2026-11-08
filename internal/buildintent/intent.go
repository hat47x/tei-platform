// Package buildintent holds the Build Intent declaration as a type and its
// strict reader, so both the reference tool in tools/ and cmd/tei can read the
// same declaration with one implementation.
//
// SB-C0 §4.13 selected this move (I-1a) for one reason: the CLI could not read
// a declaration at all, so an intent fingerprint was only ever a value a child
// reported. Reading it here lets the CLI hold its own reading.
//
// This package does not resolve a declaration into a BuildPlan. Resolution
// needs a ResolutionContext built from a fixture catalog, and moving that would
// make the reference fixture look like a product surface (§4.13 case I-1b,
// declined).
package buildintent

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/plugin"
)

const (
	// FormatVersion is the authoring format this package reads.
	FormatVersion = 1
	// ResolverVersion identifies the derivation rules the reference resolver
	// applies. It is recorded in a resolution receipt, not used here.
	ResolverVersion = "tei.build-intent-resolver/reference-v1"

	subjectKindPlugin    = "plugin"
	sourceKindDefinition = "plugin-definition"
)

// BuildIntent is the declaration a person or a generator writes. It carries no
// resolved digest or fingerprint: source bytes, the builder entity and
// dependency artifacts are derived, not authored.
type BuildIntent struct {
	FormatVersion      int                     `json:"formatVersion"`
	Subject            IntentSubject           `json:"subject"`
	Sources            []IntentSource          `json:"sources"`
	Builder            BuilderRequirement      `json:"builder"`
	Recipe             plugin.BuildRecipe      `json:"recipe"`
	OutputKind         plugin.BuildOutputKind  `json:"outputKind"`
	Dependencies       []DependencyRequirement `json:"dependencies,omitempty"`
	Target             string                  `json:"target"`
	Profile            string                  `json:"profile,omitempty"`
	Parameters         []plugin.BuildParameter `json:"parameters,omitempty"`
	SecretRequirements []SecretRequirement     `json:"secretRequirements,omitempty"`
}

// IntentSubject identifies what is being built. DefinitionSource names a
// plugin-definition source inside Sources; the definition fingerprint itself is
// never written by hand.
type IntentSubject struct {
	Kind             string           `json:"kind"`
	ID               core.CanonicalID `json:"id"`
	Version          string           `json:"version"`
	DefinitionSource string           `json:"definitionSource"`
}

// IntentSource is a logical source reference. It holds no content digest.
type IntentSource struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Ref  string `json:"ref"`
}

// BuilderRequirement is a selection policy, not an exact artifact.
type BuilderRequirement struct {
	Contract        string          `json:"contract"`
	CompatibleMajor int             `json:"compatibleMajor"`
	Provider        *ProviderWiring `json:"provider,omitempty"`
	Pin             *ArtifactPin    `json:"pin,omitempty"`
}

// ProviderWiring is an explicit logical provider selection.
type ProviderWiring struct {
	Plugin         core.CanonicalID `json:"plugin"`
	Implementation string           `json:"implementation,omitempty"`
}

// ArtifactPin is an exact pin a person writes on purpose. It is distinct from a
// value the resolver observed; a mismatch stops resolution.
type ArtifactPin struct {
	ArtifactDigest string `json:"artifactDigest"`
}

// DependencyRequirement is a logical build dependency. The exact artifact
// digest is derived from the dependency catalog.
type DependencyRequirement struct {
	Kind            string `json:"kind"`
	ID              string `json:"id"`
	CompatibleMajor int    `json:"compatibleMajor"`
}

// SecretRequirement declares a secret's name and the authority that provides
// it. It never carries the value.
type SecretRequirement struct {
	Name      string `json:"name"`
	Authority string `json:"authority"`
}

// derivedKeys are values that must not be authored. Comparison ignores case.
var derivedKeys = map[string]struct{}{
	"digest":                       {},
	"artifactdigest":               {},
	"sourcedigest":                 {},
	"closuredigest":                {},
	"fingerprint":                  {},
	"definitionfingerprint":        {},
	"plugindefinitionfingerprint":  {},
	"planfingerprint":              {},
	"buildplanfingerprint":         {},
	"intentfingerprint":            {},
	"resolutioncontextfingerprint": {},
	"plan":                         {},
	"receipt":                      {},
	"locator":                      {},
	"evidence":                     {},
	"acceptance":                   {},
	"runtimestate":                 {},
}

// secretLikeName stops a parameter name from carrying a secret value.
var secretLikeName = regexp.MustCompile(`(?i)(secret|password|passwd|token|credential|private[-_ ]?key|signing[-_ ]?key|api[-_ ]?key)`)

var sha256Digest = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// IsSHA256Digest reports whether value has the sha256:<64 lowercase hex> shape
// the declaration accepts for an explicit pin.
func IsSHA256Digest(value string) bool {
	return sha256Digest.MatchString(value)
}

// ParseIntent reads authoring JSON strictly: unknown fields, hand-written
// derived values, case-variant field names and trailing content are refused.
// Field order does not affect the result.
func ParseIntent(raw []byte) (BuildIntent, error) {
	if err := ValidateInputJSON(raw); err != nil {
		return BuildIntent{}, fmt.Errorf("build intent: %w", err)
	}
	var generic any
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&generic); err != nil {
		return BuildIntent{}, fmt.Errorf("build intent is not valid JSON: %w", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return BuildIntent{}, fmt.Errorf("build intent has trailing data")
	}
	if err := rejectDerivedKeys(generic, nil); err != nil {
		return BuildIntent{}, err
	}
	if err := RejectKeyCaseVariants(generic, reflect.TypeOf(BuildIntent{}), "$"); err != nil {
		return BuildIntent{}, err
	}
	var intent BuildIntent
	strict := json.NewDecoder(bytes.NewReader(raw))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&intent); err != nil {
		return BuildIntent{}, fmt.Errorf("build intent: %w", err)
	}
	if err := intent.Validate(); err != nil {
		return BuildIntent{}, err
	}
	return intent, nil
}

func rejectDerivedKeys(v any, trail []string) error {
	switch node := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(node))
		for k := range node {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			// builder.pin.artifactDigest is the one derived-looking field a
			// person writes on purpose.
			if _, derived := derivedKeys[strings.ToLower(k)]; derived && !isIntendedPin(trail, k) {
				return fmt.Errorf(
					"derived field %q must not be authored (at %s); remove it and let the resolver derive it",
					k, strings.Join(append(append([]string(nil), trail...), k), "."),
				)
			}
			if err := rejectDerivedKeys(node[k], append(append([]string(nil), trail...), k)); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range node {
			if err := rejectDerivedKeys(item, append(append([]string(nil), trail...), "[]")); err != nil {
				return err
			}
		}
	}
	return nil
}

func isIntendedPin(trail []string, key string) bool {
	return len(trail) == 2 && trail[0] == "builder" && trail[1] == "pin" && key == "artifactDigest"
}

// Validate checks the declaration's shape only. It reads no external state.
func (i BuildIntent) Validate() error {
	if i.FormatVersion != FormatVersion {
		return fmt.Errorf("unsupported build intent format %d", i.FormatVersion)
	}
	if i.Subject.Kind != subjectKindPlugin {
		return fmt.Errorf("build intent subject kind %q is not supported by this Reference (only %q)", i.Subject.Kind, subjectKindPlugin)
	}
	if _, err := core.ParseCanonicalID(string(i.Subject.ID)); err != nil {
		return fmt.Errorf("build intent subject id: %w", err)
	}
	if !core.ValidNumericVersion(i.Subject.Version) {
		return fmt.Errorf("build intent subject version %q must be major.minor.patch", i.Subject.Version)
	}
	if len(i.Sources) == 0 {
		return fmt.Errorf("build intent requires at least one source")
	}
	sourceIDs := map[string]IntentSource{}
	for _, s := range i.Sources {
		if strings.TrimSpace(s.Kind) == "" || strings.TrimSpace(s.ID) == "" {
			return fmt.Errorf("build intent source kind and id are required")
		}
		if _, dup := sourceIDs[s.ID]; dup {
			return fmt.Errorf("build intent source %q is declared more than once", s.ID)
		}
		if err := ValidateSourceRef(s.Ref); err != nil {
			return fmt.Errorf("build intent source %q: %w", s.ID, err)
		}
		sourceIDs[s.ID] = s
	}
	definition, ok := sourceIDs[i.Subject.DefinitionSource]
	if !ok {
		return fmt.Errorf("build intent definitionSource %q is not a declared source", i.Subject.DefinitionSource)
	}
	if definition.Kind != sourceKindDefinition {
		return fmt.Errorf("build intent definitionSource %q must have kind %s", definition.ID, sourceKindDefinition)
	}
	if strings.TrimSpace(i.Builder.Contract) == "" || i.Builder.CompatibleMajor <= 0 {
		return fmt.Errorf("build intent builder contract and compatibleMajor are required")
	}
	if i.Builder.Provider != nil {
		if _, err := core.ParseCanonicalID(string(i.Builder.Provider.Plugin)); err != nil {
			return fmt.Errorf("build intent builder provider plugin: %w", err)
		}
	}
	if i.Builder.Pin != nil && !sha256Digest.MatchString(i.Builder.Pin.ArtifactDigest) {
		return fmt.Errorf("build intent builder pin must use sha256:<64 lowercase hex>")
	}
	if strings.TrimSpace(i.Recipe.Kind) == "" || strings.TrimSpace(i.Recipe.Entrypoint) == "" {
		return fmt.Errorf("build intent recipe kind and entrypoint are required")
	}
	if strings.TrimSpace(string(i.OutputKind)) == "" {
		return fmt.Errorf("build intent output kind is required")
	}
	if strings.TrimSpace(i.Target) == "" {
		return fmt.Errorf("build intent target is required")
	}
	dependencies := map[string]struct{}{}
	for _, d := range i.Dependencies {
		if strings.TrimSpace(d.Kind) == "" || strings.TrimSpace(d.ID) == "" || d.CompatibleMajor < 0 {
			return fmt.Errorf("build intent dependency kind, id and non-negative compatibleMajor are required")
		}
		key := d.Kind + "|" + d.ID
		if _, dup := dependencies[key]; dup {
			return fmt.Errorf("build intent dependency %s is declared more than once", key)
		}
		dependencies[key] = struct{}{}
	}
	parameters := map[string]struct{}{}
	for _, p := range i.Parameters {
		if strings.TrimSpace(p.Name) == "" {
			return fmt.Errorf("build intent parameter name is required")
		}
		if secretLikeName.MatchString(p.Name) {
			return fmt.Errorf("build intent parameter %q looks like a secret; declare it in secretRequirements and materialize the value from an external Authority", p.Name)
		}
		if _, dup := parameters[p.Name]; dup {
			return fmt.Errorf("build intent parameter %q is declared more than once", p.Name)
		}
		parameters[p.Name] = struct{}{}
	}
	secrets := map[string]struct{}{}
	for _, s := range i.SecretRequirements {
		if strings.TrimSpace(s.Name) == "" || strings.TrimSpace(s.Authority) == "" {
			return fmt.Errorf("build intent secret requirement name and authority are required")
		}
		if _, dup := secrets[s.Name]; dup {
			return fmt.Errorf("build intent secret requirement %q is declared more than once", s.Name)
		}
		secrets[s.Name] = struct{}{}
	}
	return nil
}

// ValidateSourceRef refuses a reference that is not a clean relative slash path.
func ValidateSourceRef(ref string) error {
	if strings.TrimSpace(ref) == "" {
		return fmt.Errorf("source ref is required")
	}
	if strings.Contains(ref, `\`) || strings.HasPrefix(ref, "/") {
		return fmt.Errorf("source ref %q must be a relative slash path", ref)
	}
	// Volume names (C:/, c:) and alternate data streams are refused, and so are
	// control characters, whose interpretation differs per host.
	for _, r := range ref {
		if r == ':' {
			return fmt.Errorf("source ref %q must not contain ':' (volume names are not allowed)", ref)
		}
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("source ref %q must not contain control characters", ref)
		}
	}
	if path.Clean(ref) != ref || ref == "." || ref == ".." || strings.HasPrefix(ref, "../") {
		return fmt.Errorf("source ref %q must be a clean path inside the source root", ref)
	}
	return nil
}

// NormalizeIntent removes declaration order from the result.
func NormalizeIntent(i BuildIntent) BuildIntent {
	out := i
	out.Sources = append([]IntentSource(nil), i.Sources...)
	out.Dependencies = append([]DependencyRequirement(nil), i.Dependencies...)
	out.Parameters = append([]plugin.BuildParameter(nil), i.Parameters...)
	out.SecretRequirements = append([]SecretRequirement(nil), i.SecretRequirements...)
	sort.Slice(out.Sources, func(a, b int) bool { return out.Sources[a].ID < out.Sources[b].ID })
	sort.Slice(out.Dependencies, func(a, b int) bool {
		if out.Dependencies[a].Kind != out.Dependencies[b].Kind {
			return out.Dependencies[a].Kind < out.Dependencies[b].Kind
		}
		return out.Dependencies[a].ID < out.Dependencies[b].ID
	})
	sort.Slice(out.Parameters, func(a, b int) bool { return out.Parameters[a].Name < out.Parameters[b].Name })
	sort.Slice(out.SecretRequirements, func(a, b int) bool { return out.SecretRequirements[a].Name < out.SecretRequirements[b].Name })
	return out
}

// IntentFingerprint is the declaration's own identity. It covers neither source
// bytes nor catalog state.
func IntentFingerprint(i BuildIntent) (string, error) {
	if err := i.Validate(); err != nil {
		return "", err
	}
	return FingerprintJSON(NormalizeIntent(i))
}

// FingerprintJSON is the shared normalised-JSON digest used by the declaration
// and by the reference catalogs.
func FingerprintJSON(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
