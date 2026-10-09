package transform

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
)

const (
	SourceFormatVersion = 1
	ProfileV1           = "deterministic-transform-v1"
)

type Op string

const (
	OpInput           Op = "input"
	OpLiteral         Op = "literal"
	OpSelectField     Op = "select-field"
	OpConstructObject Op = "construct-object"
	OpMapArray        Op = "map-array"
	OpSortArrayBy     Op = "sort-array-by"
	OpReplaceField    Op = "replace-field"
	OpOmitField       Op = "omit-field"
	OpAssertEqual     Op = "assert-equal"
	OpAssertEnum      Op = "assert-enum"
	OpCanonicalJSON   Op = "canonical-json"
	OpSHA256          Op = "sha256"
	OpAppendBytes     Op = "append-bytes"
	OpJoinStrings     Op = "join-strings"
)

type InputKind string

const (
	InputObject InputKind = "object"
	InputArray  InputKind = "array"
	InputString InputKind = "string"
	InputBytes  InputKind = "bytes"
)

type SourceIdentity struct {
	ID              core.CanonicalID `json:"id"`
	Version         string           `json:"version"`
	ContractVersion string           `json:"contractVersion"`
}

type Limits struct {
	MaxSteps        int `json:"maxSteps"`
	MaxCollection   int `json:"maxCollection"`
	MaxOutputBytes  int `json:"maxOutputBytes"`
	MaxNestingDepth int `json:"maxNestingDepth"`
}

type InputSpec struct {
	Name string    `json:"name"`
	Kind InputKind `json:"kind"`
}

type ObjectFieldExpr struct {
	Name  string `json:"name"`
	Value Expr   `json:"value"`
}

// Expr is a closed structured transform IR. Fields irrelevant to Op must remain
// empty; validation rejects malformed combinations before evaluation.
type Expr struct {
	Op Op `json:"op"`

	Name     string          `json:"name,omitempty"`
	Literal  json.RawMessage `json:"literal,omitempty"`
	From     *Expr           `json:"from,omitempty"`
	Field    string          `json:"field,omitempty"`
	Optional bool            `json:"optional,omitempty"`

	Fields []ObjectFieldExpr `json:"fields,omitempty"`

	Item string `json:"item,omitempty"`
	Body *Expr  `json:"body,omitempty"`
	Keys []Expr `json:"keys,omitempty"`

	Value *Expr `json:"value,omitempty"`

	Actual   *Expr             `json:"actual,omitempty"`
	Expected *Expr             `json:"expected,omitempty"`
	Allowed  []json.RawMessage `json:"allowed,omitempty"`

	Values    []Expr `json:"values,omitempty"`
	Separator string `json:"separator,omitempty"`

	Suffix string `json:"suffix,omitempty"`
}

type Program struct {
	Assertions []Expr `json:"assertions,omitempty"`
	Output     Expr   `json:"output"`
}

type Source struct {
	FormatVersion int            `json:"formatVersion"`
	Identity      SourceIdentity `json:"identity"`
	Profile       string         `json:"profile"`
	Limits        Limits         `json:"limits"`
	Inputs        []InputSpec    `json:"inputs"`
	Program       Program        `json:"program"`
}

func (s Source) Validate() error {
	if s.FormatVersion != SourceFormatVersion {
		return fmt.Errorf("unsupported transform source format %d", s.FormatVersion)
	}
	if _, err := core.ParseCanonicalID(string(s.Identity.ID)); err != nil {
		return fmt.Errorf("transform source id: %w", err)
	}
	if !core.ValidNumericVersion(s.Identity.Version) {
		return fmt.Errorf("transform source version %q must be major.minor.patch", s.Identity.Version)
	}
	if strings.TrimSpace(s.Identity.ContractVersion) == "" {
		return fmt.Errorf("transform source contract version is required")
	}
	if s.Profile != ProfileV1 {
		return fmt.Errorf("unsupported transform profile %q", s.Profile)
	}
	if s.Limits.MaxSteps <= 0 || s.Limits.MaxSteps > 1_000_000 {
		return fmt.Errorf("transform maxSteps must be between 1 and 1000000")
	}
	if s.Limits.MaxCollection <= 0 || s.Limits.MaxCollection > 100_000 {
		return fmt.Errorf("transform maxCollection must be between 1 and 100000")
	}
	if s.Limits.MaxOutputBytes <= 0 || s.Limits.MaxOutputBytes > 64<<20 {
		return fmt.Errorf("transform maxOutputBytes must be between 1 and 67108864")
	}
	if s.Limits.MaxNestingDepth <= 0 || s.Limits.MaxNestingDepth > 128 {
		return fmt.Errorf("transform maxNestingDepth must be between 1 and 128")
	}
	inputs := map[string]struct{}{}
	for _, input := range s.Inputs {
		if strings.TrimSpace(input.Name) == "" {
			return fmt.Errorf("transform input name is required")
		}
		if _, exists := inputs[input.Name]; exists {
			return fmt.Errorf("transform input %q is declared more than once", input.Name)
		}
		inputs[input.Name] = struct{}{}
		switch input.Kind {
		case InputObject, InputArray, InputString, InputBytes:
		default:
			return fmt.Errorf("transform input %q has unsupported kind %q", input.Name, input.Kind)
		}
	}
	if len(s.Inputs) == 0 {
		return fmt.Errorf("transform source requires at least one input")
	}
	for i := range s.Program.Assertions {
		if err := validateExpr(s.Program.Assertions[i], 0); err != nil {
			return fmt.Errorf("transform assertion %d: %w", i, err)
		}
		switch s.Program.Assertions[i].Op {
		case OpAssertEqual, OpAssertEnum:
		default:
			return fmt.Errorf("transform assertion %d must use assert operation, got %s", i, s.Program.Assertions[i].Op)
		}
	}
	if err := validateExpr(s.Program.Output, 0); err != nil {
		return fmt.Errorf("transform output: %w", err)
	}
	return nil
}

