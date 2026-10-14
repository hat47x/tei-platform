package main

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
	// IntentFormatVersion は人・生成AIが管理するBuild Intentの形式版である。
	IntentFormatVersion = 1
	// ResolverVersion は導出規則の版であり、ResolutionContextの一部になる。
	ResolverVersion = "tei.build-intent-resolver/reference-v1"

	subjectKindPlugin     = "plugin"
	sourceKindDefinition  = "plugin-definition"
	builderAuthorityScope = "build-tool"
)

// BuildIntent は人・生成AIが書く宣言である。解決済みのdigestやfingerprintは
// 持たない。Source bytes、Builder実体、依存成果物の識別はResolverが導出する。
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

// IntentSubject は何を構築するかの論理的な識別である。
// DefinitionSource はSources内のplugin-definition Sourceを指すIDであり、
// definition fingerprintそのものは書かない。
type IntentSubject struct {
	Kind             string           `json:"kind"`
	ID               core.CanonicalID `json:"id"`
	Version          string           `json:"version"`
	DefinitionSource string           `json:"definitionSource"`
}

// IntentSource は論理的なSource参照である。Source内容のdigestは持たない。
type IntentSource struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Ref  string `json:"ref"`
}

// BuilderRequirement は選択方針である。exact artifactではない。
type BuilderRequirement struct {
	Contract        string          `json:"contract"`
	CompatibleMajor int             `json:"compatibleMajor"`
	Provider        *ProviderWiring `json:"provider,omitempty"`
	Pin             *ArtifactPin    `json:"pin,omitempty"`
}

// ProviderWiring は明示的な論理Provider選択である。
type ProviderWiring struct {
	Plugin         core.CanonicalID `json:"plugin"`
	Implementation string           `json:"implementation,omitempty"`
}

// ArtifactPin は人が意図して書くexact pinである。Resolverが観測した値とは
// 区別し、一致しない場合は停止する。
type ArtifactPin struct {
	ArtifactDigest string `json:"artifactDigest"`
}

// DependencyRequirement はBuild依存の論理要求である。exact artifact digestは
// ResolverがBuild依存Catalogから導出する。
type DependencyRequirement struct {
	Kind            string `json:"kind"`
	ID              string `json:"id"`
	CompatibleMajor int    `json:"compatibleMajor"`
}

// SecretRequirement はsecretの名前と取得元Authorityだけを宣言する。値は持たない。
type SecretRequirement struct {
	Name      string `json:"name"`
	Authority string `json:"authority"`
}

// derivedKeys は通常のauthoring sourceへ書いてはならない導出値の名前である。
// 比較は大文字小文字を区別しない。
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

// secretLikeName はParameter名にsecretを埋め込む運用を止めるための保守的な検査である。
var secretLikeName = regexp.MustCompile(`(?i)(secret|password|passwd|token|credential|private[-_ ]?key|signing[-_ ]?key|api[-_ ]?key)`)

var sha256Digest = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// ParseIntent はauthoring JSONを厳密に読む。未知のfield、導出値の手書き、
// 末尾の余分なJSONを拒否する。field順は結果に影響しない。
func ParseIntent(raw []byte) (BuildIntent, error) {
	if err := validateInputJSON(raw); err != nil {
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
	if err := rejectKeyCaseVariants(generic, reflect.TypeOf(BuildIntent{}), "$"); err != nil {
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
			// builder.pin.artifactDigestだけは人が意図として書くpinである。
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

// rejectKeyCaseVariants は構造体のjson名と大文字小文字まで一致しないkeyを拒否する。
// 標準のdecodeは大文字小文字を区別せずに受け取るため、別綴りの同名keyが通る。
func rejectKeyCaseVariants(v any, t reflect.Type, path string) error {
	switch t.Kind() {
	case reflect.Pointer:
		return rejectKeyCaseVariants(v, t.Elem(), path)
	case reflect.Slice:
		list, ok := v.([]any)
		if !ok {
			return nil
		}
		for i, item := range list {
			if err := rejectKeyCaseVariants(item, t.Elem(), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil
	}
	node, ok := v.(map[string]any)
	if !ok || t.Kind() != reflect.Struct {
		return nil
	}
	fields := map[string]reflect.Type{}
	for i := 0; i < t.NumField(); i++ {
		name := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			fields[name] = t.Field(i).Type
		}
	}
	for key, child := range node {
		ft, exact := fields[key]
		if !exact {
			return fmt.Errorf("input JSON key %q at %s does not match a field name exactly (case variants and unknown keys are rejected)", key, path)
		}
		if err := rejectKeyCaseVariants(child, ft, path+"."+key); err != nil {
			return err
		}
	}
	return nil
}

func isIntendedPin(trail []string, key string) bool {
	return len(trail) == 2 && trail[0] == "builder" && trail[1] == "pin" && key == "artifactDigest"
}

// Validate は宣言の形だけを検査する。外部状態は参照しない。
func (i BuildIntent) Validate() error {
	if i.FormatVersion != IntentFormatVersion {
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
		if err := validateSourceRef(s.Ref); err != nil {
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

func validateSourceRef(ref string) error {
	if strings.TrimSpace(ref) == "" {
		return fmt.Errorf("source ref is required")
	}
	if strings.Contains(ref, `\`) || strings.HasPrefix(ref, "/") {
		return fmt.Errorf("source ref %q must be a relative slash path", ref)
	}
	// ボリューム名（C:/、c:）と代替データ流を避けるため、':'は使わせない。
	// 制御文字（NULを含む）もHostごとに解釈が分かれるため拒否する。
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

// NormalizeIntent は宣言順を結果から外す。
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

// IntentFingerprint は宣言そのもののidentityである。Source bytesや
// Catalogの状態は含めない。
func IntentFingerprint(i BuildIntent) (string, error) {
	if err := i.Validate(); err != nil {
		return "", err
	}
	return fingerprintJSON(NormalizeIntent(i))
}

func fingerprintJSON(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
