// Package standardmessages defines a language-neutral, read-only reference
// contract for std.message. Translation, persistence, and formatting belong
// to other plugins and do not affect a message's stable identity.
package standardmessages

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/text/language"
	"tei/internal/core"
	"tei/internal/plugin"
)

const (
	FormatPlain   = "plain"
	FormatNamedV0 = "named-v0"
)

type ParameterKind string

const (
	ParameterText    ParameterKind = "text"
	ParameterInteger ParameterKind = "integer"
	ParameterDecimal ParameterKind = "decimal"
	ParameterBoolean ParameterKind = "boolean"
)

type Parameter struct {
	Name string        `json:"name"`
	Kind ParameterKind `json:"kind"`
}

type Ref struct {
	Owner plugin.Identity  `json:"owner"`
	ID    core.CanonicalID `json:"id"`
}

// Definition's Text is source copy. Its revision does not imply acceptance.
// The initial contract supports literal text or simple named placeholders;
// neither mode is MessageFormat 2, and neither performs formatting.
type Definition struct {
	Ref          Ref         `json:"ref"`
	SourceLocale string      `json:"sourceLocale"`
	Text         string      `json:"text"`
	Format       string      `json:"format"`
	Parameters   []Parameter `json:"parameters,omitempty"`
	Critical     bool        `json:"critical,omitempty"`
}

type Catalog struct {
	Definitions []Definition `json:"definitions"`
}

func validName(value string) bool {
	if len(value) == 0 {
		return false
	}
	for i, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (i > 0 && ((r >= '0' && r <= '9') || r == '_')) {
			continue
		}
		return false
	}
	return true
}

func placeholders(text string) (map[string]bool, error) {
	found := map[string]bool{}
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '}':
			return nil, fmt.Errorf("unbalanced-placeholder")
		case '{':
			end := strings.IndexByte(text[i+1:], '}')
			if end < 0 {
				return nil, fmt.Errorf("unclosed-placeholder")
			}
			name := text[i+1 : i+1+end]
			if !validName(name) {
				return nil, fmt.Errorf("unsupported-placeholder %q", name)
			}
			found[name] = true
			i += end + 1
		}
	}
	return found, nil
}

// ValidateText ensures that a candidate text uses precisely the declarations
// of the base message. This does not parse or execute MF2 expressions.
func ValidateText(def Definition, text string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("empty-text")
	}
	if def.Format == FormatPlain {
		if len(def.Parameters) > 0 {
			return fmt.Errorf("plain-with-parameters")
		}
		return nil
	}
	if def.Format != FormatNamedV0 {
		return fmt.Errorf("unsupported-format %q", def.Format)
	}
	tokens, err := placeholders(text)
	if err != nil {
		return err
	}
	required := map[string]bool{}
	for _, p := range def.Parameters {
		required[p.Name] = true
	}
	for name := range tokens {
		if !required[name] {
			return fmt.Errorf("undeclared-placeholder %s", name)
		}
	}
	for name := range required {
		if !tokens[name] {
			return fmt.Errorf("missing-placeholder %s", name)
		}
	}
	return nil
}

func Validate(def Definition) error {
	if _, err := core.ParseCanonicalID(string(def.Ref.ID)); err != nil {
		return fmt.Errorf("message ID: %w", err)
	}
	if _, err := core.ParseCanonicalID(string(def.Ref.Owner.ID)); err != nil {
		return fmt.Errorf("owner: %w", err)
	}
	if _, err := plugin.ParseVersion(def.Ref.Owner.Version); err != nil {
		return fmt.Errorf("owner version: %w", err)
	}
	tag, err := language.Parse(def.SourceLocale)
	if err != nil || tag.String() != def.SourceLocale || def.SourceLocale == "und" {
		return fmt.Errorf("invalid-source-locale")
	}
	seen := map[string]bool{}
	for _, p := range def.Parameters {
		if !validName(p.Name) || seen[p.Name] {
			return fmt.Errorf("duplicate-or-invalid-parameter %q", p.Name)
		}
		seen[p.Name] = true
		switch p.Kind {
		case ParameterText, ParameterInteger, ParameterDecimal, ParameterBoolean:
		default:
			return fmt.Errorf("unsupported-parameter-type")
		}
	}
	return ValidateText(def, def.Text)
}

func (c Catalog) Validate() error {
	seen := map[Ref]bool{}
	for _, def := range c.Definitions {
		if err := Validate(def); err != nil {
			return fmt.Errorf("%s: %w", def.Ref.ID, err)
		}
		if seen[def.Ref] {
			return fmt.Errorf("duplicate-message %s@%s/%s", def.Ref.Owner.ID, def.Ref.Owner.Version, def.Ref.ID)
		}
		seen[def.Ref] = true
	}
	return nil
}

// Lookup never invents a missing message and returns defensive copies.
func (c Catalog) Lookup(ref Ref) (Definition, error) {
	if err := c.Validate(); err != nil {
		return Definition{}, err
	}
	for _, d := range c.Definitions {
		if d.Ref == ref {
			d.Parameters = append([]Parameter(nil), d.Parameters...)
			return d, nil
		}
	}
	return Definition{}, fmt.Errorf("missing-message %s", ref.ID)
}

// Revision binds source wording, placeholder signature and criticality. A
// parameter type change invalidates a translated text even if text is equal.
func Revision(def Definition) string {
	copy := def
	copy.Parameters = append([]Parameter(nil), def.Parameters...)
	sort.Slice(copy.Parameters, func(i, j int) bool { return copy.Parameters[i].Name < copy.Parameters[j].Name })
	bytes, _ := json.Marshal(copy)
	sum := sha256.Sum256(bytes)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ValidateArguments only validates a caller-supplied type signature; values
// remain opaque until a separately selected, capable formatter is invoked.
func ValidateArguments(def Definition, supplied map[string]ParameterKind) error {
	if err := Validate(def); err != nil {
		return err
	}
	if len(supplied) != len(def.Parameters) {
		return fmt.Errorf("argument-count-mismatch")
	}
	for _, p := range def.Parameters {
		if supplied[p.Name] != p.Kind {
			return fmt.Errorf("argument-type-mismatch %s", p.Name)
		}
	}
	return nil
}