func validateExpr(expr Expr, depth int) error {
	if depth > 128 {
		return fmt.Errorf("transform expression nesting exceeds 128 levels")
	}
	child := func(e *Expr, name string) error {
		if e == nil {
			return fmt.Errorf("%s expression is required", name)
		}
		return validateExpr(*e, depth+1)
	}
	switch expr.Op {
	case OpInput:
		if strings.TrimSpace(expr.Name) == "" {
			return fmt.Errorf("input name is required")
		}
	case OpLiteral:
		if len(bytes.TrimSpace(expr.Literal)) == 0 {
			return fmt.Errorf("literal JSON is required")
		}
		if _, err := ParseJSON(expr.Literal); err != nil {
			return fmt.Errorf("literal JSON: %w", err)
		}
	case OpSelectField:
		if err := child(expr.From, "select-field source"); err != nil {
			return err
		}
		if strings.TrimSpace(expr.Field) == "" {
			return fmt.Errorf("select-field field is required")
		}
	case OpConstructObject:
		if len(expr.Fields) == 0 {
			return fmt.Errorf("construct-object requires at least one field")
		}
		seen := map[string]struct{}{}
		for i := range expr.Fields {
			field := expr.Fields[i]
			if strings.TrimSpace(field.Name) == "" {
				return fmt.Errorf("construct-object field name is required")
			}
			if _, exists := seen[field.Name]; exists {
				return fmt.Errorf("construct-object field %q is declared more than once", field.Name)
			}
			seen[field.Name] = struct{}{}
			if err := validateExpr(field.Value, depth+1); err != nil {
				return fmt.Errorf("construct-object field %q: %w", field.Name, err)
			}
		}
	case OpMapArray:
		if err := child(expr.From, "map-array source"); err != nil {
			return err
		}
		if strings.TrimSpace(expr.Item) == "" {
			return fmt.Errorf("map-array item name is required")
		}
		if err := child(expr.Body, "map-array body"); err != nil {
			return err
		}
	case OpSortArrayBy:
		if err := child(expr.From, "sort-array source"); err != nil {
			return err
		}
		if strings.TrimSpace(expr.Item) == "" {
			return fmt.Errorf("sort-array item name is required")
		}
		if len(expr.Keys) == 0 {
			return fmt.Errorf("sort-array requires at least one key")
		}
		for i := range expr.Keys {
			if err := validateExpr(expr.Keys[i], depth+1); err != nil {
				return fmt.Errorf("sort-array key %d: %w", i, err)
			}
		}
	case OpReplaceField:
		if err := child(expr.From, "replace-field source"); err != nil {
			return err
		}
		if strings.TrimSpace(expr.Field) == "" {
			return fmt.Errorf("replace-field field is required")
		}
		if err := child(expr.Value, "replace-field value"); err != nil {
			return err
		}
	case OpOmitField:
		if err := child(expr.From, "omit-field source"); err != nil {
			return err
		}
		if strings.TrimSpace(expr.Field) == "" {
			return fmt.Errorf("omit-field field is required")
		}
	case OpAssertEqual:
		if err := child(expr.Actual, "assert-equal actual"); err != nil {
			return err
		}
		if err := child(expr.Expected, "assert-equal expected"); err != nil {
			return err
		}
	case OpAssertEnum:
		if err := child(expr.Actual, "assert-enum actual"); err != nil {
			return err
		}
		if len(expr.Allowed) == 0 {
			return fmt.Errorf("assert-enum requires allowed values")
		}
		for _, allowed := range expr.Allowed {
			if _, err := ParseJSON(allowed); err != nil {
				return fmt.Errorf("assert-enum value: %w", err)
			}
		}
	case OpCanonicalJSON, OpSHA256:
		if err := child(expr.From, string(expr.Op)+" source"); err != nil {
			return err
		}
	case OpAppendBytes:
		if err := child(expr.From, "append-bytes source"); err != nil {
			return err
		}
	case OpJoinStrings:
		if len(expr.Values) == 0 {
			return fmt.Errorf("join-strings requires values")
		}
		for i := range expr.Values {
			if err := validateExpr(expr.Values[i], depth+1); err != nil {
				return fmt.Errorf("join-strings value %d: %w", i, err)
			}
		}
	default:
		return fmt.Errorf("unsupported transform operation %q", expr.Op)
	}
	return nil
}

