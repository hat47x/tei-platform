// Package inputprofile provides opt-in, versioned input admission before the
// existing deterministic TEI transform engine. It does not change the shipped
// SB-1 Source, the engine instruction set, or the BuildPlan fingerprint.
package inputprofile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

// PluginDefinitionJSONV2 is an opt-in ingress profile. The historical v1
// engine must never apply this normalization implicitly.
const PluginDefinitionJSONV2 = "tei.plugin-definition-json/v2"

// Evidence records the exact input, normalized input, and authoritative Source
// identities. It is an observation, not an authorization or an artifact lock.
type Evidence struct {
	Profile              string `json:"profile"`
	SourceFingerprint    string `json:"sourceFingerprint"`
	OriginalInputDigest  string `json:"originalInputDigest"`
	CanonicalInputDigest string `json:"canonicalInputDigest"`
}

func digest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// checkExactMemberNames keeps the opt-in adapter fail-closed: Go's JSON
// decoder accepts case-insensitive aliases ("ID" for "id") by default,
// which can otherwise overwrite a valid sibling field without a duplicate
// key diagnostic. The shape check uses the declared Go JSON tags, including
// nested objects and arrays, before any struct-based normalization.
func checkExactMemberNames(value transform.Value, schema reflect.Type, path string) error {
	for schema.Kind() == reflect.Pointer {
		schema = schema.Elem()
	}
	if value.Kind == transform.ValueNull {
		return fmt.Errorf("null value is not allowed at %s", path)
	}
	switch schema.Kind() {
	case reflect.Struct:
		if value.Kind != transform.ValueObject {
			return fmt.Errorf("%s requires an object", path)
		}
		fields := map[string]reflect.Type{}
		for i := 0; i < schema.NumField(); i++ {
			field := schema.Field(i)
			if !field.IsExported() {
				continue
			}
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = field.Name
			}
			fields[name] = field.Type
		}
		for _, field := range value.Object {
			memberType, ok := fields[field.Name]
			if !ok {
				return fmt.Errorf("%s has unknown or noncanonical JSON field %q", path, field.Name)
			}
			if err := checkExactMemberNames(field.Value, memberType, path+"."+field.Name); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		if value.Kind != transform.ValueArray {
			return fmt.Errorf("%s requires an array", path)
		}
		for i, item := range value.Array {
			if err := checkExactMemberNames(item, schema.Elem(), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

// CanonicalizePluginDefinition is an explicit host-side adapter for the exact
// shipped SB-1 package Source. Typed, strict JSON decoding gives the same
// member order and omitempty representation as Go's Definition serializer.
// The adapter is NOT invoked by transform.Evaluate or any legacy caller.
// A caller must opt in to the v2 ingress profile, supply a trusted locked\n// Source fingerprint, and record returned Evidence.
//
// Changing the Source identity requires an explicit qualification of a new
// adapter/source pair, not a silent fallback to another builder.
func CanonicalizePluginDefinition(profile string, source transform.Source, expectedSourceFingerprint string, raw []byte) (transform.Value, Evidence, error) {
	if profile != PluginDefinitionJSONV2 {
		return transform.Value{}, Evidence{}, fmt.Errorf("unsupported plugin definition input profile %q", profile)
	}
	if source.FormatVersion != transform.SourceFormatVersion ||
		source.Profile != transform.ProfileV1 ||
		string(source.Identity.ID) != "tei.builder.plugin_package" ||
		source.Identity.Version != "1.0.0" ||
		source.Identity.ContractVersion != "tei.plugin-transform/v1" {
		return transform.Value{}, Evidence{}, fmt.Errorf("input profile %q requires exact SB-1 package Source identity", profile)
	}
	if err := source.Validate(); err != nil {
		return transform.Value{}, Evidence{}, fmt.Errorf("invalid bound package Source: %w", err)
	}
	sourceFingerprint, err := transform.SourceFingerprint(source)
	if err != nil {
		return transform.Value{}, Evidence{}, fmt.Errorf("fingerprint bound package Source: %w", err)
	}
	if expectedSourceFingerprint == "" || sourceFingerprint != expectedSourceFingerprint {
		return transform.Value{}, Evidence{}, fmt.Errorf("input profile %q requires exact locked SB-1 Source fingerprint", profile)
	}
	if len(raw) > source.Limits.MaxInputBytes {
		return transform.Value{}, Evidence{}, fmt.Errorf("plugin definition JSON exceeds Source maxInputBytes")
	}

	// Reject duplicate keys (including escaped aliases), invalid Unicode,
	// excessive JSON nesting, and trailing data before decoding typed fields.
	input, err := transform.ParseJSON(raw)
	if err != nil {
		return transform.Value{}, Evidence{}, fmt.Errorf("plugin definition input: %w", err)
	}
	if err := checkExactMemberNames(input, reflect.TypeOf(plugin.Definition{}), "pluginDefinition"); err != nil {
		return transform.Value{}, Evidence{}, fmt.Errorf("plugin definition input: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var definition plugin.Definition
	if err := decoder.Decode(&definition); err != nil {
		return transform.Value{}, Evidence{}, fmt.Errorf("plugin definition schema: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return transform.Value{}, Evidence{}, fmt.Errorf("plugin definition JSON has trailing data")
		}
		return transform.Value{}, Evidence{}, fmt.Errorf("plugin definition JSON has trailing data: %w", err)
	}

	canonical, err := json.Marshal(definition)
	if err != nil {
		return transform.Value{}, Evidence{}, fmt.Errorf("marshal canonical plugin definition: %w", err)
	}
	if len(canonical) > source.Limits.MaxInputBytes {
		return transform.Value{}, Evidence{}, fmt.Errorf("canonical plugin definition exceeds Source maxInputBytes")
	}
	value, err := transform.ParseJSON(canonical)
	if err != nil {
		return transform.Value{}, Evidence{}, fmt.Errorf("parse canonical plugin definition: %w", err)
	}
	return value, Evidence{
		Profile:              profile,
		SourceFingerprint:    sourceFingerprint,
		OriginalInputDigest:  digest(raw),
		CanonicalInputDigest: digest(canonical),
	}, nil
}