func NormalizeSource(source Source) (Source, error) {
	if err := source.Validate(); err != nil {
		return Source{}, err
	}
	out := source
	out.Inputs = append([]InputSpec(nil), source.Inputs...)
	out.Program.Assertions = make([]Expr, len(source.Program.Assertions))
	for i := range source.Program.Assertions {
		normalized, err := normalizeExpr(source.Program.Assertions[i])
		if err != nil {
			return Source{}, err
		}
		out.Program.Assertions[i] = normalized
	}
	normalizedOutput, err := normalizeExpr(source.Program.Output)
	if err != nil {
		return Source{}, err
	}
	out.Program.Output = normalizedOutput
	return out, nil
}

func normalizeExpr(expr Expr) (Expr, error) {
	out := expr
	if len(expr.Literal) > 0 {
		value, err := ParseJSON(expr.Literal)
		if err != nil {
			return Expr{}, err
		}
		raw, err := value.JSON()
		if err != nil {
			return Expr{}, err
		}
		out.Literal = raw
	}
	if expr.From != nil {
		normalized, err := normalizeExpr(*expr.From)
		if err != nil {
			return Expr{}, err
		}
		out.From = &normalized
	}
	if expr.Body != nil {
		normalized, err := normalizeExpr(*expr.Body)
		if err != nil {
			return Expr{}, err
		}
		out.Body = &normalized
	}
	if expr.Value != nil {
		normalized, err := normalizeExpr(*expr.Value)
		if err != nil {
			return Expr{}, err
		}
		out.Value = &normalized
	}
	if expr.Actual != nil {
		normalized, err := normalizeExpr(*expr.Actual)
		if err != nil {
			return Expr{}, err
		}
		out.Actual = &normalized
	}
	if expr.Expected != nil {
		normalized, err := normalizeExpr(*expr.Expected)
		if err != nil {
			return Expr{}, err
		}
		out.Expected = &normalized
	}
	out.Fields = make([]ObjectFieldExpr, len(expr.Fields))
	for i := range expr.Fields {
		normalized, err := normalizeExpr(expr.Fields[i].Value)
		if err != nil {
			return Expr{}, err
		}
		out.Fields[i] = ObjectFieldExpr{Name: expr.Fields[i].Name, Value: normalized}
	}
	out.Keys = make([]Expr, len(expr.Keys))
	for i := range expr.Keys {
		normalized, err := normalizeExpr(expr.Keys[i])
		if err != nil {
			return Expr{}, err
		}
		out.Keys[i] = normalized
	}
	out.Values = make([]Expr, len(expr.Values))
	for i := range expr.Values {
		normalized, err := normalizeExpr(expr.Values[i])
		if err != nil {
			return Expr{}, err
		}
		out.Values[i] = normalized
	}
	if len(expr.Allowed) > 0 {
		out.Allowed = make([]json.RawMessage, len(expr.Allowed))
		for i, raw := range expr.Allowed {
			value, err := ParseJSON(raw)
			if err != nil {
				return Expr{}, err
			}
			canonical, err := value.JSON()
			if err != nil {
				return Expr{}, err
			}
			out.Allowed[i] = canonical
		}
		sort.Slice(out.Allowed, func(i, j int) bool {
			return bytes.Compare(out.Allowed[i], out.Allowed[j]) < 0
		})
	}
	return out, nil
}

func SourceFingerprint(source Source) (string, error) {
	normalized, err := NormalizeSource(source)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func MaterializeSource(source Source) ([]byte, string, error) {
	normalized, err := NormalizeSource(source)
	if err != nil {
		return nil, "", err
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return nil, "", err
	}
	raw = append(raw, '\n')
	sum := sha256.Sum256(raw)
	return raw, "sha256:" + hex.EncodeToString(sum[:]), nil
}
